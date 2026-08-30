package tengod

import "testing"

func TestEvaluateAggregatesGodsAndCategories(t *testing.T) {
	in := Input{DayStem: "甲", DayElement: "木", Contributions: []Contribution{
		{Source: "stem", Position: "year_stem", Symbol: "甲", TenGod: "比肩", Score: 8},
		{Source: "hidden_stem", Position: "month_branch", Symbol: "己", TenGod: "正财", Score: 18},
		{Source: "hidden_stem", Position: "day_branch", Symbol: "丙", TenGod: "食神", Score: 7.5},
	}, Relations: []Relation{{Code: "stem.combination.甲己", Type: "stem_combination", Element: "土", Score: 10, Positions: []string{"year_stem", "month_stem"}, Symbols: []string{"甲", "己"}, Reason: "existing strength evidence"}}}
	got, err := Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	if got.RuleVersion != RuleVersionV1 || len(got.Gods) != 10 || len(got.Categories) != 5 {
		t.Fatalf("unexpected result: %+v", got)
	}
	if len(got.DominantGods) == 0 || got.DominantGods[0] != "正财" {
		t.Fatalf("dominant=%v", got.DominantGods)
	}
	var wealth *CategoryScore
	for i := range got.Categories {
		if got.Categories[i].Category == "wealth" {
			wealth = &got.Categories[i]
		}
	}
	if wealth == nil || wealth.RawScore != 18 || wealth.RelationAdjustment != 10 || wealth.EffectiveScore != 28 {
		t.Fatalf("wealth=%+v", wealth)
	}
}

func TestEvaluateRejectsUnknownTenGod(t *testing.T) {
	_, err := Evaluate(Input{DayStem: "甲", DayElement: "木", Contributions: []Contribution{{Source: "stem", Position: "year_stem", TenGod: "未知", Score: 8}}})
	if err == nil {
		t.Fatal("expected unknown ten-god error")
	}
}
