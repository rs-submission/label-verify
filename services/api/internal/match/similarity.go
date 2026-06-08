package match

import (
	"unicode"

	"github.com/agnivade/levenshtein"
)

// Similarity returns 1 minus normalized rune-level edit distance.
func Similarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	ra, rb := []rune(a), []rune(b)
	maxLen := len(ra)
	if len(rb) > maxLen {
		maxLen = len(rb)
	}
	if maxLen == 0 {
		return 1.0
	}
	d := levenshtein.ComputeDistance(a, b)
	return 1.0 - float64(d)/float64(maxLen)
}

func FlexibleSimilarity(a, b string) float64 {
	aNorm := NormalizeFlexible(a)
	bNorm := NormalizeFlexible(b)
	score := Similarity(aNorm, bNorm)

	aCompact := compactLettersAndDigits(aNorm)
	bCompact := compactLettersAndDigits(bNorm)
	if aCompact != "" && bCompact != "" {
		score = max(score, Similarity(aCompact, bCompact))
	}
	return score
}

func compactLettersAndDigits(value string) string {
	var out []rune
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, r)
		}
	}
	return string(out)
}
