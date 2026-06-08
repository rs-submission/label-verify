package match

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/ttb/labelverify/internal/classify"
)

// warningCoverageThreshold requires essentially every mandated word to be present;
// at typical statement lengths this fails on a single missing or substituted word.
const warningCoverageThreshold = 0.99

// warningWordSimilarity tolerates OCR character noise per word (e.g. "wornen" for
// "women") while staying far above the distance of a genuine word substitution.
const warningWordSimilarity = 0.6

type FieldSpec struct {
	Field     string
	Expected  string
	MatchType string
	Threshold float64
}

func FuzzyField(field, expected string, threshold float64) FieldSpec {
	return FieldSpec{Field: field, Expected: expected, MatchType: "fuzzy", Threshold: threshold}
}

func NameAddressField(field, expected string, threshold float64) FieldSpec {
	return FieldSpec{Field: field, Expected: expected, MatchType: "name_address", Threshold: threshold}
}

func ExactField(field, expected string) FieldSpec {
	return FieldSpec{Field: field, Expected: expected, MatchType: "exact"}
}

func FormatField(field, expected string) FieldSpec {
	return FieldSpec{Field: field, Expected: expected, MatchType: "format"}
}

func PresenceField(field, expected string) FieldSpec {
	return FieldSpec{Field: field, Expected: expected, MatchType: "presence"}
}

// WarningField matches mandated fixed-text statements (e.g. the TTB GOVERNMENT
// WARNING) by word coverage: every expected word must appear in the OCR'd block,
// with per-word fuzzy tolerance for OCR character noise. A missing or substituted
// word drops coverage below threshold and fails.
func WarningField(field, expected string) FieldSpec {
	return FieldSpec{Field: field, Expected: expected, MatchType: "warning", Threshold: warningCoverageThreshold}
}

func MatchAssigned(specs []FieldSpec, pool []TextRegion) []FieldResult {
	results := make([]FieldResult, len(specs))
	assignable := make([]int, 0, len(specs))
	for i, spec := range specs {
		if spec.MatchType == "presence" {
			results[i] = MatchPresence(spec.Expected)
			results[i].Field = spec.Field
			continue
		}
		assignable = append(assignable, i)
	}
	if len(assignable) == 0 {
		return results
	}

	candidates := AssembleCandidateWindows(pool, 0.25)
	if hasWarningSpec(specs, assignable) {
		candidates = append(candidates, documentCandidate(pool))
	}
	candidates = append(candidates, nameAddressCandidates(specs, assignable, candidates)...)
	candidates = uniqueCandidates(candidates)
	if len(candidates) == 0 {
		for _, specIndex := range assignable {
			results[specIndex] = missingResult(specs[specIndex])
		}
		return results
	}

	rowWeights := make([][]float64, len(assignable))
	for row, specIndex := range assignable {
		rowWeights[row] = make([]float64, len(candidates))
		for col, candidate := range candidates {
			rowWeights[row][col] = assignmentWeight(specs[specIndex], candidate)
		}
	}

	assignments := greedyAssignments(rowWeights)
	for row, col := range assignments {
		spec := specs[assignable[row]]
		if col < 0 || col >= len(candidates) {
			results[assignable[row]] = missingResult(spec)
			continue
		}
		results[assignable[row]] = resultForCandidate(spec, candidates[col])
	}
	applyDisplayLineClassFallback(specs, results)
	return results
}

