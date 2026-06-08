package match

import (
	"strings"
	"testing"
)

func TestMatchAssignedUsesGlobalFieldCandidateChoice(t *testing.T) {
	pool := []TextRegion{
		reg("POM BOURBON", 40, 80),
		reg("DISTILLED FROM MASH", 50, 210),
		reg("INGREDIENTS", 55, 275),
		reg("45% ALC. BY VOL. (90 PROOF)", 50, 430),
	}
	specs := []FieldSpec{
		FuzzyField("brand", "Stone Throw", 0.85),
		FuzzyField("class_type", "Red Wine", 0.95),
		FormatField("abv", "13.5% ALC/VOL"),
		ExactField("government_warning", "GOVERNMENT WARNING"),
	}

	results := MatchAssigned(specs, pool)
	byField := indexResults(results)

	if byField["brand"].Extracted != "POM BOURBON" {
		t.Fatalf("brand extracted %q, want POM BOURBON", byField["brand"].Extracted)
	}
	if byField["abv"].Extracted != "45% ALC. BY VOL. (90 PROOF)" {
		t.Fatalf("abv extracted %q, want alcohol content", byField["abv"].Extracted)
	}
	if byField["government_warning"].Extracted != "" {
		t.Fatalf("government_warning extracted %q, want missing", byField["government_warning"].Extracted)
	}
	if byField["brand"].Pass {
		t.Fatal("mismatched brand must still fail")
	}
}

func TestMatchAssignedPrefersNarrowFormatWindow(t *testing.T) {
	pool := []TextRegion{
		reg("POMCREEKDISTILLING COMPANY,LLC", 379, 1294),
		reg("PURCELLVILLE,VA", 753, 1339),
		reg("25% ALC.BYVOL. (50 PR00F)", 552, 1470),
	}
	specs := []FieldSpec{
		FormatField("abv", "25% ALC/VOL(50 Proof)"),
	}

	results := MatchAssigned(specs, pool)

	if !results[0].Pass {
		t.Fatalf("abv should pass: %+v", results[0])
	}
	if results[0].Extracted != "25% ALC.BYVOL. (50 PR00F)" {
		t.Fatalf("abv extracted %q, want narrow alcohol candidate", results[0].Extracted)
	}
}

func TestClassTypeScoreUsesEquivalentClassTerms(t *testing.T) {
	pool := []TextRegion{
		reg("POM BLUEBERRY CORDIAL", 7, 11),
	}
	specs := []FieldSpec{
		FuzzyField("class_type", "Blueberry Liqueur/Cordial", 0.95),
	}

	results := MatchAssigned(specs, pool)

	if !results[0].Pass {
		t.Fatalf("class_type should pass through liqueur/cordial equivalence: %+v", results[0])
	}
	if results[0].Extracted != "POM BLUEBERRY CORDIAL" {
		t.Fatalf("class_type extracted %q, want label product text", results[0].Extracted)
	}
}

func TestSingleClassTermAllowsExpectedDescriptorInProductPhrase(t *testing.T) {
	pool := []TextRegion{
		reg("POM RYE WHISKEY", 7, 11),
	}
	specs := []FieldSpec{
		FuzzyField("class_type", "Rye Whiskey", 0.95),
	}

	results := MatchAssigned(specs, pool)

	if !results[0].Pass {
		t.Fatalf("class_type should pass when product phrase preserves expected descriptor: %+v", results[0])
	}
}

func TestSingleClassTermAllowsCommonClassQualifiers(t *testing.T) {
	pool := []TextRegion{
		reg("KENTUCKY STRAIGHT BOURBON WHISKEY", 7, 11),
	}
	specs := []FieldSpec{
		FuzzyField("class_type", "Bourbon", 0.95),
	}

	results := MatchAssigned(specs, pool)

	if !results[0].Pass {
		t.Fatalf("class_type should pass when richer class designation contains expected class: %+v", results[0])
	}
}

