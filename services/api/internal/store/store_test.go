package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ttb/labelverify/internal/verify"
)

func TestApplicationToRowPreNormalizesFields(t *testing.T) {
	row, err := applicationToRow(verify.Application{
		ID:                "app-1",
		Brand:             "ＳＴＯＮＥ",
		ClassType:         "Red Wine",
		NetContents:       "750 mL",
		ABV:               "13.5% ALC/VOL",
		GovernmentWarning: "GOVERNMENT WARNING",
		NameAndAddress:    "POM Creek Distilling Company, LLC Purcellville, VA",
		ForeignBlocks: []verify.ForeignBlock{{
			Text:               "Produit de France",
			EnglishTranslation: "Product of France",
			Language:           "fr",
		}},
		DeclaredLanguages: []string{"en", "fr"},
	})
	if err != nil {
		t.Fatalf("applicationToRow returned error: %v", err)
	}

	if row.BrandNorm != "stone" {
		t.Fatalf("BrandNorm=%q want stone", row.BrandNorm)
	}
	if row.GovernmentWarningNorm != "GOVERNMENT WARNING" {
		t.Fatalf("GovernmentWarningNorm=%q want GOVERNMENT WARNING", row.GovernmentWarningNorm)
	}
	if row.NameAndAddressNorm != "pom creek distilling company, llc purcellville, va" {
		t.Fatalf("NameAndAddressNorm=%q want normalized name/address", row.NameAndAddressNorm)
	}
	if len(row.ForeignBlocks) == 0 {
		t.Fatal("foreign blocks must be serialized")
	}
}

func TestApplicationToRowDefaultsDeclaredLanguagesToEnglish(t *testing.T) {
	row, err := applicationToRow(verify.Application{ID: "app-1"})
	if err != nil {
		t.Fatalf("applicationToRow returned error: %v", err)
	}
	if len(row.DeclaredLanguages) != 1 || row.DeclaredLanguages[0] != "en" {
		t.Fatalf("DeclaredLanguages=%v want [en]", row.DeclaredLanguages)
	}
}

func TestMigrationFilesSortLexically(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "0002_second.sql"), []byte("second"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0001_first.sql"), []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("skip"), 0o600); err != nil {
		t.Fatal(err)
	}

	files, err := migrationFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files)=%d want 2", len(files))
	}
	if filepath.Base(files[0]) != "0001_first.sql" || filepath.Base(files[1]) != "0002_second.sql" {
		t.Fatalf("files not sorted: %v", files)
	}
}
