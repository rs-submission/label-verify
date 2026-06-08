package match

import "testing"

func TestSimilarityScore(t *testing.T) {
	if s := Similarity("government warning", "government warnng"); s < 0.9 {
		t.Fatalf("close strings should score high, got %f", s)
	}
	if s := Similarity("merlot", "cabernet"); s > 0.5 {
		t.Fatalf("different strings should score low, got %f", s)
	}
	if s := Similarity("Café", "Café"); s != 1.0 {
		t.Fatalf("identical multibyte strings must be 1.0, got %f", s)
	}
}