func applyDisplayLineClassFallback(specs []FieldSpec, results []FieldResult) {
	brandIndex := -1
	classIndex := -1
	for i, spec := range specs {
		switch spec.Field {
		case "brand":
			brandIndex = i
		case "class_type":
			classIndex = i
		}
	}
	if brandIndex < 0 || classIndex < 0 {
		return
	}
	brand := results[brandIndex]
	classResult := results[classIndex]
	classSpec := specs[classIndex]
	threshold := classSpec.Threshold
	if threshold <= 0 {
		threshold = 0.85
	}
	if !brand.Pass {
		return
	}
	if classify.ExpectedEvidenceTermCount(classSpec.Expected) != 1 {
		return
	}
	if classify.ClassTypeScore(classSpec.Expected, specs[brandIndex].Expected) < threshold {
		return
	}
	score := classify.ClassTypeScore(classSpec.Expected, brand.Extracted)
	if score < threshold {
		return
	}
	if classResult.Pass && candidateSpecificity(classResult.Extracted) <= candidateSpecificity(brand.Extracted) {
		return
	}
	results[classIndex] = FieldResult{
		Field:     classSpec.Field,
		Expected:  classSpec.Expected,
		Extracted: brand.Extracted,
		MatchType: classSpec.MatchType,
		Score:     score,
		Pass:      true,
	}
}

func candidateSpecificity(value string) int {
	return len(tokenSet(value))
}

func hasWarningSpec(specs []FieldSpec, assignable []int) bool {
	for _, index := range assignable {
		if specs[index].MatchType == "warning" {
			return true
		}
	}
	return false
}

func documentCandidate(pool []TextRegion) TextCandidate {
	regions := append([]TextRegion(nil), pool...)
	lineTolerance := adaptiveLineTolerance(regions)
	sort.SliceStable(regions, func(i, j int) bool {
		if abs(regions[i].Y-regions[j].Y) <= lineTolerance {
			return regions[i].X < regions[j].X
		}
		return regions[i].Y < regions[j].Y
	})

	candidate := emptyCandidate()
	for _, region := range regions {
		candidate = candidate.with(region)
	}
	candidate.Document = true
	return candidate
}

func nameAddressCandidates(specs []FieldSpec, assignable []int, base []TextCandidate) []TextCandidate {
	var out []TextCandidate
	for _, index := range assignable {
		spec := specs[index]
		if spec.MatchType != "name_address" {
			continue
		}
		out = append(out, nameAddressCandidatesForSpec(spec, base)...)
	}
	return out
}

func nameAddressCandidatesForSpec(spec FieldSpec, base []TextCandidate) []TextCandidate {
	var nameParts []TextCandidate
	var locationParts []TextCandidate
	for _, candidate := range base {
		if candidate.Document {
			continue
		}
		if isNameAddressNamePart(spec.Expected, candidate.Text) {
			nameParts = append(nameParts, candidate)
		}
		if isNameAddressLocationPart(spec.Expected, candidate.Text) {
			locationParts = append(locationParts, candidate)
		}
	}

	out := make([]TextCandidate, 0, len(nameParts)*len(locationParts))
	for _, namePart := range nameParts {
		for _, locationPart := range locationParts {
			if NormalizeFlexible(namePart.Text) == NormalizeFlexible(locationPart.Text) {
				continue
			}
			out = append(out, combineCandidates(namePart, locationPart))
		}
	}
	return out
}

func isNameAddressNamePart(expected, candidate string) bool {
	text := NormalizeFlexible(candidate)
	if text == "" || containsAny(text, formatTerms, warningTerms) {
		return false
	}
	if containsAny(text, producerTerms, roleTerms) {
		return true
	}
	return nameAddressTokenCoverage(expected, candidate, true) >= 0.45
}

func isNameAddressLocationPart(expected, candidate string) bool {
	text := NormalizeFlexible(candidate)
	if text == "" || containsAny(text, formatTerms, warningTerms) {
		return false
	}
	if mentionsLocation(candidate) || nameAddressLocationCoverage(expected, candidate) >= 0.50 {
		return true
	}
	return mentionsLocation(expected) && nameAddressLocationCoverage(expected, candidate) >= 0.25
}

