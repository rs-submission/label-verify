package match

import "testing"

func TestVerdictConsistentWhenAllPass(t *testing.T) {
	v := Aggregate([]FieldResult{{Pass: true, Score: 1}, {Pass: true, Score: 1}})
	if v.Status != "consistent" {
		t.Fatal("all-pass => consistent")
	}
}

func TestVerdictFlaggedOnAnyRegulatoryFail(t *testing.T) {
	v := Aggregate([]FieldResult{{Pass: false, MatchType: "exact", Field: "government_warning"}})
	if v.Status != "flagged" {
		t.Fatal("regulatory fail => flagged")
	}
}
