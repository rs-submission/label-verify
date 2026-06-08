package match

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

var caseFold = cases.Fold()

// NormalizeExact preserves regulatory text shape while still canonicalizing accents.
func NormalizeExact(s string) string {
	return strings.TrimSpace(norm.NFC.String(s))
}

// NormalizeFlexible folds compatibility forms and case for non-regulatory fields.
func NormalizeFlexible(s string) string {
	s = norm.NFKC.String(s)
	s = caseFold.String(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}