func combineCandidates(first, second TextCandidate) TextCandidate {
	text := strings.TrimSpace(first.Text + " " + second.Text)
	x0 := min(first.X, second.X)
	y0 := min(first.Y, second.Y)
	x1 := max(first.X+first.W, second.X+second.W)
	y1 := max(first.Y+first.H, second.Y+second.H)
	regions := first.Regions + second.Regions
	confidence := 0.0
	if regions > 0 {
		confidence = (first.Confidence*float64(first.Regions) + second.Confidence*float64(second.Regions)) / float64(regions)
	}
	return TextCandidate{
		Text:       text,
		Confidence: confidence,
		X:          x0,
		Y:          y0,
		W:          x1 - x0,
		H:          y1 - y0,
		Regions:    regions,
	}
}

func uniqueCandidates(candidates []TextCandidate) []TextCandidate {
	seen := make(map[string]TextCandidate, len(candidates))
	order := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		key := NormalizeFlexible(candidate.Text)
		if key == "" {
			continue
		}
		existing, ok := seen[key]
		if !ok {
			seen[key] = candidate
			order = append(order, key)
			continue
		}
		if candidateQuality(candidate) > candidateQuality(existing) {
			seen[key] = candidate
		}
	}

	out := make([]TextCandidate, 0, len(order))
	for _, key := range order {
		out = append(out, seen[key])
	}
	sort.SliceStable(out, func(i, j int) bool {
		return candidateQuality(out[i]) > candidateQuality(out[j])
	})
	return out
}

func candidateQuality(candidate TextCandidate) float64 {
	return candidate.Confidence + 0.05/(float64(candidate.Regions)+1) - candidate.Y*0.0001
}

func resultForCandidate(spec FieldSpec, candidate TextCandidate) FieldResult {
	switch spec.MatchType {
	case "exact":
		return exactCandidateResult(spec, candidate)
	case "format":
		return formatCandidateResult(spec, candidate)
	case "warning":
		return warningCandidateResult(spec, candidate)
	default:
		return fuzzyCandidateResult(spec, candidate)
	}
}

func warningCandidateResult(spec FieldSpec, candidate TextCandidate) FieldResult {
	threshold := spec.Threshold
	if threshold <= 0 {
		threshold = warningCoverageThreshold
	}
	score, headerOK := warningCandidateScore(spec.Expected, candidate.Text)
	result := FieldResult{
		Field:     spec.Field,
		Expected:  spec.Expected,
		Extracted: candidate.Text,
		MatchType: "warning",
		Score:     score,
	}
	result.Pass = score >= threshold && headerOK
	if result.Pass && NormalizeExact(spec.Expected) == "GOVERNMENT WARNING" && warningHeaderOK(spec.Expected, candidate.Text) {
		result.Extracted = spec.Expected
	}
	if !result.Pass {
		switch {
		case !headerOK:
			result.Diff = "government warning header missing or not in required case"
		default:
			result.Diff = fmt.Sprintf("government warning incomplete or altered (word coverage %.0f%%)", score*100)
		}
	}
	return result
}

func warningCandidateScore(expected, candidate string) (float64, bool) {
	score := warningCoverage(expected, candidate)
	headerOK := warningHeaderOK(expected, candidate)
	return score, headerOK
}

// warningHeaderOK enforces case-sensitivity on the fixed all-caps header (the
// leading uppercase run of the expected text, e.g. "GOVERNMENT WARNING") while
// the body is matched case-insensitively by coverage. Each header word must
// appear in the candidate as an all-caps token (char-noise tolerated, case not).
func warningHeaderOK(expected, candidate string) bool {
	header := warningHeaderWords(expected)
	if len(header) == 0 {
		return true
	}
	candidateWords := warningWords(candidate)
	for _, headerWord := range header {
		if !candidateHasCasedWord(headerWord, candidateWords) {
			return false
		}
	}
	return true
}

func warningHeaderWords(expected string) []string {
	var header []string
	for _, word := range warningWords(expected) {
		if !isAllCaps(word) {
			break
		}
		header = append(header, word)
	}
	return header
}

