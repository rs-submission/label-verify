package match

import (
	"strings"
	"testing"
)

// Canonical 27 CFR 16.21 Government Warning statement.
const ttbWarning = "GOVERNMENT WARNING: (1) According to the Surgeon General, women should not drink alcoholic beverages during pregnancy because of the risk of birth defects. (2) Consumption of alcoholic beverages impairs your ability to drive a car or operate machinery, and may cause health problems."

func warningResult(t *testing.T, expected, ocr string) FieldResult {
	t.Helper()
	pool := []TextRegion{{Text: ocr, Confidence: 0.99, X: 10, Y: 260, W: 900, H: 60}}
	return MatchAssigned([]FieldSpec{WarningField("government_warning", expected)}, pool)[0]
}

func TestWarning_CompleteTextPasses(t *testing.T) {
	res := warningResult(t, ttbWarning, ttbWarning)
	if !res.Pass {
		t.Fatalf("complete warning should pass, got score=%.3f diff=%q", res.Score, res.Diff)
	}
}

func TestWarning_HeaderSplitAcrossRegionsPasses(t *testing.T) {
	pool := []TextRegion{
		{Text: "GOVERNMENT", Confidence: 0.99, X: 10, Y: 260, W: 120, H: 20},
		{Text: "WARNING", Confidence: 0.99, X: 10, Y: 390, W: 110, H: 20},
	}

	res := MatchAssigned([]FieldSpec{WarningField("government_warning", "GOVERNMENT WARNING")}, pool)[0]

	if !res.Pass {
		t.Fatalf("split warning header should pass, got extracted=%q score=%.3f diff=%q", res.Extracted, res.Score, res.Diff)
	}
	if res.Extracted != "GOVERNMENT WARNING" {
		t.Fatalf("extracted=%q want GOVERNMENT WARNING", res.Extracted)
	}
}

func TestWarning_HeaderOnlyFailsWhenWarningTokenMissingDespiteBodyEvidence(t *testing.T) {
	ocr := "POM VODKA : (1) According to the Surgeon General, women birth defeets. (2) Consumption of alcohplic beverages impars your ability to drive car or operate machinery, and may cause health problems. GOVERNMENT"

	res := warningResult(t, "GOVERNMENT WARNING", ocr)

	if res.Pass {
		t.Fatalf("header-only warning must fail when WARNING is missing, got score=%.3f", res.Score)
	}
}

func TestWarning_HeaderOnlyBodyEvidenceStillRequiresAllCapsGovernment(t *testing.T) {
	ocr := "Government : (1) According to the Surgeon General, women birth defects. (2) Consumption of alcoholic beverages impairs your ability to drive car or operate machinery, and may cause health problems."

	res := warningResult(t, "GOVERNMENT WARNING", ocr)

	if res.Pass {
		t.Fatalf("title-cased Government must not pass header-only warning, got score=%.3f", res.Score)
	}
}

func TestWarningDocumentCandidateIsNotAssignedToClassType(t *testing.T) {
	pool := []TextRegion{
		{Text: "POM VODKA", Confidence: 0.99, X: 10, Y: 20, W: 160, H: 20},
		{Text: "GOVERNMENT", Confidence: 0.99, X: 10, Y: 260, W: 120, H: 20},
		{Text: "WARNING", Confidence: 0.99, X: 10, Y: 390, W: 110, H: 20},
	}
	specs := []FieldSpec{
		FuzzyField("class_type", "Vodka", 0.95),
		WarningField("government_warning", "GOVERNMENT WARNING"),
	}

	results := MatchAssigned(specs, pool)
	byField := indexResults(results)

	if byField["class_type"].Extracted != "POM VODKA" {
		t.Fatalf("class_type extracted %q want POM VODKA", byField["class_type"].Extracted)
	}
	if byField["government_warning"].Extracted != "GOVERNMENT WARNING" {
		t.Fatalf("government_warning extracted %q want GOVERNMENT WARNING", byField["government_warning"].Extracted)
	}
	if !byField["government_warning"].Pass {
		t.Fatalf("government_warning should pass, got score=%.3f diff=%q", byField["government_warning"].Score, byField["government_warning"].Diff)
	}
}

