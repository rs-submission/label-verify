package verify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ttb/labelverify/internal/match"
	"github.com/ttb/labelverify/internal/ocr"
)

const imageReviewPersistenceReserve = 250 * time.Millisecond

var ErrApplicationNotFound = errors.New("application not found")

type Application struct {
	ID                string
	Brand             string
	ClassType         string
	NetContents       string
	ABV               string
	GovernmentWarning string
	NameAndAddress    string
	ForeignBlocks     []ForeignBlock
	DeclaredLanguages []string
}

type ApplicationSummary struct {
	ID        string
	Brand     string
	ClassType string
}

type ForeignBlock struct {
	Text               string
	EnglishTranslation string
	Language           string
}

type StoredVerification struct {
	ApplicationID string
	ImageRef      string
	Verdict       match.Verdict
}

type DeletedApplication struct {
	ApplicationID string
	ImageRefs     []string
}

type BatchItem struct {
	ApplicationID string
	Image         []byte
}

type BatchResult struct {
	ApplicationID string
	Verdict       match.Verdict
	Error         error
}

type Store interface {
	GetApplication(ctx context.Context, id string) (Application, error)
	SaveVerification(ctx context.Context, result StoredVerification) error
}

type OCRClient interface {
	Recognize(ctx context.Context, image []byte, langs []string) (ocr.RecognizeResponse, error)
}

type ImageStore interface {
	Put(id string, data []byte) (ref string, err error)
}

type FieldAdjudicator interface {
	ReviewFields(ctx context.Context, fields []match.FieldResult, pool []match.TextRegion) []match.FieldResult
}

type ImageReviewFuture interface {
	ReviewFields(ctx context.Context, fields []match.FieldResult) []match.FieldResult
}

type ImageReviewer interface {
	StartReview(ctx context.Context, app Application, image []byte) ImageReviewFuture
}

type Service struct {
	store           Store
	ocr             OCRClient
	images          ImageStore
	adjudicator     FieldAdjudicator
	perImageTimeout time.Duration
}

func NewService(store Store, ocrClient OCRClient) *Service {
	return &Service{store: store, ocr: ocrClient}
}

func NewServiceWithAdjudicator(store Store, ocrClient OCRClient, adjudicator FieldAdjudicator) *Service {
	return &Service{store: store, ocr: ocrClient, adjudicator: adjudicator}
}

func (s *Service) WithImageStore(images ImageStore) *Service {
	s.images = images
	return s
}

func (s *Service) WithTimeout(timeout time.Duration) *Service {
	s.perImageTimeout = timeout
	return s
}

func (s *Service) WithAdjudicator(adjudicator FieldAdjudicator) *Service {
	s.adjudicator = adjudicator
	return s
}

func (s *Service) VerifyBatch(ctx context.Context, items []BatchItem, maxConcurrency int) []BatchResult {
	if maxConcurrency <= 0 {
		maxConcurrency = 4
	}
	results := make([]BatchResult, len(items))
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, item := range items {
		i, item := i, item
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[i] = BatchResult{ApplicationID: item.ApplicationID, Error: ctx.Err()}
				return
			}

			verdict, err := s.VerifySingle(ctx, item.ApplicationID, item.Image)
			results[i] = BatchResult{ApplicationID: item.ApplicationID, Verdict: verdict, Error: err}
		}()
	}

	wg.Wait()
	return results
}

func (s *Service) VerifySingle(ctx context.Context, appID string, image []byte) (match.Verdict, error) {
	return s.verifySingle(ctx, appID, image, s.adjudicator, nil)
}

func (s *Service) VerifySingleWithAdjudicator(ctx context.Context, appID string, image []byte, adjudicator FieldAdjudicator) (match.Verdict, error) {
	return s.verifySingle(ctx, appID, image, adjudicator, nil)
}

func (s *Service) VerifySingleWithImageReviewer(ctx context.Context, appID string, image []byte, reviewer ImageReviewer) (match.Verdict, error) {
	return s.verifySingle(ctx, appID, image, nil, reviewer)
}