func candidateHasCasedWord(headerWord string, candidateWords []string) bool {
	for _, word := range candidateWords {
		if word == headerWord && isAllCaps(word) {
			return true
		}
	}
	return false
}

// warningWords splits into alphanumeric runs preserving original case.
func warningWords(value string) []string {
	var out []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			out = append(out, string(current))
			current = nil
		}
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func isAllCaps(word string) bool {
	hasLetter := false
	for _, r := range word {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}
	return hasLetter
}

// warningCoverage is the fraction of expected words present in the candidate,
// each matched exactly or within warningWordSimilarity to absorb OCR noise.
func warningCoverage(expected, candidate string) float64 {
	expectedWords := tokenSet(expected)
	if len(expectedWords) == 0 {
		return 0
	}
	candidateWords := tokenSet(candidate)
	matched := 0
	for word := range expectedWords {
		if warningWordPresent(word, candidateWords) {
			matched++
		}
	}
	return float64(matched) / float64(len(expectedWords))
}

func warningWordPresent(word string, candidateWords map[string]bool) bool {
	if candidateWords[word] {
		return true
	}
	for candidate := range candidateWords {
		if Similarity(word, candidate) >= warningWordSimilarity {
			return true
		}
	}
	return false
}

func exactCandidateResult(spec FieldSpec, candidate TextCandidate) FieldResult {
	expectedNorm := NormalizeExact(spec.Expected)
	gotNorm := NormalizeExact(candidate.Text)
	result := FieldResult{
		Field:     spec.Field,
		Expected:  spec.Expected,
		Extracted: candidate.Text,
		MatchType: "exact",
	}
	if gotNorm == expectedNorm {
		result.Score = 1
		result.Pass = true
		return result
	}
	result.Score = FlexibleSimilarity(spec.Expected, candidate.Text)
	result.Diff = diff(spec.Expected, candidate.Text)
	return result
}

func formatCandidateResult(spec FieldSpec, candidate TextCandidate) FieldResult {
	expected, expectedOK := parseFormat(spec.Field, spec.Expected)
	got, gotOK := parseFormat(spec.Field, candidate.Text)
	result := FieldResult{
		Field:     spec.Field,
		Expected:  spec.Expected,
		Extracted: candidate.Text,
		MatchType: "format",
	}
	if !expectedOK {
		result.Diff = "could not parse expected " + quote(spec.Expected)
		return result
	}
	if gotOK && got.equal(expected) {
		result.Score = 1
		result.Pass = true
		return result
	}
	result.Diff = diff(spec.Expected, candidate.Text)
	return result
}

func fuzzyCandidateResult(spec FieldSpec, candidate TextCandidate) FieldResult {
	threshold := spec.Threshold
	if threshold <= 0 {
		threshold = 0.85
	}
	result := FieldResult{
		Field:     spec.Field,
		Expected:  spec.Expected,
		Extracted: candidate.Text,
		MatchType: spec.MatchType,
		Score:     fuzzyScore(spec, candidate),
	}
	if result.MatchType == "" {
		result.MatchType = "fuzzy"
	}
	result.Pass = result.Score >= threshold
	if !result.Pass {
		result.Diff = diff(spec.Expected, candidate.Text)
	}
	return result
}

func missingResult(spec FieldSpec) FieldResult {
	result := FieldResult{Field: spec.Field, Expected: spec.Expected, MatchType: spec.MatchType}
	if spec.MatchType == "" {
		result.MatchType = "fuzzy"
	}
	result.Diff = diff(spec.Expected, "")
	if spec.MatchType == "format" {
		if _, ok := parseFormat(spec.Field, spec.Expected); !ok {
			result.Diff = "could not parse expected " + quote(spec.Expected)
		}
	}
	return result
}

