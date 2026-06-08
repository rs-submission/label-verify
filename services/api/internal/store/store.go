package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ttb/labelverify/internal/match"
	"github.com/ttb/labelverify/internal/verify"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func Connect(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return New(pool), nil
}

func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) SaveApplication(ctx context.Context, app verify.Application) error {
	row, err := applicationToRow(app)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO applications (
			application_id, brand, class_type, net_contents, abv, government_warning,
			name_address, foreign_blocks, declared_languages, brand_norm, class_type_norm,
			net_contents_norm, abv_norm, government_warning_norm, name_address_norm
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (application_id) DO UPDATE SET
			brand = EXCLUDED.brand,
			class_type = EXCLUDED.class_type,
			net_contents = EXCLUDED.net_contents,
			abv = EXCLUDED.abv,
			government_warning = EXCLUDED.government_warning,
			name_address = EXCLUDED.name_address,
			foreign_blocks = EXCLUDED.foreign_blocks,
			declared_languages = EXCLUDED.declared_languages,
			brand_norm = EXCLUDED.brand_norm,
			class_type_norm = EXCLUDED.class_type_norm,
			net_contents_norm = EXCLUDED.net_contents_norm,
			abv_norm = EXCLUDED.abv_norm,
			government_warning_norm = EXCLUDED.government_warning_norm,
			name_address_norm = EXCLUDED.name_address_norm
	`, row.ID, row.Brand, row.ClassType, row.NetContents, row.ABV, row.GovernmentWarning,
		row.NameAndAddress, row.ForeignBlocks, row.DeclaredLanguages, row.BrandNorm, row.ClassTypeNorm,
		row.NetContentsNorm, row.ABVNorm, row.GovernmentWarningNorm, row.NameAndAddressNorm)
	return err
}

func (s *Store) GetApplication(ctx context.Context, id string) (verify.Application, error) {
	var app verify.Application
	var foreignBlocks []byte
	err := s.pool.QueryRow(ctx, `
		SELECT application_id, brand, class_type, net_contents, abv, government_warning,
		       name_address, foreign_blocks, declared_languages
		FROM applications
		WHERE application_id = $1
	`, id).Scan(&app.ID, &app.Brand, &app.ClassType, &app.NetContents, &app.ABV,
		&app.GovernmentWarning, &app.NameAndAddress, &foreignBlocks, &app.DeclaredLanguages)
	if errors.Is(err, pgx.ErrNoRows) {
		return verify.Application{}, verify.ErrApplicationNotFound
	}
	if err != nil {
		return verify.Application{}, err
	}
	if err := json.Unmarshal(foreignBlocks, &app.ForeignBlocks); err != nil {
		return verify.Application{}, err
	}
	return app, nil
}

func (s *Store) ListApplications(ctx context.Context) ([]verify.ApplicationSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT application_id, brand, class_type
		FROM applications
		ORDER BY application_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]verify.ApplicationSummary, 0)
	for rows.Next() {
		var summary verify.ApplicationSummary
		if err := rows.Scan(&summary.ID, &summary.Brand, &summary.ClassType); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

func (s *Store) SaveVerification(ctx context.Context, result verify.StoredVerification) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var verificationID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO verification_results (application_id, image_ref, status, confidence)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, result.ApplicationID, result.ImageRef, result.Verdict.Status, result.Verdict.Confidence).Scan(&verificationID)
	if err != nil {
		return err
	}

	for _, field := range result.Verdict.Fields {
		if _, err := tx.Exec(ctx, `
			INSERT INTO field_results (
				verification_id, field_name, expected_value, extracted_value,
				match_type, score, pass, diff
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		`, verificationID, field.Field, field.Expected, field.Extracted,
			field.MatchType, field.Score, field.Pass, field.Diff); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *Store) DeleteApplication(ctx context.Context, id string) (verify.DeletedApplication, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return verify.DeletedApplication{}, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT DISTINCT image_ref
		FROM verification_results
		WHERE application_id = $1 AND image_ref <> ''
	`, id)
	if err != nil {
		return verify.DeletedApplication{}, err
	}
	var imageRefs []string
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			rows.Close()
			return verify.DeletedApplication{}, err
		}
		imageRefs = append(imageRefs, ref)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return verify.DeletedApplication{}, err
	}
	rows.Close()

	if _, err := tx.Exec(ctx, `
		DELETE FROM verification_results
		WHERE application_id = $1
	`, id); err != nil {
		return verify.DeletedApplication{}, err
	}

	tag, err := tx.Exec(ctx, `
		DELETE FROM applications
		WHERE application_id = $1
	`, id)
	if err != nil {
		return verify.DeletedApplication{}, err
	}
	if tag.RowsAffected() == 0 {
		return verify.DeletedApplication{}, verify.ErrApplicationNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return verify.DeletedApplication{}, err
	}
	sort.Strings(imageRefs)
	return verify.DeletedApplication{ApplicationID: id, ImageRefs: imageRefs}, nil
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) RunMigrations(ctx context.Context, dir string) error {
	files, err := migrationFiles(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		sql, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, string(sql)); err != nil {
			return err
		}
	}
	return nil
}

func migrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

type applicationRow struct {
	ID                    string
	Brand                 string
	ClassType             string
	NetContents           string
	ABV                   string
	GovernmentWarning     string
	NameAndAddress        string
	ForeignBlocks         []byte
	DeclaredLanguages     []string
	BrandNorm             string
	ClassTypeNorm         string
	NetContentsNorm       string
	ABVNorm               string
	GovernmentWarningNorm string
	NameAndAddressNorm    string
}

func applicationToRow(app verify.Application) (applicationRow, error) {
	foreignBlocks, err := json.Marshal(app.ForeignBlocks)
	if err != nil {
		return applicationRow{}, err
	}
	declaredLanguages := app.DeclaredLanguages
	if len(declaredLanguages) == 0 {
		declaredLanguages = []string{"en"}
	}
	return applicationRow{
		ID:                    app.ID,
		Brand:                 app.Brand,
		ClassType:             app.ClassType,
		NetContents:           app.NetContents,
		ABV:                   app.ABV,
		GovernmentWarning:     app.GovernmentWarning,
		NameAndAddress:        app.NameAndAddress,
		ForeignBlocks:         foreignBlocks,
		DeclaredLanguages:     declaredLanguages,
		BrandNorm:             match.NormalizeFlexible(app.Brand),
		ClassTypeNorm:         match.NormalizeFlexible(app.ClassType),
		NetContentsNorm:       match.NormalizeFlexible(app.NetContents),
		ABVNorm:               match.NormalizeFlexible(app.ABV),
		GovernmentWarningNorm: match.NormalizeExact(app.GovernmentWarning),
		NameAndAddressNorm:    match.NormalizeFlexible(app.NameAndAddress),
	}, nil
}
