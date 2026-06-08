package match

import "testing"

func TestExactRegulatoryMatchSingleRegion(t *testing.T) {
	r := MatchExact("GOVERNMENT WARNING", poolFrom(reg("GOVERNMENT WARNING", 0.1, 0.6)))
	if !r.Pass {
		t.Fatal("identical regulatory text must pass")
	}
	r = MatchExact("GOVERNMENT WARNING", poolFrom(reg("Government Warning", 0.1, 0.6)))
	if r.Pass {
		t.Fatal("case difference in regulatory text must fail")
	}
}

func TestExactRegulatoryMatchFragmentedAcrossBoxes(t *testing.T) {
	pool := poolFrom(reg("GOVERNMENT", 0.10, 0.60), reg("WARNING", 0.34, 0.60))
	if !MatchExact("GOVERNMENT WARNING", pool).Pass {
		t.Fatal("must join adjacent regions into a window and match exactly")
	}
}

func TestExactRegulatoryMatchMultiLineWindow(t *testing.T) {
	pool := poolFrom(
		reg("GOVERNMENT WARNING: (1) According to the", 0.1, 0.60),
		reg("Surgeon General, women should not drink", 0.1, 0.64),
		reg("alcoholic beverages during pregnancy.", 0.1, 0.68),
	)
	exp := "GOVERNMENT WARNING: (1) According to the Surgeon General, women should not drink alcoholic beverages during pregnancy."
	if !MatchExact(exp, pool).Pass {
		t.Fatal("multi-line warning must match a joined reading-order window")
	}
}

func TestNetContentsFormatFragmented(t *testing.T) {
	if !MatchFormat("net_contents", "750 mL", poolFrom(reg("750 ml", 0.4, 0.8))).Pass {
		t.Fatal("750 mL == 750 ml by value+unit")
	}
	if !MatchFormat("abv", "13.5% ALC/VOL", poolFrom(reg("13.5%", 0.4, 0.85), reg("ALC/VOL", 0.5, 0.85))).Pass {
		t.Fatal("ABV value+unit equal even when split")
	}
	if MatchFormat("net_contents", "750 mL", poolFrom(reg("700 ml", 0.4, 0.8))).Pass {
		t.Fatal("different quantity must fail")
	}
}

func TestBrandFuzzyMatch(t *testing.T) {
	if !MatchFuzzy("Stone's Throw", poolFrom(reg("STONE'S THROW", 0.3, 0.2)), 0.85).Pass {
		t.Fatal("case-insensitive brand should pass")
	}
}

func TestTranslationPresence(t *testing.T) {
	if !MatchPresence("English translation here").Pass {
		t.Fatal("non-empty translation should pass")
	}
	if MatchPresence("   ").Pass {
		t.Fatal("blank translation must fail")
	}
}

func reg(text string, x, y float64) TextRegion {
	return TextRegion{Text: text, Confidence: 0.99, X: x, Y: y, W: 0.12, H: 0.02}
}

func poolFrom(regions ...TextRegion) []TextRegion {
	return regions
}