func assignmentWeight(spec FieldSpec, candidate TextCandidate) float64 {
	if candidate.Document && spec.MatchType != "warning" {
		return -1
	}
	base := logicalScore(spec, candidate)
	affinity := semanticAffinity(spec, candidate)
	layout := layoutAffinity(spec, candidate)
	confidence := (candidate.Confidence - 0.5) * 0.05
	lengthPenalty := 0.0
	if candidate.Regions > 4 && !candidate.Document {
		lengthPenalty = 0.01 * float64(candidate.Regions-4)
	}
	documentAffinity := 0.0
	if spec.MatchType == "warning" && candidate.Document {
		documentAffinity = 0.10
	}
	return base + affinity + layout + confidence + documentAffinity - lengthPenalty
}

func logicalScore(spec FieldSpec, candidate TextCandidate) float64 {
	switch spec.MatchType {
	case "exact":
		if NormalizeExact(candidate.Text) == NormalizeExact(spec.Expected) {
			return 1
		}
		return FlexibleSimilarity(spec.Expected, candidate.Text)
	case "format":
		expected, expectedOK := parseFormat(spec.Field, spec.Expected)
		got, gotOK := parseFormat(spec.Field, candidate.Text)
		if !expectedOK || !gotOK {
			return 0
		}
		if got.equal(expected) {
			return 1
		}
		return 0.35
	case "warning":
		score, _ := warningCandidateScore(spec.Expected, candidate.Text)
		return score
	case "name_address":
		return nameAddressScore(spec.Expected, candidate.Text)
	default:
		return fuzzyScore(spec, candidate)
	}
}

func fuzzyScore(spec FieldSpec, candidate TextCandidate) float64 {
	score := FlexibleSimilarity(spec.Expected, candidate.Text)
	if spec.Field == "class_type" {
		score = max(score, classify.ClassTypeScore(spec.Expected, candidate.Text))
	}
	if spec.MatchType == "name_address" || spec.Field == "name_address" {
		score = max(score, nameAddressScore(spec.Expected, candidate.Text))
	}
	return score
}

func nameAddressScore(expected, candidate string) float64 {
	coverage := nameAddressTokenCoverage(expected, candidate, false)
	if coverage < 0.5 {
		return coverage * 0.8
	}
	score := 0.4 + 0.6*coverage
	expectedText := NormalizeFlexible(expected)
	candidateText := NormalizeFlexible(candidate)
	if containsAny(expectedText, producerTerms, roleTerms) && !containsAny(candidateText, producerTerms, roleTerms) {
		score = min(score, 0.80)
	}
	if expectedHasLocation(expected) && !candidateHasExpectedLocation(expected, candidate) {
		score = min(score, 0.80)
	}
	if containsAny(candidateText, formatTerms, warningTerms) {
		score = min(score, 0.84)
	}
	return score
}

func nameAddressTokenCoverage(expected, candidate string, excludeLocationTokens bool) float64 {
	expectedTokens := tokenSet(expected)
	if len(expectedTokens) == 0 {
		return 0
	}
	candidateTokens := tokenSet(candidate)
	if len(candidateTokens) == 0 {
		return 0
	}
	candidateCompact := compactAlnum(candidate)
	matched := 0
	considered := 0
	for token := range expectedTokens {
		if excludeLocationTokens && isLocationToken(token) {
			continue
		}
		considered++
		if candidateTokens[token] ||
			fuzzyTokenPresent(token, candidateTokens, 0.88) ||
			(len(token) >= 3 && strings.Contains(candidateCompact, token)) {
			matched++
		}
	}
	if considered == 0 {
		return 0
	}
	return float64(matched) / float64(considered)
}

func isLocationToken(token string) bool {
	return token == "county" ||
		token == "usa" ||
		token == "united" ||
		token == "states" ||
		len(token) == 2
}

func expectedHasLocation(expected string) bool {
	return mentionsLocation(expected) || len(locationTokens(expected)) >= 2
}

func candidateHasExpectedLocation(expected, candidate string) bool {
	return mentionsLocation(candidate) || nameAddressLocationCoverage(expected, candidate) >= 0.50
}

