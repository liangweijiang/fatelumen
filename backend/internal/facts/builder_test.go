package facts

import (
	"encoding/json"
	"testing"

	"fatelumen/backend/internal/model"
)

func sampleBuildInput() BuildInput {
	score := model.StrengthAnalysis{RuleVersion: "strength-rule-v1", DayElement: "木", MonthScore: 40, SupportScore: 82, RestraintScore: 18, SupportRatio: .82, RootLevel: "strong", Pattern: "normal", Contributions: []model.StrengthContribution{{Code: "month_command", Source: "month_command", Position: "month_branch", Symbol: "寅", Element: "木", Category: "support", Score: 40}}, Relations: []model.StrengthRelation{{Code: "stem_combination", Type: "stem_combination", Positions: []string{"year_stem", "month_stem"}, Symbols: []string{"甲", "己"}, Element: "土", Score: 10, Transformed: true, Reason: "month command supports transformation"}, {Code: "branch_clash", Type: "clash", Positions: []string{"day_branch", "hour_branch"}, Symbols: []string{"寅", "申"}, Score: -16, Reason: "clash adjustment"}}}
	tenGod := model.TenGodAnalysis{RuleVersion: "ten-god-rule-v1", DayStem: "甲", DayElement: "木", TotalScore: 30, DominantGods: []string{"比肩"}, SecondaryGods: []string{"正财"}, MissingGods: []string{"正官"}, VisibleGods: []string{"比肩"}, RootedGods: []string{"正财"}, Concentration: .6667, Gods: []model.TenGodScore{{Code: "peer.same", Name: "比肩", Category: "peer", RawScore: 20, EffectiveScore: 20, Ratio: .6667, Rank: 1, Visible: true}, {Code: "wealth.opposite", Name: "正财", Category: "wealth", RawScore: 10, EffectiveScore: 10, Ratio: .3333, Rank: 2, Rooted: true}}, Categories: []model.TenGodCategoryScore{{Category: "peer", RawScore: 20, EffectiveScore: 20, Ratio: .6667, Rank: 1}, {Category: "wealth", RawScore: 10, RelationAdjustment: 10, EffectiveScore: 20, Ratio: .3333, Rank: 2}}, Evidence: []model.TenGodEvidence{{Code: "ten-god-rule-v1.occurrence", Source: "stem", Position: "year_stem", Symbols: []string{"甲"}, TenGod: "比肩", Category: "peer", RawScore: 20, Reason: "fixture"}}}
	elementPower := model.ElementPowerAnalysis{RuleVersion: "bazi-power-v2.0", Season: model.ElementSeasonAnalysis{Branch: "寅", NextBranch: "卯", Progress: .5, Coefficients: model.ElementPowerVector{Wood: 1.6}}, RawPower: model.ElementPowerVector{Wood: 2}, SeasonalPower: model.ElementPowerVector{Wood: 3.2}, EffectivePower: model.ElementPowerVector{Wood: 2.7}, EffectiveRatio: model.ElementPowerVector{Wood: 40, Fire: 20, Earth: 15, Metal: 10, Water: 15}, Contributions: []model.ElementPowerContribution{{Code: "stem", Position: "year_stem", Symbol: "甲", Element: "木", RawPower: .85, SeasonCoefficient: 1.6, VisibilityMultiplier: 1, SeasonalPower: 1.36}}, Structures: []model.ElementStructureEvidence{{Code: "stem.combination.甲己", Type: "combination", TargetElement: "土", State: "partial", Positions: []string{"year_stem", "month_stem"}, Symbols: []string{"甲", "己"}, Confidence: 60, TransferRate: .35}}}
	tenGodEffective := tenGod
	tenGodEffective.RuleVersion = "ten-god-effective-v2.0"
	strengthV2 := model.StrengthV2Analysis{RuleVersion: "day-master-strength-v2.0", Level: "slightly_strong", BaseScore: 30, Score: 25, Confidence: .8, Support: 60, Pressure: 40, SupportRatio: .6, DeLing: 80, DeDi: 60, DeShi: 55, Patterns: []model.StrengthPatternCandidate{{Type: "normal", Matched: true, Confidence: .8, Evidence: []string{"fixture"}}}, Trace: []model.StrengthV2Trace{{Rule: "day-master-strength-v2.0.base", Result: "slightly_strong", Reason: "fixture", Score: 25}}}
	climate := model.ClimateAnalysis{RuleVersion: "climate-v1.0", Temperature: -30, Moisture: 20}
	pattern := model.PatternAnalysis{RuleVersion: "pattern-review-v1.0", Primary: "normal"}
	disease := model.DiseaseAnalysis{RuleVersion: "disease-v1.0"}
	mediation := model.MediationAnalysis{RuleVersion: "mediation-v1.0"}
	usefulGod := model.UsefulGodAnalysis{RuleVersion: "useful-god-v1.0", Primary: "火", Confidence: .8, Candidates: []model.UsefulGodCandidate{{Element: "火", Role: "primary_useful", Score: 80, Valid: true, Breakdown: model.UsefulGodScoreBreakdown{MarginalUtility: 20}}}}
	return BuildInput{Input: model.ReportInputSnapshot{Year: 1990, Month: 1, Day: 1, Hour: 12, TimezoneID: "Asia/Shanghai", SchemaVersion: model.ReportInputSchemaVersion}, Chart: model.ChartSnapshot{ChartHash: "chart-a", ChartSchemaVersion: model.ChartSnapshotVersion, EngineVersion: "engine-v1", LunarGoVersion: "v1.4.6", Data: model.ChartData{DayMaster: model.DayMaster{Stem: "甲", Element: "木", YinYang: "阳"}, Strength: model.Strength{Level: "strong", Score: 82, Analysis: &score}, TenGodAnalysis: &tenGod, TenGodEffective: &tenGodEffective, ElementPower: &elementPower, StrengthV2: &strengthV2, Climate: &climate, Pattern: &pattern, Disease: &disease, Mediation: &mediation, UsefulGod: &usefulGod, LuckCycles: []model.LuckCycle{}, AnnualFortunes: []model.AnnualFortune{}}}, Versions: model.InterpretationFactVersions{PromptVersion: "full-v1"}}
}

