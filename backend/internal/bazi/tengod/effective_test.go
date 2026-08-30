package tengod

import "testing"

func TestEvaluateEffectivePreservesRawYinYangSplit(t *testing.T) {
	raw := Result{RuleVersion: RuleVersionV1, DayStem: "甲", DayElement: "木", Gods: []GodScore{
		{Code: "peer.same", Name: "比肩", Category: "peer", RawScore: 30, Visible: true}, {Code: "peer.opposite", Name: "劫财", Category: "peer", RawScore: 10, Rooted: true},
		{Code: "resource.same", Name: "偏印", Category: "resource"}, {Code: "resource.opposite", Name: "正印", Category: "resource"},
		{Code: "output.same", Name: "食神", Category: "output", RawScore: 10}, {Code: "output.opposite", Name: "伤官", Category: "output", RawScore: 10},
		{Code: "wealth.same", Name: "偏财", Category: "wealth", RawScore: 5}, {Code: "wealth.opposite", Name: "正财", Category: "wealth", RawScore: 5},
		{Code: "officer.same", Name: "七杀", Category: "officer", RawScore: 5}, {Code: "officer.opposite", Name: "正官", Category: "officer", RawScore: 5},
	}}
	got, err := EvaluateEffective(EffectiveInput{Raw: raw, ElementRatios: map[string]float64{"木": 40, "火": 20, "土": 15, "金": 10, "水": 15}})
	if err != nil {
		t.Fatal(err)
	}
	if got.RuleVersion != EffectiveRuleVersionV2 {
		t.Fatalf("version=%s", got.RuleVersion)
	}
	byName := map[string]GodScore{}
	for _, g := range got.Gods {
		byName[g.Name] = g
	}
	if byName["比肩"].EffectiveScore != 30 || byName["劫财"].EffectiveScore != 10 {
		t.Fatalf("peer split changed: %+v", byName)
	}
	total := 0.0
	for _, g := range got.Gods {
		total += g.EffectiveScore
	}
	// Resource has no raw yin-yang evidence in this fixture, so its effective
	// power is deliberately left unallocated instead of inventing a split.
	if total != 85 || got.TotalScore != 85 {
		t.Fatalf("effective total=%.2f", total)
	}
}