func nameAddressLocationCoverage(expected, candidate string) float64 {
	expectedTokens := locationTokens(expected)
	if len(expectedTokens) == 0 {
		return 0
	}
	candidateTokens := tokenSet(candidate)
	if len(candidateTokens) == 0 {
		return 0
	}
	candidateCompact := compactAlnum(candidate)
	matched := 0
	for token := range expectedTokens {
		if candidateTokens[token] ||
			fuzzyTokenPresent(token, candidateTokens, 0.88) ||
			(len(token) >= 3 && strings.Contains(candidateCompact, token)) {
			matched++
		}
	}
	return float64(matched) / float64(len(expectedTokens))
}

func locationTokens(value string) map[string]bool {
	tokens := tokenSet(value)
	out := make(map[string]bool)
	for token := range tokens {
		if isLocationEvidenceToken(token) {
			out[token] = true
		}
	}
	return out
}

func isLocationEvidenceToken(token string) bool {
	if token == "county" || token == "usa" || token == "united" || token == "states" {
		return true
	}
	if len(token) == 2 {
		return true
	}
	return !commonNameAddressToken[token] && len(token) >= 4
}

func fuzzyTokenPresent(expected string, candidates map[string]bool, threshold float64) bool {
	for candidate := range candidates {
		if Similarity(expected, candidate) >= threshold {
			return true
		}
	}
	return false
}

func semanticAffinity(spec FieldSpec, candidate TextCandidate) float64 {
	text := NormalizeFlexible(candidate.Text)
	switch spec.Field {
	case "brand":
		score := 0.08
		if containsAny(text, productionTerms, formatTerms, warningTerms) || mentionsLocation(candidate.Text) {
			score -= 0.35
		}
		if containsAny(text, producerTerms, roleTerms) && !containsAny(NormalizeFlexible(spec.Expected), producerTerms, roleTerms) {
			score -= 0.55
		}
		if looksLikeProductName(text) {
			score += 0.18
		}
		return score
	case "class_type":
		score := 0.0
		if classify.HasDesignationSignal(candidate.Text) {
			score += 0.25
		}
		if containsAny(text, formatTerms, warningTerms) || mentionsLocation(candidate.Text) {
			score -= 0.25
		}
		return score
	case "net_contents":
		if _, ok := parseFormat("net_contents", candidate.Text); ok {
			return 0.35 - formatWindowPenalty(candidate)
		}
		return -0.2
	case "abv":
		if _, ok := parseFormat("abv", candidate.Text); ok {
			return 0.35 - formatWindowPenalty(candidate)
		}
		if containsAny(text, formatTerms) {
			return 0.15
		}
		return -0.2
	case "government_warning":
		score := 0.0
		if containsAny(text, warningTerms) {
			score += 0.35
		} else if NormalizeExact(candidate.Text) != NormalizeExact(spec.Expected) {
			score -= 0.35
		}
		if containsAny(text, formatTerms) {
			score -= 0.15
		}
		return score
	case "name_address":
		score := 0.0
		if containsAny(text, producerTerms, roleTerms) {
			score += 0.25
		}
		if mentionsLocation(candidate.Text) {
			score += 0.25
		}
		if containsAny(text, formatTerms, warningTerms) {
			score -= 0.25
		}
		return score
	default:
		if strings.HasPrefix(spec.Field, "foreign_text") {
			return 0.05
		}
		return 0
	}
}

func layoutAffinity(spec FieldSpec, candidate TextCandidate) float64 {
	switch spec.Field {
	case "brand":
		if candidate.Y <= 0.25 || candidate.Y <= 250 {
			return 0.08
		}
	case "government_warning":
		if candidate.Y >= 0.45 || candidate.Y >= 450 {
			return 0.03
		}
	case "net_contents", "abv":
		if candidate.Y >= 0.35 || candidate.Y >= 350 {
			return 0.03
		}
	}
	return 0
}