func TestBuildProjectsStrengthWithoutRecalculation(t *testing.T) {
	f, err := Build(sampleBuildInput())
	if err != nil {
		t.Fatal(err)
	}
	if f.DayMasterStrength.AnalysisV2 == nil || f.DayMasterStrength.AnalysisV2.Score != 25 || f.DayMasterStrength.Level != "slightly_strong" {
		t.Fatalf("strength changed: %+v", f.DayMasterStrength)
	}
	if len(f.StemRelations) != 1 || len(f.BranchRelations) != 0 {
		t.Fatalf("relations not projected: stems=%d branches=%d", len(f.StemRelations), len(f.BranchRelations))
	}
	if f.Versions.RuleSetVersion != "day-master-strength-v2.0" {
		t.Fatalf("version=%s", f.Versions.RuleSetVersion)
	}
	if f.Versions.TenGodRuleVersion != "ten-god-effective-v2.0" || len(f.TenGodStructure) == 0 {
		t.Fatalf("ten-god facts missing: version=%s facts=%d", f.Versions.TenGodRuleVersion, len(f.TenGodStructure))
	}
	if f.Versions.ElementPowerRuleVersion != "bazi-power-v2.0" || len(f.ElementStrength) != 5 {
		t.Fatalf("element-power facts missing: version=%s facts=%d", f.Versions.ElementPowerRuleVersion, len(f.ElementStrength))
	}
	if f.FactsHash == "" {
		t.Fatal("facts hash is empty")
	}
}