func TestClassTypeScoreUsesSingleMaltDesignationEvidence(t *testing.T) {
	pool := []TextRegion{
		reg("POM SINGLE MALT", 7, 11),
		reg("100% MALTED BARLEY", 50, 90),
	}
	specs := []FieldSpec{
		FuzzyField("class_type", "Single Malt Whiskey", 0.95),
	}

	results := MatchAssigned(specs, pool)

	if !results[0].Pass {
		t.Fatalf("class_type should pass from explicit single malt display evidence: %+v", results[0])
	}
	if results[0].Extracted != "POM SINGLE MALT" {
		t.Fatalf("class_type extracted %q, want product display line", results[0].Extracted)
	}
}

func TestClassTypeDoesNotPassFromIngredientEvidenceAlone(t *testing.T) {
	pool := []TextRegion{
		reg("100% MALTED BARLEY", 50, 90),
	}
	specs := []FieldSpec{
		FuzzyField("class_type", "Single Malt Whiskey", 0.95),
	}

	results := MatchAssigned(specs, pool)

	if results[0].Pass {
		t.Fatalf("class_type must not pass from ingredient evidence alone: %+v", results[0])
	}
}

func TestSingleTokenClassCanReuseMatchedBrandDisplayLine(t *testing.T) {
	pool := []TextRegion{
		reg("POM BOURBON", 40, 80),
		reg("750 mL", 50, 420),
		reg("45% ALC/VOL (90 Proof)", 50, 455),
		reg("GOVERNMENT WARNING", 50, 500),
	}
	specs := []FieldSpec{
		FuzzyField("brand", "POM BOURBON", 0.85),
		FuzzyField("class_type", "Bourbon", 0.95),
		FormatField("net_contents", "750 mL"),
		FormatField("abv", "45% ALC/VOL (90 Proof)"),
		ExactField("government_warning", "GOVERNMENT WARNING"),
	}

	results := MatchAssigned(specs, pool)
	byField := indexResults(results)

	if !byField["brand"].Pass {
		t.Fatalf("brand should own exact product phrase: %+v", byField["brand"])
	}
	if !byField["class_type"].Pass {
		t.Fatalf("class_type should pass from matched brand display line: %+v", byField["class_type"])
	}
	if byField["class_type"].Extracted != "POM BOURBON" {
		t.Fatalf("class_type extracted %q want POM BOURBON", byField["class_type"].Extracted)
	}
}

func TestClassTypeDoesNotFallThroughToProductionWhenBrandDisplayLineContainsClass(t *testing.T) {
	pool := []TextRegion{
		reg("POM VODKA", 40, 80),
		reg("DISTILLEDFROMGRAIN", 40, 180),
		reg("40% ALC. BY VOL. (80 PR00F)", 40, 420),
	}
	specs := []FieldSpec{
		FuzzyField("brand", "POM VODKA", 0.85),
		FuzzyField("class_type", "Vodka", 0.95),
		FormatField("abv", "40% Alc./Vol. (80 Proof)"),
	}

	results := MatchAssigned(specs, pool)
	byField := indexResults(results)

	if byField["class_type"].Extracted != "POM VODKA" {
		t.Fatalf("class_type extracted %q want POM VODKA", byField["class_type"].Extracted)
	}
	if !byField["class_type"].Pass {
		t.Fatalf("class_type should pass from brand display line: %+v", byField["class_type"])
	}
}

func TestNameAddressMatchesProducerAndCityStateWindow(t *testing.T) {
	pool := []TextRegion{
		reg("POM CREEK DISTILLING COMPANY, LLC", 0.04, 0.52),
		reg("PURCELLVILLE, VA", 0.045, 0.555),
	}
	specs := []FieldSpec{
		NameAddressField("name_address", "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA", 0.85),
	}

	results := MatchAssigned(specs, pool)

	if !results[0].Pass {
		t.Fatalf("name_address should pass from producer plus city/state window: %+v", results[0])
	}
	if results[0].Extracted != "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA" {
		t.Fatalf("name_address extracted %q want joined producer/address", results[0].Extracted)
	}
}

