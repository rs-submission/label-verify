package classify

import (
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

type Designation struct {
	ID        string
	Commodity string
	Class     string
	Type      string
	Aliases   []string
}

type expectedMatch struct {
	designation Designation
	specificity int
}

var fold = cases.Fold()

var designations = []Designation{
	{
		ID:        "vodka",
		Commodity: "distilled_spirits",
		Class:     "neutral_spirits",
		Type:      "vodka",
		Aliases:   []string{"vodka"},
	},
	{
		ID:        "gin",
		Commodity: "distilled_spirits",
		Class:     "gin",
		Aliases:   []string{"gin", "distilled gin", "redistilled gin", "compounded gin", "london dry gin"},
	},
	{
		ID:        "rum",
		Commodity: "distilled_spirits",
		Class:     "rum",
		Aliases:   []string{"rum", "puerto rico rum", "jamaica rum", "demerara rum"},
	},
	{
		ID:        "tequila",
		Commodity: "distilled_spirits",
		Class:     "agave_spirits",
		Type:      "tequila",
		Aliases:   []string{"tequila"},
	},
	{
		ID:        "mezcal",
		Commodity: "distilled_spirits",
		Class:     "agave_spirits",
		Type:      "mezcal",
		Aliases:   []string{"mezcal"},
	},
	{
		ID:        "brandy",
		Commodity: "distilled_spirits",
		Class:     "brandy",
		Aliases:   []string{"brandy", "fruit brandy", "grape brandy"},
	},
	{
		ID:        "apple_brandy",
		Commodity: "distilled_spirits",
		Class:     "brandy",
		Type:      "apple",
		Aliases:   []string{"apple brandy"},
	},
	{
		ID:        "peach_brandy",
		Commodity: "distilled_spirits",
		Class:     "brandy",
		Type:      "peach",
		Aliases:   []string{"peach brandy"},
	},
	{
		ID:        "cherry_brandy",
		Commodity: "distilled_spirits",
		Class:     "brandy",
		Type:      "cherry",
		Aliases:   []string{"cherry brandy"},
	},
	{
		ID:        "cognac",
		Commodity: "distilled_spirits",
		Class:     "brandy",
		Type:      "cognac",
		Aliases:   []string{"cognac"},
	},
	{
		ID:        "pomace_brandy",
		Commodity: "distilled_spirits",
		Class:     "brandy",
		Type:      "pomace",
		Aliases:   []string{"pomace brandy", "marc"},
	},
	{
		ID:        "cordial_liqueur",
		Commodity: "distilled_spirits",
		Class:     "cordials_and_liqueurs",
		Aliases:   []string{"cordial", "cordials", "liqueur", "liqueurs", "creme de", "crème de"},
	},
	{
		ID:        "whiskey",
		Commodity: "distilled_spirits",
		Class:     "whiskey",
		Aliases:   []string{"whiskey", "whisky"},
	},
	{
		ID:        "bourbon_whiskey",
		Commodity: "distilled_spirits",
		Class:     "whiskey",
		Type:      "bourbon",
		Aliases:   []string{"bourbon", "bourbon whiskey", "bourbon whisky"},
	},
	{
		ID:        "rye_whiskey",
		Commodity: "distilled_spirits",
		Class:     "whiskey",
		Type:      "rye",
		Aliases:   []string{"rye whiskey", "rye whisky"},
	},
	{
		ID:        "wheat_whiskey",
		Commodity: "distilled_spirits",
		Class:     "whiskey",
		Type:      "wheat",
		Aliases:   []string{"wheat whiskey", "wheat whisky"},
	},
	{
		ID:        "malt_whiskey",
		Commodity: "distilled_spirits",
		Class:     "whiskey",
		Type:      "malt",
		Aliases:   []string{"malt whiskey", "malt whisky"},
	},
	{
		ID:        "single_malt_whiskey",
		Commodity: "distilled_spirits",
		Class:     "whiskey",
		Type:      "single_malt",
		Aliases:   []string{"single malt", "single malt whiskey", "single malt whisky", "american single malt", "american single malt whiskey", "american single malt whisky"},
	},
	{
		ID:        "corn_whiskey",
		Commodity: "distilled_spirits",
		Class:     "whiskey",
		Type:      "corn",
		Aliases:   []string{"corn whiskey", "corn whisky"},
	},
	{
		ID:        "blended_whiskey",
		Commodity: "distilled_spirits",
		Class:     "whiskey",
		Type:      "blended",
		Aliases:   []string{"blended whiskey", "blended whisky", "blend of straight whiskies", "blend of straight whiskeys"},
	},
	{
		ID:        "grape_wine",
		Commodity: "wine",
		Class:     "grape_wine",
		Aliases:   []string{"grape wine", "table wine", "light wine", "dessert wine"},
	},
	{
		ID:        "red_wine",
		Commodity: "wine",
		Class:     "grape_wine",
		Type:      "red",
		Aliases:   []string{"red wine"},
	},
	{
		ID:        "white_wine",
		Commodity: "wine",
		Class:     "grape_wine",
		Type:      "white",
		Aliases:   []string{"white wine"},
	},
	{
		ID:        "sparkling_grape_wine",
		Commodity: "wine",
		Class:     "sparkling_grape_wine",
		Aliases:   []string{"sparkling grape wine", "champagne", "crackling wine", "petillant", "pétillant", "frizzante"},
	},
	{
		ID:        "fruit_wine",
		Commodity: "wine",
		Class:     "fruit_wine",
		Aliases:   []string{"fruit wine", "berry wine"},
	},
	{
		ID:        "cider",
		Commodity: "wine",
		Class:     "fruit_wine",
		Type:      "cider",
		Aliases:   []string{"cider"},
	},
	{
		ID:        "perry",
		Commodity: "wine",
		Class:     "fruit_wine",
		Type:      "perry",
		Aliases:   []string{"perry"},
	},
	{
		ID:        "agricultural_wine",
		Commodity: "wine",
		Class:     "agricultural_wine",
		Aliases:   []string{"wine from other agricultural products"},
	},
	{
		ID:        "honey_wine",
		Commodity: "wine",
		Class:     "agricultural_wine",
		Type:      "honey",
		Aliases:   []string{"honey wine", "mead"},
	},
	{
		ID:        "raisin_wine",
		Commodity: "wine",
		Class:     "agricultural_wine",
		Type:      "raisin",
		Aliases:   []string{"raisin wine"},
	},
	{
		ID:        "sake",
		Commodity: "wine",
		Class:     "agricultural_wine",
		Type:      "sake",
		Aliases:   []string{"sake"},
	},
	{
		ID:        "aperitif_wine",
		Commodity: "wine",
		Class:     "aperitif_wine",
		Aliases:   []string{"aperitif wine", "vermouth"},
	},
	{
		ID:        "malt_beverage",
		Commodity: "malt_beverage",
		Class:     "malt_beverage",
		Aliases:   []string{"malt beverage"},
	},
	{
		ID:        "beer",
		Commodity: "malt_beverage",
		Class:     "beer",
		Aliases:   []string{"beer"},
	},
	{
		ID:        "lager",
		Commodity: "malt_beverage",
		Class:     "lager",
		Aliases:   []string{"lager", "lager beer"},
	},
	{
		ID:        "ale",
		Commodity: "malt_beverage",
		Class:     "ale",
		Aliases:   []string{"ale"},
	},
	{
		ID:        "porter",
		Commodity: "malt_beverage",
		Class:     "porter",
		Aliases:   []string{"porter"},
	},
	{
		ID:        "stout",
		Commodity: "malt_beverage",
		Class:     "stout",
		Aliases:   []string{"stout"},
	},
	{
		ID:        "malt_liquor",
		Commodity: "malt_beverage",
		Class:     "malt_liquor",
		Aliases:   []string{"malt liquor"},
	},
	{
		ID:        "wheat_ale",
		Commodity: "malt_beverage",
		Class:     "ale",
		Type:      "wheat",
		Aliases:   []string{"wheat ale", "wheat beer"},
	},
	{
		ID:        "rye_ale",
		Commodity: "malt_beverage",
		Class:     "ale",
		Type:      "rye",
		Aliases:   []string{"rye ale", "rye beer"},
	},
}

func ClassTypeScore(expected, candidate string) float64 {
	matches := expectedDesignations(expected)
	if len(matches) == 0 {
		return 0
	}

	best := 0.0
	for _, match := range matches {
		score := designationScore(match.designation, expected, candidate)
		if score > best {
			best = score
		}
	}
	return best
}

func HasDesignationSignal(value string) bool {
	for _, designation := range designations {
		if aliasScore(designation, value) > 0 {
			return true
		}
	}
	return false
}

func ExpectedEvidenceTermCount(expected string) int {
	matches := expectedDesignations(expected)
	if len(matches) == 0 {
		return 0
	}
	best := 0
	for _, match := range matches {
		if match.specificity > best {
			best = match.specificity
		}
	}
	return best
}

func expectedDesignations(expected string) []expectedMatch {
	var matches []expectedMatch
	for _, designation := range designations {
		best := bestAliasSpecificity(designation, expected)
		if best == 0 {
			continue
		}
		matches = append(matches, expectedMatch{designation: designation, specificity: best})
	}
	if len(matches) == 0 {
		return nil
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].specificity > matches[j].specificity
	})
	best := matches[0].specificity
	out := matches[:0]
	for _, match := range matches {
		if match.specificity == best {
			out = append(out, match)
		}
	}
	return out
}