func TestWarningDocumentCandidateFailsWhenWarningTokenMissingButBodyPresent(t *testing.T) {
	pool := []TextRegion{
		{Text: "POM VODKA", Confidence: 0.99, X: 10, Y: 20, W: 160, H: 20},
		{Text: ": (1) According to the Surgeon General, women birth defeets. (2) Consumption of alcohplic beverages impars your ability to drive", Confidence: 0.97, X: 10, Y: 120, W: 900, H: 20},
		{Text: "car or operate machinery, and may cause health problems.", Confidence: 0.97, X: 10, Y: 150, W: 520, H: 20},
		{Text: "GOVERNMENT", Confidence: 0.99, X: 10, Y: 260, W: 120, H: 20},
	}
	specs := []FieldSpec{
		FuzzyField("class_type", "Vodka", 0.95),
		WarningField("government_warning", "GOVERNMENT WARNING"),
	}

	results := MatchAssigned(specs, pool)
	byField := indexResults(results)

	if byField["government_warning"].Pass {
		t.Fatalf("government_warning must fail when WARNING is missing, got extracted=%q score=%.3f", byField["government_warning"].Extracted, byField["government_warning"].Score)
	}
}

func TestWarning_CompleteWithOCRCharNoisePasses(t *testing.T) {
	// Every mandated word present but char-corrupted as real OCR does.
	noisy := "GOVERNMENT WARNING: (1) Accordng to the Surgeon Genral, wornen should not drink alcoholc beverages durng pregnancy because of the risk of birth defects. (2) Consumpton of alcoholic beverages impairs your abilty to drive a car or operate machnery, and may cause health problms."
	res := warningResult(t, ttbWarning, noisy)
	if !res.Pass {
		t.Fatalf("char-noisy but complete warning should pass, got score=%.3f diff=%q", res.Score, res.Diff)
	}
}

func TestWarning_SubstitutedWordFails(t *testing.T) {
	altered := "GOVERNMENT CAUTION: (1) According to the Surgeon General, women should not drink alcoholic beverages during pregnancy because of the risk of birth defects. (2) Consumption of alcoholic beverages impairs your ability to drive a car or operate machinery, and may cause health problems."
	res := warningResult(t, ttbWarning, altered)
	if res.Pass {
		t.Fatalf("warning with a substituted word must fail, got score=%.3f", res.Score)
	}
}

func TestWarning_AbsentFails(t *testing.T) {
	res := warningResult(t, ttbWarning, "BREWED AND BOTTLED IN VERMONT")
	if res.Pass {
		t.Fatalf("absent warning must fail, got score=%.3f", res.Score)
	}
}

func TestWarning_TruncatedFails(t *testing.T) {
	truncated := "GOVERNMENT WARNING: (1) According to the Surgeon General"
	res := warningResult(t, ttbWarning, truncated)
	if res.Pass {
		t.Fatalf("truncated warning must fail, got score=%.3f", res.Score)
	}
}

func TestWarning_HeaderWrongCaseFails(t *testing.T) {
	// Complete text, but the GOVERNMENT WARNING header is title-cased.
	titled := strings.Replace(ttbWarning, "GOVERNMENT WARNING", "Government Warning", 1)
	res := warningResult(t, ttbWarning, titled)
	if res.Pass {
		t.Fatalf("title-cased header must fail (case-sensitive header), got score=%.3f", res.Score)
	}
}

func TestWarning_HeaderAllCapsWithCharNoiseFails(t *testing.T) {
	// Header must match word-for-word; "WARNNG" is not "WARNING".
	noisyHeader := strings.Replace(ttbWarning, "GOVERNMENT WARNING", "GOVERNMENT WARNNG", 1)
	res := warningResult(t, ttbWarning, noisyHeader)
	if res.Pass {
		t.Fatalf("char-noisy header must fail, got score=%.3f", res.Score)
	}
}