func TestNameAddressRequiresAddressWhenExpectedIncludesLocation(t *testing.T) {
	pool := []TextRegion{
		reg("POM CREEK DISTILLING COMPANY, LLC", 0.04, 0.52),
	}
	specs := []FieldSpec{
		NameAddressField("name_address", "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA", 0.85),
	}

	results := MatchAssigned(specs, pool)

	if results[0].Pass {
		t.Fatalf("name_address must not pass from producer-only text when city/state is expected: %+v", results[0])
	}
}

func TestNameAddressCombinesSeparatedProducerAndCityState(t *testing.T) {
	pool := []TextRegion{
		reg("POM CREEK DISTILLING COMPANY, LLC", 0.1, 0.75),
		reg("750 mL", 0.1, 0.80),
		reg("45% ALC/VOL (90 Proof)", 0.1, 0.85),
		reg("PURCELLVILLE, VA", 0.1, 0.60),
	}
	specs := []FieldSpec{
		NameAddressField("name_address", "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA", 0.85),
	}

	results := MatchAssigned(specs, pool)

	if !results[0].Pass {
		t.Fatalf("name_address should combine separated producer and city/state: %+v", results[0])
	}
	if strings.Contains(results[0].Extracted, "750 mL") || strings.Contains(results[0].Extracted, "ALC/VOL") {
		t.Fatalf("name_address should avoid format fields in extracted text: %+v", results[0])
	}
}

func TestNameAddressAcceptsCompactProducerOCR(t *testing.T) {
	pool := []TextRegion{
		reg("POMCREEKDISTILLINGCOMPANY,LLC", 0.1, 0.75),
		reg("PURCELLVILLE,VA", 0.1, 0.80),
	}
	specs := []FieldSpec{
		NameAddressField("name_address", "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA", 0.85),
	}

	results := MatchAssigned(specs, pool)

	if !results[0].Pass {
		t.Fatalf("name_address should pass with compact OCR producer text: %+v", results[0])
	}
}

func TestNameAddressMatchesRolePhrasePersonNameAndCityState(t *testing.T) {
	pool := []TextRegion{
		reg("BOTTLED BY JANE SMITH", 0.1, 0.55),
		reg("AUSTIN, TX", 0.1, 0.75),
	}
	specs := []FieldSpec{
		NameAddressField("name_address", "Bottled by Jane Smith Austin, TX", 0.85),
	}

	results := MatchAssigned(specs, pool)

	if !results[0].Pass {
		t.Fatalf("name_address should pass for role phrase plus person/city/state: %+v", results[0])
	}
}

func TestNameAddressMatchesImporterWithoutCompanySuffix(t *testing.T) {
	pool := []TextRegion{
		reg("IMPORTED BY MAISON ROUGE", 0.1, 0.55),
		reg("NEW YORK, NY", 0.1, 0.75),
	}
	specs := []FieldSpec{
		NameAddressField("name_address", "Imported by Maison Rouge New York, NY", 0.85),
	}

	results := MatchAssigned(specs, pool)

	if !results[0].Pass {
		t.Fatalf("name_address should pass for importer role and city/state: %+v", results[0])
	}
}

func TestNameAddressRejectsNameOnlyWhenExpectedHasLocationWithoutStatePattern(t *testing.T) {
	pool := []TextRegion{
		reg("BOTTLED BY JANE SMITH", 0.1, 0.55),
	}
	specs := []FieldSpec{
		NameAddressField("name_address", "Bottled by Jane Smith Austin, TX", 0.85),
	}

	results := MatchAssigned(specs, pool)

	if results[0].Pass {
		t.Fatalf("name_address must not pass from name/role without expected city/state: %+v", results[0])
	}
}

func TestNameAddressRejectsPollutedFormatWindow(t *testing.T) {
	candidate := TextCandidate{
		Text:       "POM CREEK DISTILLING COMPANY, LLC 750 mL 45% ALC/VOL GOVERNMENT WARNING PURCELLVILLE, VA",
		Confidence: 0.99,
		Regions:    5,
	}
	spec := NameAddressField("name_address", "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA", 0.85)

	result := resultForCandidate(spec, candidate)

	if result.Pass {
		t.Fatalf("polluted name/address candidate must not pass: %+v", result)
	}
}

