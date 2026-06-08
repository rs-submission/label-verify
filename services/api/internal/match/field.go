package match

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type FieldResult struct {
	Field              string
	Expected           string
	Extracted          string
	MatchType          string
	Score              float64
	Pass               bool
	Diff               string
	ReviewSource       string  `json:",omitempty"`
	ReviewDecision     string  `json:",omitempty"`
	ReviewReason       string  `json:",omitempty"`
	ReviewConfidence   float64 `json:",omitempty"`
	ReviewAccepted     bool    `json:",omitempty"`
	ReviewExtracted    string  `json:",omitempty"`
	DeterministicScore float64 `json:",omitempty"`
}

func MatchExact(expected string, pool []TextRegion) FieldResult {
	windows := AssembleWindows(pool, 0.25)
	expectedNorm := NormalizeExact(expected)
	result := FieldResult{Expected: expected, MatchType: "exact", Score: 0}
	var best string
	var bestScore float64

	for _, window := range windows {
		got := NormalizeExact(window)
		if got == expectedNorm {
			result.Extracted = window
			result.Score = 1
			result.Pass = true
			return result
		}
		score := FlexibleSimilarity(expected, window)
		if score > bestScore {
			bestScore = score
			best = window
		}
	}

	result.Extracted = best
	result.Score = bestScore
	result.Diff = diff(expected, best)
	return result
}

func MatchFormat(field, expected string, pool []TextRegion) FieldResult {
	windows := AssembleWindows(pool, 0.25)
	exp, ok := parseFormat(field, expected)
	result := FieldResult{Field: field, Expected: expected, MatchType: "format"}
	if !ok {
		result.Diff = fmt.Sprintf("could not parse expected %q", expected)
		return result
	}

	for _, window := range windows {
		got, ok := parseFormat(field, window)
		if !ok {
			continue
		}
		result.Extracted = window
		if got.equal(exp) {
			result.Score = 1
			result.Pass = true
			return result
		}
		result.Score = 0
		result.Diff = diff(expected, window)
		return result
	}

	result.Diff = diff(expected, "")
	return result
}

func MatchFuzzy(expected string, pool []TextRegion, threshold float64) FieldResult {
	windows := AssembleWindows(pool, 0.25)
	result := FieldResult{Expected: expected, MatchType: "fuzzy"}
	expectedNorm := NormalizeFlexible(expected)

	for _, window := range windows {
		score := max(Similarity(expectedNorm, NormalizeFlexible(window)), FlexibleSimilarity(expected, window))
		if score > result.Score {
			result.Score = score
			result.Extracted = window
		}
	}

	result.Pass = result.Score >= threshold
	if !result.Pass {
		result.Diff = diff(expected, result.Extracted)
	}
	return result
}

func MatchPresence(value string) FieldResult {
	pass := strings.TrimSpace(value) != ""
	result := FieldResult{Expected: value, Extracted: value, MatchType: "presence", Pass: pass}
	if pass {
		result.Score = 1
	} else {
		result.Diff = "required value is blank"
	}
	return result
}

type parsedFormat struct {
	value float64
	unit  string
}

func (p parsedFormat) equal(other parsedFormat) bool {
	return p.value == other.value && p.unit == other.unit
}

var (
	abvPattern = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*%\s*(?:alc\s*/\s*vol)?`)
	netPattern = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(ml|mL|l|cl|oz|fl\s*oz)`)
)

func parseFormat(field, value string) (parsedFormat, bool) {
	normalized := NormalizeFlexible(value)
	pattern := netPattern
	if field == "abv" {
		pattern = abvPattern
	}
	matches := pattern.FindStringSubmatch(normalized)
	if len(matches) < 2 {
		return parsedFormat{}, false
	}
	n, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return parsedFormat{}, false
	}
	unit := ""
	if field == "abv" {
		unit = "%alc/vol"
	} else if len(matches) >= 3 {
		unit = strings.ReplaceAll(matches[2], " ", "")
	}
	return parsedFormat{value: n, unit: unit}, true
}

func diff(expected, found string) string {
	return fmt.Sprintf("expected %q, found %q", expected, found)
}