func (s *Service) verifySingle(ctx context.Context, appID string, image []byte, adjudicator FieldAdjudicator, reviewer ImageReviewer) (match.Verdict, error) {
	if s.perImageTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.perImageTimeout)
		defer cancel()
	}

	app, err := s.store.GetApplication(ctx, appID)
	if err != nil {
		return match.Verdict{}, err
	}
	var imageReview ImageReviewFuture
	if reviewer != nil {
		imageReview = reviewer.StartReview(ctx, app, image)
	}

	ocrResult, err := s.ocr.Recognize(ctx, image, advisoryLangs(app))
	if err != nil {
		return match.Verdict{}, err
	}
	pool := toMatchRegions(ocrResult.Regions)

	specs := []match.FieldSpec{
		match.FuzzyField("brand", app.Brand, 0.85),
		match.FuzzyField("class_type", app.ClassType, 0.95),
		match.FormatField("net_contents", app.NetContents),
		match.FormatField("abv", app.ABV),
		match.WarningField("government_warning", app.GovernmentWarning),
	}
	if strings.TrimSpace(app.NameAndAddress) != "" {
		specs = append(specs, match.NameAddressField("name_address", app.NameAndAddress, 0.85))
	}

	for i, block := range app.ForeignBlocks {
		specs = append(specs, match.PresenceField(numberedField("foreign_translation", i), block.EnglishTranslation))
		specs = append(specs, match.FuzzyField(numberedField("foreign_text", i), block.Text, 0.85))
	}
	fields := match.MatchAssigned(specs, pool)
	if adjudicator != nil {
		fields = adjudicator.ReviewFields(ctx, fields, pool)
	}
	if imageReview != nil {
		reviewCtx, cancelReview := imageReviewContext(ctx)
		fields = imageReview.ReviewFields(reviewCtx, fields)
		cancelReview()
	}

	verdict := match.Aggregate(fields)
	imageRef, err := s.storeImage(appID, image)
	if err != nil {
		return match.Verdict{}, err
	}
	if err := s.store.SaveVerification(ctx, StoredVerification{ApplicationID: appID, ImageRef: imageRef, Verdict: verdict}); err != nil {
		return match.Verdict{}, err
	}
	return verdict, nil
}

func imageReviewContext(ctx context.Context) (context.Context, context.CancelFunc) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return ctx, func() {}
	}
	timeout := time.Until(deadline) - imageReviewPersistenceReserve
	if timeout <= 0 {
		reviewCtx, cancel := context.WithCancel(ctx)
		cancel()
		return reviewCtx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func (s *Service) storeImage(appID string, image []byte) (string, error) {
	if s.images == nil || len(image) == 0 {
		return "", nil
	}
	// Store before the DB row so the verification audit can reference the saved image.
	return s.images.Put(imageRef(appID, image), image)
}

func imageRef(appID string, image []byte) string {
	sum := sha256.Sum256(image)
	return safeRefPrefix(appID) + "-" + hex.EncodeToString(sum[:8]) + imageExt(image)
}

func safeRefPrefix(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "application"
	}
	return b.String()
}

func imageExt(image []byte) string {
	switch http.DetectContentType(image) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}

func toMatchRegions(regions []ocr.Region) []match.TextRegion {
	out := make([]match.TextRegion, 0, len(regions))
	for _, region := range regions {
		out = append(out, match.TextRegion{
			Text:       region.Text,
			Confidence: region.Confidence,
			X:          region.BBox[0],
			Y:          region.BBox[1],
			W:          region.BBox[2],
			H:          region.BBox[3],
		})
	}
	return out
}

func advisoryLangs(app Application) []string {
	langs := []string{"en"}
	for _, lang := range app.DeclaredLanguages {
		if len(lang) == 2 && !slices.Contains(langs, lang) {
			langs = append(langs, lang)
		}
	}
	for _, block := range app.ForeignBlocks {
		if len(block.Language) == 2 && !slices.Contains(langs, block.Language) {
			langs = append(langs, block.Language)
		}
	}
	return langs
}

func numberedField(prefix string, index int) string {
	return prefix + "_" + strconv.Itoa(index+1)
}
