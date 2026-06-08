package match

import "testing"

func TestNormalizeFoldsCaseAndUnicode(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Stone's Throw", "stone's throw"},
		{"Château", "château"},
		{"GROSSE", "grosse"},
		{"  Spaced  Out ", "spaced out"},
		{"ＳＴＯＮＥ", "stone"},
		{"ﬁzz", "fizz"},
	}
	for _, c := range cases {
		if got := NormalizeFlexible(c.in); got != c.want {
			t.Fatalf("NormalizeFlexible(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeExactPreservesCaseAndDoesNotFoldCompatibility(t *testing.T) {
	if NormalizeExact("GOVERNMENT WARNING") != "GOVERNMENT WARNING" {
		t.Fatal("exact normalization must preserve case")
	}
	if NormalizeExact("GOVERNMENT WARNING ") != "GOVERNMENT WARNING" {
		t.Fatal("exact normalization should trim")
	}
	if NormalizeExact("ＷＡＲＮＩＮＧ") == "WARNING" {
		t.Fatal("exact normalization must not apply NFKC compatibility folding")
	}
}