func designationScore(designation Designation, expected, candidate string) float64 {
	alias := aliasScore(designation, candidate)
	if alias == 0 {
		return 0
	}

	descriptors := expectedDescriptors(designation, expected)
	if len(descriptors) == 0 {
		return alias
	}

	candidateTokens := tokenSet(candidate)
	matched := 0
	for descriptor := range descriptors {
		if candidateTokens[descriptor] {
			matched++
		}
	}
	coverage := float64(matched) / float64(len(descriptors))
	if coverage == 1 {
		return alias
	}
	return min(alias, 0.70+0.20*coverage)
}

func aliasScore(designation Designation, value string) float64 {
	best := 0.0
	for _, alias := range designation.Aliases {
		if phrasePresent(value, alias) {
			best = max(best, 1)
		}
	}
	return best
}

func bestAliasSpecificity(designation Designation, value string) int {
	best := 0
	for _, alias := range designation.Aliases {
		if !phrasePresent(value, alias) {
			continue
		}
		count := len(tokens(alias))
		if count > best {
			best = count
		}
	}
	return best
}

func expectedDescriptors(designation Designation, expected string) map[string]bool {
	aliasTokens := make(map[string]bool)
	for _, alias := range designation.Aliases {
		for _, token := range tokens(alias) {
			aliasTokens[token] = true
		}
	}

	out := make(map[string]bool)
	for _, token := range tokens(expected) {
		if aliasTokens[token] || ignoredDescriptorTokens[token] {
			continue
		}
		out[token] = true
	}
	return out
}

func phrasePresent(value, phrase string) bool {
	phraseTokens := tokens(phrase)
	if len(phraseTokens) == 0 {
		return false
	}
	if len(phraseTokens) == 1 {
		token := phraseTokens[0]
		if tokenSet(value)[token] {
			return true
		}
		return len(token) >= 4 && strings.Contains(compact(value), compact(phrase))
	}
	return strings.Contains(normalize(value), normalize(phrase)) ||
		strings.Contains(compact(value), compact(phrase))
}

func tokens(value string) []string {
	var out []string
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		out = append(out, string(current))
		current = nil
	}
	for _, r := range normalize(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func tokenSet(value string) map[string]bool {
	out := make(map[string]bool)
	for _, token := range tokens(value) {
		out[token] = true
	}
	return out
}

func compact(value string) string {
	var out []rune
	for _, r := range normalize(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, r)
		}
	}
	return string(out)
}

func normalize(value string) string {
	value = norm.NFKC.String(value)
	value = fold.String(value)
	return strings.Join(strings.Fields(value), " ")
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

var ignoredDescriptorTokens = map[string]bool{
	"class": true,
	"type":  true,
	"and":   true,
	"or":    true,
	"of":    true,
	"the":   true,
	"with":  true,
}
