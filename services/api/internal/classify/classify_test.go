package classify

import "testing"

func TestClassTypeScoreUsesTTBAliases(t *testing.T) {
	tests := []struct {
		name      string
		expected  string
		candidate string
		wantMin   float64
	}{
		{
			name:      "liqueur cordial",
			expected:  "Cherry Liqueur/Cordial",
			candidate: "POM CHERRY CORDIAL",
			wantMin:   0.95,
		},
		{
			name:      "whiskey spelling",
			expected:  "Rye Whiskey",
			candidate: "POM RYE WHISKY",
			wantMin:   0.95,
		},
		{
			name:      "single malt display",
			expected:  "Single Malt Whiskey",
			candidate: "POM SINGLE MALT",
			wantMin:   0.95,
		},
		{
			name:      "mead honey wine",
			expected:  "Honey Wine",
			candidate: "MEAD",
			wantMin:   0.95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassTypeScore(tt.expected, tt.candidate); got < tt.wantMin {
				t.Fatalf("ClassTypeScore(%q, %q)=%f want >= %f", tt.expected, tt.candidate, got, tt.wantMin)
			}
		})
	}
}

func TestClassTypeScoreRejectsSpecificFamilyMismatches(t *testing.T) {
	tests := []struct {
		expected  string
		candidate string
	}{
		{expected: "Vodka", candidate: "STRAIGHT RYE WHISKEY"},
		{expected: "Red Wine", candidate: "WHITE WINE"},
		{expected: "Single Malt Whiskey", candidate: "100% MALTED BARLEY"},
		{expected: "Cherry Liqueur/Cordial", candidate: "POM CORDIAL"},
		{expected: "Apple Brandy", candidate: "CHERRY BRANDY"},
		{expected: "Cider", candidate: "PERRY"},
		{expected: "Ale", candidate: "LAGER"},
	}

	for _, tt := range tests {
		t.Run(tt.expected+" vs "+tt.candidate, func(t *testing.T) {
			if got := ClassTypeScore(tt.expected, tt.candidate); got >= 0.95 {
				t.Fatalf("ClassTypeScore(%q, %q)=%f want < 0.95", tt.expected, tt.candidate, got)
			}
		})
	}
}

func TestHasDesignationSignalAvoidsCommonSubstrings(t *testing.T) {
	if HasDesignationSignal("PALE INGREDIENTS") {
		t.Fatal("ale should not be detected inside pale")
	}
	if !HasDesignationSignal("POMVODKA") {
		t.Fatal("compact OCR product text should expose vodka signal")
	}
}