func TestBuildFactsHashChangesWithTenGodRule(t *testing.T) {
	a, err := Build(sampleBuildInput())
	if err != nil {
		t.Fatal(err)
	}
	in := sampleBuildInput()
	in.Chart.Data.TenGodEffective.RuleVersion = "ten-god-effective-v2.1"
	b, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if a.FactsHash == b.FactsHash {
		t.Fatal("facts hash must change with ten-god rule version")
	}
}

func TestBuildFactsHashChangesWithElementPowerRule(t *testing.T) {
	a, err := Build(sampleBuildInput())
	if err != nil {
		t.Fatal(err)
	}
	in := sampleBuildInput()
	in.Chart.Data.ElementPower.RuleVersion = "bazi-power-v2.1"
	b, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if a.FactsHash == b.FactsHash {
		t.Fatal("facts hash must change with element-power rule version")
	}
}

func TestBuildFactsHashChangesWithV2CRule(t *testing.T) {
	a, err := Build(sampleBuildInput())
	if err != nil {
		t.Fatal(err)
	}
	in := sampleBuildInput()
	in.Chart.Data.Climate.RuleVersion = "climate-v1.1"
	b, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if a.FactsHash == b.FactsHash {
		t.Fatal("facts hash did not change with v2-c rule version")
	}
}

func TestBuildFactsHashChangesWithUsefulGodRule(t *testing.T) {
	a, err := Build(sampleBuildInput())
	if err != nil {
		t.Fatal(err)
	}
	in := sampleBuildInput()
	in.Chart.Data.UsefulGod.RuleVersion = "useful-god-v1.1"
	b, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if a.FactsHash == b.FactsHash {
		t.Fatal("facts hash did not change with useful-god rule")
	}
}

func TestBuildFactsHashStable(t *testing.T) {
	a, err := Build(sampleBuildInput())
	if err != nil {
		t.Fatal(err)
	}
	b, err := Build(sampleBuildInput())
	if err != nil {
		t.Fatal(err)
	}
	if a.FactsHash != b.FactsHash {
		t.Fatalf("unstable hash: %s != %s", a.FactsHash, b.FactsHash)
	}
	payload, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) == 0 {
		t.Fatal("empty json")
	}
}

func TestBuildFactsHashChangesWithStrengthRule(t *testing.T) {
	aIn := sampleBuildInput()
	a, err := Build(aIn)
	if err != nil {
		t.Fatal(err)
	}
	bIn := sampleBuildInput()
	bIn.Chart.Data.StrengthV2.RuleVersion = "day-master-strength-v2.1"
	b, err := Build(bIn)
	if err != nil {
		t.Fatal(err)
	}
	if a.FactsHash == b.FactsHash {
		t.Fatal("facts hash must change with strength rule version")
	}
}

func TestBuildFactsHashDoesNotChangeWithPromptVersion(t *testing.T) {
	aIn := sampleBuildInput()
	a, err := Build(aIn)
	if err != nil {
		t.Fatal(err)
	}
	bIn := sampleBuildInput()
	bIn.Versions.PromptVersion = "full-v2"
	b, err := Build(bIn)
	if err != nil {
		t.Fatal(err)
	}
	if a.FactsHash != b.FactsHash {
		t.Fatal("prompt version must not alter deterministic facts hash")
	}
}

func TestBuildRejectsLegacyChartWithoutStrengthTrace(t *testing.T) {
	in := sampleBuildInput()
	in.Chart.Data.Strength.Analysis = nil
	if _, err := Build(in); err == nil {
		t.Fatal("expected missing analysis error")
	}
}

func TestBuildCreatesIndependentSnapshot(t *testing.T) {
	in := sampleBuildInput()
	f, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	in.Chart.Data.Strength.Analysis.Contributions[0].Score = 999
	if f.Chart.Data.Strength.Analysis.Contributions[0].Score == 999 {
		t.Fatal("facts chart shares mutable contribution slice with input")
	}
}