func TestBrandDoesNotClaimProducerAddressText(t *testing.T) {
	pool := []TextRegion{
		reg("BOURBON", 0.04, 0.08),
		reg("POM CREEK DISTILLING COMPANY, LLC", 0.04, 0.52),
		reg("PURCELLVILLE, VA", 0.045, 0.555),
	}
	specs := []FieldSpec{
		FuzzyField("brand", "POM BOURBON", 0.85),
		FuzzyField("class_type", "Bourbon", 0.95),
		NameAddressField("name_address", "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA", 0.85),
	}

	results := MatchAssigned(specs, pool)
	byField := indexResults(results)

	if !byField["name_address"].Pass {
		t.Fatalf("name_address should pass: %+v", byField["name_address"])
	}
	if !byField["class_type"].Pass {
		t.Fatalf("class_type should pass: %+v", byField["class_type"])
	}
	if byField["brand"].Extracted == "POM CREEK DISTILLING COMPANY, LLC" ||
		byField["brand"].Extracted == "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA" {
		t.Fatalf("brand must not claim producer/address text: %+v", byField["brand"])
	}
}

func TestLocationPenaltyUsesGenericLocationSignals(t *testing.T) {
	location := TextCandidate{Text: "BOTTLED IN RICHMOND, VA", Confidence: 0.99, X: 10, Y: 100, W: 200, H: 20, Regions: 1}
	brandish := TextCandidate{Text: "POM CREEK DISTILLING COMPANY", Confidence: 0.99, X: 10, Y: 100, W: 200, H: 20, Regions: 1}

	if assignmentWeight(FuzzyField("brand", "POM CREEK DISTILLING COMPANY", 0.85), location) >= assignmentWeight(FuzzyField("brand", "POM CREEK DISTILLING COMPANY", 0.85), brandish) {
		t.Fatal("generic state-code location text should be penalized for brand assignment")
	}
}

func TestLocationPenaltyDoesNotTreatLowercaseStateWordsAsLocations(t *testing.T) {
	phrase := TextCandidate{Text: "DISTILLED IN OAK OR STEEL", Confidence: 0.99, X: 10, Y: 100, W: 200, H: 20, Regions: 1}
	baseline := TextCandidate{Text: "DISTILLED WITH OAK AND STEEL", Confidence: 0.99, X: 10, Y: 100, W: 200, H: 20, Regions: 1}

	field := FuzzyField("class_type", "Oak Whiskey", 0.95)
	if assignmentWeight(field, phrase) < assignmentWeight(field, baseline)-0.2 {
		t.Fatal("bare words IN/OR must not trigger state-code location penalty")
	}
}

func TestFlexibleSimilarityHandlesMissingOCRSpaces(t *testing.T) {
	score := FlexibleSimilarity(
		"POM CREEK DISTILLING COMPANY, LLC",
		"POMCREEKDISTILLING COMPANY,LLC",
	)
	if score < 0.95 {
		t.Fatalf("score=%f want >= 0.95", score)
	}
}

func TestAssembleCandidateWindowsSupportsPixelCoordinates(t *testing.T) {
	pool := []TextRegion{
		{Text: "POM", Confidence: 0.99, X: 40, Y: 80, W: 90, H: 32},
		{Text: "CHERRY", Confidence: 0.99, X: 150, Y: 82, W: 135, H: 32},
		{Text: "CORDIAL", Confidence: 0.99, X: 300, Y: 81, W: 170, H: 32},
	}

	windows := AssembleWindows(pool, 0.25)
	if !contains(windows, "POM CHERRY CORDIAL") {
		t.Fatalf("windows=%v, want joined pixel-coordinate brand", windows)
	}
}

func indexResults(results []FieldResult) map[string]FieldResult {
	out := make(map[string]FieldResult, len(results))
	for _, result := range results {
		out[result.Field] = result
	}
	return out
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
