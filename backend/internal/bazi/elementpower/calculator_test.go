package elementpower

import (
	"math"
	"testing"
)

func sampleInput() Input {
	return Input{Year: Pillar{"庚", "午"}, Month: Pillar{"丙", "子"}, Day: Pillar{"丙", "寅"}, Hour: Pillar{"甲", "午"}, SeasonProgress: .6}
}

func TestEvaluateProducesTraceablePowerStages(t *testing.T) {
	got, err := Evaluate(sampleInput(), RuleV1())
	if err != nil {
		t.Fatal(err)
	}
	if got.RuleVersion != RuleVersionV1 {
		t.Fatalf("version=%s", got.RuleVersion)
	}
	if len(got.Contributions) != 12 {
		t.Fatalf("contributions=%d", len(got.Contributions))
	}
	if len(got.Interactions) != 20 {
		t.Fatalf("interactions=%d", len(got.Interactions))
	}
	if len(got.Trace) != 4 {
		t.Fatalf("trace=%d", len(got.Trace))
	}
	rawTotal := got.RawPower.Wood + got.RawPower.Fire + got.RawPower.Earth + got.RawPower.Metal + got.RawPower.Water
	if math.Abs(rawTotal-9.15) > .0001 {
		t.Fatalf("raw total=%.4f want 9.15", rawTotal)
	}
	ratioTotal := got.EffectiveRatio.Wood + got.EffectiveRatio.Fire + got.EffectiveRatio.Earth + got.EffectiveRatio.Metal + got.EffectiveRatio.Water
	if math.Abs(ratioTotal-100) > .001 {
		t.Fatalf("ratio total=%.4f", ratioTotal)
	}
	if got.Season.Branch != "子" || got.Season.NextBranch != "丑" || got.Season.Coefficients.Wood != 0.94 {
		t.Fatalf("season=%+v", got.Season)
	}
	if got.RootPower <= 0 || len(got.Roots) == 0 {
		t.Fatalf("roots=%+v", got.Roots)
	}
}

func TestRuleV2AppliesStructuresAndClearsPendingWarning(t *testing.T) {
	in := sampleInput()
	in.LongLifeStages = map[string]string{"year": "临官", "month": "帝旺", "day": "长生", "hour": "墓"}
	got, err := Evaluate(in, RuleV2())
	if err != nil {
		t.Fatal(err)
	}
	if got.RuleVersion != RuleVersionV2 || len(got.Structures) == 0 {
		t.Fatalf("structure result missing: %+v", got)
	}
	for _, w := range got.Warnings {
		if w == "structural_transformations_pending_v2_b" {
			t.Fatal("pending warning must be removed in V2")
		}
	}
}

func TestRuleV2DistinguishesCombinationAndClashStates(t *testing.T) {
	in := Input{Year: Pillar{"甲", "子"}, Month: Pillar{"己", "辰"}, Day: Pillar{"戊", "午"}, Hour: Pillar{"丙", "戌"}, SeasonProgress: .5}
	got, err := Evaluate(in, RuleV2())
	if err != nil {
		t.Fatal(err)
	}
	foundCombination, foundClash := false, false
	for _, s := range got.Structures {
		if s.Type == "combination" {
			foundCombination = true
			if s.Confidence <= 0 || s.State == "" {
				t.Fatalf("invalid combination: %+v", s)
			}
		}
		if s.Type == "clash" {
			foundClash = true
			if s.State != "balanced_clash" && s.State != "strong_weak_clash" {
				t.Fatalf("invalid clash: %+v", s)
			}
		}
	}
	if !foundCombination || !foundClash {
		t.Fatalf("missing structures: %+v", got.Structures)
	}
}

func TestEvaluateIsDeterministic(t *testing.T) {
	a, err := Evaluate(sampleInput(), RuleV1())
	if err != nil {
		t.Fatal(err)
	}
	b, err := Evaluate(sampleInput(), RuleV1())
	if err != nil {
		t.Fatal(err)
	}
	if a.EffectivePower != b.EffectivePower || a.EffectiveRatio != b.EffectiveRatio || a.RootPower != b.RootPower {
		t.Fatalf("unstable result: %+v %+v", a, b)
	}
}

func TestEvaluateRejectsInvalidSeasonProgress(t *testing.T) {
	in := sampleInput()
	in.SeasonProgress = 1.1
	if _, err := Evaluate(in, RuleV1()); err == nil {
		t.Fatal("expected progress validation error")
	}
}