func formatWindowPenalty(candidate TextCandidate) float64 {
	penalty := 0.0
	if candidate.Regions > 1 {
		penalty += 0.08 * float64(candidate.Regions-1)
	}
	text := NormalizeFlexible(candidate.Text)
	if containsAny(text, productionTerms, warningTerms) || mentionsLocation(candidate.Text) {
		penalty += 0.12
	}
	return penalty
}

var (
	wordLikePattern = regexp.MustCompile(`[[:alpha:]]`)
	productionTerms = []string{
		"distilled", "distilled from", "distilled with", "ingredients", "produced", "bottled",
	}
	producerTerms = []string{
		"bottler", "bottling", "company", "distillery", "distilling", "importer", "inc", "llc", "ltd", "packer", "producer",
	}
	roleTerms = []string{
		"bottled by", "bottled for", "distilled by", "imported by", "packed by", "produced by", "vinted by",
	}
	formatTerms = []string{
		"alc", "vol", "proof", "%", "ml", "cl", "oz", "liter", "litre",
	}
	warningTerms = []string{
		"government", "warning", "surgeon", "pregnancy",
	}
	stateCodePattern       = regexp.MustCompile(`,\s*(?:AL|AK|AZ|AR|CA|CO|CT|DE|FL|GA|HI|ID|IL|IN|IA|KS|KY|LA|ME|MD|MA|MI|MN|MS|MO|MT|NE|NV|NH|NJ|NM|NY|NC|ND|OH|OK|OR|PA|RI|SC|SD|TN|TX|UT|VT|VA|WA|WV|WI|WY)(?:\b|,)`)
	commonNameAddressToken = map[string]bool{
		"and": true, "by": true, "for": true, "in": true, "of": true, "the": true,
		"bottled": true, "distilled": true, "imported": true, "packed": true, "produced": true, "vinted": true,
		"bottler": true, "bottling": true, "company": true, "distillery": true, "distilling": true, "importer": true, "inc": true, "llc": true, "ltd": true, "packer": true, "producer": true,
	}
)

func containsAny(text string, groups ...[]string) bool {
	for _, group := range groups {
		for _, term := range group {
			if strings.Contains(text, term) {
				return true
			}
		}
	}
	return false
}

func looksLikeProductName(text string) bool {
	if !wordLikePattern.MatchString(text) {
		return false
	}
	words := strings.Fields(text)
	return len(words) >= 1 && len(words) <= 5
}

func mentionsLocation(raw string) bool {
	text := NormalizeFlexible(raw)
	return strings.Contains(text, "county") ||
		strings.Contains(text, "united states") ||
		strings.Contains(text, " usa") ||
		stateCodePattern.MatchString(raw)
}

func tokenSet(value string) map[string]bool {
	out := make(map[string]bool)
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		out[string(current)] = true
		current = nil
	}
	for _, r := range NormalizeFlexible(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			current = append(current, r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func compactAlnum(value string) string {
	var out []rune
	for _, r := range NormalizeFlexible(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out = append(out, r)
		}
	}
	return string(out)
}

func quote(value string) string {
	return `"` + value + `"`
}

type weightedPair struct {
	row    int
	col    int
	weight float64
}

func greedyAssignments(weights [][]float64) []int {
	assignments := make([]int, len(weights))
	for i := range assignments {
		assignments[i] = -1
	}

	pairs := make([]weightedPair, 0)
	for row, rowWeights := range weights {
		for col, weight := range rowWeights {
			pairs = append(pairs, weightedPair{row: row, col: col, weight: weight})
		}
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].weight > pairs[j].weight
	})

	usedRows := make([]bool, len(weights))
	usedCols := make(map[int]bool)
	for _, pair := range pairs {
		if pair.weight <= 0 {
			break
		}
		if usedRows[pair.row] || usedCols[pair.col] {
			continue
		}
		assignments[pair.row] = pair.col
		usedRows[pair.row] = true
		usedCols[pair.col] = true
	}
	return assignments
}
