package strengthv2

import "testing"

func TestEvaluateContinuousStrength(t *testing.T) {
	in := Input{DayElement: "木", MonthBranch: "寅", SeasonCoefficient: 1.55, SeasonProgress: .4, RootPower: 1.8, Categories: CategoryPower{Peer: 35, Resource: 25, Output: 15, Wealth: 10, Officer: 15}, ElementRatios: map[string]float64{"木": 35, "水": 25, "火": 15, "土": 10, "金": 15}}
	got, err := Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	if got.RuleVersion != RuleVersionV2 || got.Score <= 15 || got.DeLing != 100 || got.DeDi <= 0 || got.Confidence <= 0 {
		t.Fatalf("unexpected result: %+v", got)
	}
	if len(got.Trace) != 3 || len(got.Patterns) == 0 {
		t.Fatalf("trace/pattern missing: %+v", got)
	}
}

func TestEvaluateFollowCandidateAndRootRejection(t *testing.T) {
	base := Input{DayElement: "木", MonthBranch: "申", SeasonCoefficient: .55, SeasonProgress: .5, RootPower: 0, Categories: CategoryPower{Peer: 1, Resource: 1, Output: 5, Wealth: 72, Officer: 21}, ElementRatios: map[string]float64{"土": 72, "金": 21, "火": 5, "木": 1, "水": 1}}
	got, err := Evaluate(base)
	if err != nil {
		t.Fatal(err)
	}
	if got.Score > -75 {
		t.Fatalf("expected extreme weakness: %+v", got)
	}
	if !got.Patterns[0].Matched || got.Patterns[0].Subtype != "follow_wealth" {
		t.Fatalf("follow candidate=%+v", got.Patterns[0])
	}
	base.RootPower = 1
	got, err = Evaluate(base)
	if err != nil {
		t.Fatal(err)
	}
	if got.Patterns[0].Matched || len(got.Patterns[0].RejectedBy) == 0 {
		t.Fatalf("root must reject following: %+v", got.Patterns[0])
	}
}
