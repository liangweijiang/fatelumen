package facts

import (
	"encoding/json"
	"fmt"
	"strings"

	"fatelumen/backend/internal/bazi/basedata"
	"fatelumen/backend/internal/model"
	hashutil "fatelumen/backend/internal/pkg/hash"
)

const (
	dayMasterStrengthCode = "day_master.strength"
	strengthModule        = "strength"
	tenGodSummaryCode     = "ten_god.structure"
	tenGodModule          = "ten_god"
)

type BuildInput struct {
	Input           model.ReportInputSnapshot
	TimeCalculation model.TimeCalculationSnapshot
	Chart           model.ChartSnapshot
	Versions        model.InterpretationFactVersions
}

// Build creates the immutable deterministic facts currently available for a
// full report. It never recalculates stem/branch relationships: those are
// projected from the versioned strength analysis attached to ChartData.
func Build(in BuildInput) (model.InterpretationFacts, error) {
	chart, err := cloneChartSnapshot(in.Chart)
	if err != nil {
		return model.InterpretationFacts{}, err
	}
	in.Chart = chart
	analysis := in.Chart.Data.Strength.Analysis
	if analysis == nil {
		return model.InterpretationFacts{}, fmt.Errorf("chart strength analysis is required")
	}
	if analysis.RuleVersion == "" {
		return model.InterpretationFacts{}, fmt.Errorf("strength rule version is required")
	}
	tenGodAnalysis := in.Chart.Data.TenGodAnalysis
	if tenGodAnalysis == nil {
		return model.InterpretationFacts{}, fmt.Errorf("chart ten-god analysis is required")
	}
	if tenGodAnalysis.RuleVersion == "" {
		return model.InterpretationFacts{}, fmt.Errorf("ten-god rule version is required")
	}
	elementPower := in.Chart.Data.ElementPower
	if elementPower == nil || elementPower.RuleVersion == "" {
		return model.InterpretationFacts{}, fmt.Errorf("chart element-power analysis and rule version are required")
	}
	strengthV2 := in.Chart.Data.StrengthV2
	if strengthV2 == nil || strengthV2.RuleVersion == "" {
		return model.InterpretationFacts{}, fmt.Errorf("chart strength-v2 analysis and rule version are required")
	}
	tenGodEffective := in.Chart.Data.TenGodEffective
	if tenGodEffective == nil || tenGodEffective.RuleVersion == "" {
		return model.InterpretationFacts{}, fmt.Errorf("chart effective ten-god analysis and rule version are required")
	}
	climate, pattern, disease, mediation := in.Chart.Data.Climate, in.Chart.Data.Pattern, in.Chart.Data.Disease, in.Chart.Data.Mediation
	if climate == nil || pattern == nil || disease == nil || mediation == nil || climate.RuleVersion == "" || pattern.RuleVersion == "" || disease.RuleVersion == "" || mediation.RuleVersion == "" {
		return model.InterpretationFacts{}, fmt.Errorf("complete v2-c deterministic analyses and rule versions are required")
	}
	usefulGod := in.Chart.Data.UsefulGod
	if usefulGod == nil || usefulGod.RuleVersion == "" {
		return model.InterpretationFacts{}, fmt.Errorf("v2-d useful-god analysis and rule version are required")
	}
	normalizeVersions(&in, strengthV2.RuleVersion, tenGodEffective.RuleVersion, elementPower.RuleVersion, climate.RuleVersion, pattern.RuleVersion, disease.RuleVersion, mediation.RuleVersion, usefulGod.RuleVersion)

	score := strengthV2.Score
	evidence := buildStrengthV2Evidence(*strengthV2)
	stemRelations, branchRelations := projectV2Relations(elementPower.Structures)
	level := strengthV2.Level
	patterns := buildPatternFactsV2(*strengthV2)
	tenGodFacts := buildTenGodFacts(*tenGodEffective)
	elementFacts := buildElementFacts(*elementPower)
	ruleMatches := append(buildStrengthV2RuleMatches(*strengthV2, elementPower.Structures), buildTenGodRuleMatches(*tenGodEffective)...)
	warnings := append([]string{}, strengthV2.Warnings...)
	warnings = append(warnings, elementPower.Warnings...)
	warnings = append(warnings, climate.Warnings...)
	warnings = append(warnings, pattern.Warnings...)
	warnings = append(warnings, disease.Warnings...)
	warnings = append(warnings, mediation.Warnings...)
	warnings = append(warnings, usefulGod.Warnings...)
	favorable := buildUsefulGodFact(*usefulGod)

	f := model.InterpretationFacts{
		Input: in.Input, TimeCalculation: in.TimeCalculation, Chart: in.Chart,
		ElementStrength: elementFacts,
		DayMasterStrength: model.DayMasterStrengthFact{
			Code: dayMasterStrengthCode, Level: level, Score: &score,
			Values: strengthV2Values(*strengthV2), Evidence: evidence, Conflicts: []string{}, Analysis: *analysis, AnalysisV2: strengthV2,
		},
		TenGodStructure: tenGodFacts, StemRelations: stemRelations, BranchRelations: branchRelations,
		PatternCandidates: patterns,
		Climate:           climate, Pattern: pattern, Disease: disease, Mediation: mediation, UsefulGod: usefulGod,
		FavorableElements: favorable,
		LuckCycles:        append([]model.LuckCycle{}, in.Chart.Data.LuckCycles...), AnnualFortunes: append([]model.AnnualFortune{}, in.Chart.Data.AnnualFortunes...),
		ChapterFacts: []model.ChapterFacts{{ChapterKey: "structure", FactCodes: append([]string{dayMasterStrengthCode, tenGodSummaryCode}, elementFactCodes(elementFacts)...)}, {ChapterKey: "day_master", FactCodes: append([]string{dayMasterStrengthCode}, elementFactCodes(elementFacts)...)}, {ChapterKey: "ten_gods", FactCodes: tenGodFactCodes(tenGodFacts)}},
		RuleMatches:  ruleMatches, Warnings: warnings, Versions: in.Versions,
	}
	hash, err := CalculateHash(f)
	if err != nil {
		return model.InterpretationFacts{}, err
	}
	f.FactsHash = hash
	return f, nil
}

func cloneChartSnapshot(in model.ChartSnapshot) (model.ChartSnapshot, error) {
	payload, err := json.Marshal(in)
	if err != nil {
		return model.ChartSnapshot{}, fmt.Errorf("marshal chart snapshot: %w", err)
	}
	var out model.ChartSnapshot
	if err := json.Unmarshal(payload, &out); err != nil {
		return model.ChartSnapshot{}, fmt.Errorf("unmarshal chart snapshot: %w", err)
	}
	return out, nil
}

func normalizeVersions(in *BuildInput, strengthVersion, tenGodVersion, elementVersion, climateVersion, patternVersion, diseaseVersion, mediationVersion, usefulGodVersion string) {
	if in.Versions.BaziBaseDataVersion == "" {
		in.Versions.BaziBaseDataVersion = basedata.VersionV1
	}
	if in.Input.SchemaVersion == "" {
		in.Input.SchemaVersion = model.ReportInputSchemaVersion
	}
	if in.Chart.ChartSchemaVersion == "" {
		in.Chart.ChartSchemaVersion = model.ChartSnapshotVersion
	}
	if in.Versions.InputSchemaVersion == "" {
		in.Versions.InputSchemaVersion = in.Input.SchemaVersion
	}
	if in.Versions.ChartSchemaVersion == "" {
		in.Versions.ChartSchemaVersion = in.Chart.ChartSchemaVersion
	}
	if in.Versions.FactsSchemaVersion == "" {
		in.Versions.FactsSchemaVersion = model.FactsSchemaVersion
	}
	// The strength rules are the only active report rule set in this slice.
	in.Versions.RuleSetVersion = strengthVersion
	in.Versions.TenGodRuleVersion = tenGodVersion
	in.Versions.ElementPowerRuleVersion = elementVersion
	in.Versions.StrengthV2RuleVersion = strengthVersion
	in.Versions.TenGodEffectiveVersion = tenGodVersion
	in.Versions.ClimateRuleVersion = climateVersion
	in.Versions.PatternRuleVersion = patternVersion
	in.Versions.DiseaseRuleVersion = diseaseVersion
	in.Versions.MediationRuleVersion = mediationVersion
	in.Versions.UsefulGodRuleVersion = usefulGodVersion
	if in.Versions.LunarGoVersion == "" {
		in.Versions.LunarGoVersion = in.Chart.LunarGoVersion
	}
}

func buildUsefulGodFact(a model.UsefulGodAnalysis) model.ScoredFact {
	values := []string{}
	if a.Primary != "" {
		values = append(values, a.Primary)
	}
	if a.Secondary != "" {
		values = append(values, a.Secondary)
	}
	values = append(values, a.Favorable...)
	evidence, conflicts := []model.FactEvidence{}, []string{}
	for _, c := range a.Candidates {
		evidence = append(evidence, model.FactEvidence{RuleCode: a.RuleVersion + ".candidate", Source: "useful_god_simulation", Symbols: []string{c.Element}, Reason: fmt.Sprintf("role=%s score=%.2f marginal=%.2f valid=%t", c.Role, c.Score, c.Breakdown.MarginalUtility, c.Valid)})
		conflicts = append(conflicts, c.RejectReasons...)
	}
	score := a.Confidence * 100
	return model.ScoredFact{Code: "favorable.elements", Level: "determined", Score: &score, Values: values, Evidence: evidence, Conflicts: conflicts}
}

func buildElementFacts(a model.ElementPowerAnalysis) []model.ScoredFact {
	values := []struct {
		name, code string
		ratio      float64
	}{
		{"木", "wood", a.EffectiveRatio.Wood}, {"火", "fire", a.EffectiveRatio.Fire}, {"土", "earth", a.EffectiveRatio.Earth}, {"金", "metal", a.EffectiveRatio.Metal}, {"水", "water", a.EffectiveRatio.Water},
	}
	out := make([]model.ScoredFact, 0, len(values))
	for _, v := range values {
		score := v.ratio
		evidence := []model.FactEvidence{{RuleCode: a.RuleVersion, Source: "element_power_summary", Reason: fmt.Sprintf("season=%s progress=%.4f effective_ratio=%.4f", a.Season.Branch, a.Season.Progress, v.ratio)}}
		for _, c := range a.Contributions {
			if c.Element != v.name {
				continue
			}
			evidence = append(evidence, model.FactEvidence{RuleCode: a.RuleVersion + "." + c.Code, Source: "element_contribution", Pillars: []string{c.Position}, Symbols: []string{c.Symbol}, Reason: fmt.Sprintf("raw=%.4f season=%.4f visibility=%.4f seasonal=%.4f", c.RawPower, c.SeasonCoefficient, c.VisibilityMultiplier, c.SeasonalPower)})
		}
		out = append(out, model.ScoredFact{Code: "element.power." + v.code, Level: elementPowerLevel(v.ratio), Score: &score, Values: []string{v.name}, Evidence: evidence, Conflicts: []string{}})
	}
	return out
}

func elementPowerLevel(v float64) string {
	switch {
	case v == 0:
		return "missing"
	case v >= 30:
		return "strong"
	case v >= 22:
		return "relatively_strong"
	case v >= 14:
		return "balanced"
	default:
		return "weak"
	}
}
func elementFactCodes(v []model.ScoredFact) []string {
	out := make([]string, 0, len(v))
	for _, x := range v {
		out = append(out, x.Code)
	}
	return out
}

func buildTenGodFacts(a model.TenGodAnalysis) []model.ScoredFact {
	evidence := make([]model.FactEvidence, 0, len(a.Evidence)+1)
	evidence = append(evidence, model.FactEvidence{RuleCode: a.RuleVersion, Source: "ten_god_summary", Reason: fmt.Sprintf("total=%.2f concentration=%.4f", a.TotalScore, a.Concentration)})
	for _, e := range a.Evidence {
		evidence = append(evidence, model.FactEvidence{RuleCode: e.Code, Source: e.Source, Pillars: splitNonEmpty(e.Position), Symbols: append([]string{}, e.Symbols...), Reason: fmt.Sprintf("ten_god=%s category=%s raw=%.2f adjustment=%.2f; %s", e.TenGod, e.Category, e.RawScore, e.Adjustment, e.Reason)})
	}
	concentration := a.Concentration * 100
	values := append([]string{}, a.DominantGods...)
	values = append(values, a.SecondaryGods...)
	out := []model.ScoredFact{{Code: tenGodSummaryCode, Level: concentrationLevel(a.Concentration), Score: &concentration, Values: values, Evidence: evidence, Conflicts: []string{}}}
	for _, god := range a.Gods {
		score := god.EffectiveScore
		out = append(out, model.ScoredFact{Code: "ten_god." + god.Code, Level: presenceLevel(god), Score: &score, Values: []string{god.Name, god.Category}, Evidence: filterTenGodEvidence(a.Evidence, god.Name, god.Category), Conflicts: []string{}})
	}
	for _, category := range a.Categories {
		score := category.EffectiveScore
		out = append(out, model.ScoredFact{Code: "ten_god.category." + category.Category, Level: rankLevel(category.Rank), Score: &score, Values: []string{category.Category, fmt.Sprintf("raw=%.2f", category.RawScore), fmt.Sprintf("relation_adjustment=%.2f", category.RelationAdjustment)}, Evidence: filterTenGodEvidence(a.Evidence, "", category.Category), Conflicts: []string{}})
	}
	return out
}

func filterTenGodEvidence(in []model.TenGodEvidence, god, category string) []model.FactEvidence {
	out := []model.FactEvidence{}
	for _, e := range in {
		if god != "" && e.TenGod != god {
			continue
		}
		if category != "" && e.Category != category {
			continue
		}
		out = append(out, model.FactEvidence{RuleCode: e.Code, Source: e.Source, Pillars: splitNonEmpty(e.Position), Symbols: append([]string{}, e.Symbols...), Reason: fmt.Sprintf("raw=%.2f adjustment=%.2f; %s", e.RawScore, e.Adjustment, e.Reason)})
	}
	return out
}

func splitNonEmpty(v string) []string {
	if v == "" {
		return nil
	}
	return strings.Split(v, ",")
}
func presenceLevel(g model.TenGodScore) string {
	if g.RawScore == 0 {
		return "missing"
	}
	if g.Visible && g.Rooted {
		return "visible_and_rooted"
	}
	if g.Visible {
		return "visible"
	}
	if g.Rooted {
		return "rooted"
	}
	return "present"
}
func concentrationLevel(v float64) string {
	if v >= .5 {
		return "concentrated"
	}
	if v >= .3 {
		return "moderate"
	}
	return "distributed"
}
func rankLevel(rank int) string {
	if rank == 1 {
		return "dominant"
	}
	if rank == 2 {
		return "secondary"
	}
	return "supporting"
}
func tenGodFactCodes(facts []model.ScoredFact) []string {
	out := make([]string, 0, len(facts))
	for _, fact := range facts {
		out = append(out, fact.Code)
	}
	return out
}
func buildTenGodRuleMatches(a model.TenGodAnalysis) []model.RuleMatch {
	evidence := []model.FactEvidence{{RuleCode: a.RuleVersion, Source: "ten_god_analysis", Reason: "ten-god structure calculation completed"}}
	return []model.RuleMatch{{RuleCode: a.RuleVersion, Module: tenGodModule, Matched: true, Priority: 80, Evidence: evidence}}
}

func buildStrengthEvidence(a model.StrengthAnalysis) []model.FactEvidence {
	out := make([]model.FactEvidence, 0, len(a.Contributions)+len(a.Relations)+1)
	out = append(out, model.FactEvidence{RuleCode: a.RuleVersion, Source: "strength_summary", Reason: fmt.Sprintf("support=%.2f restraint=%.2f ratio=%.4f root=%s pattern=%s", a.SupportScore, a.RestraintScore, a.SupportRatio, a.RootLevel, a.Pattern)})
	for _, c := range a.Contributions {
		out = append(out, model.FactEvidence{RuleCode: a.RuleVersion + "." + c.Code, Source: c.Source, Pillars: nonEmpty(c.Position), Symbols: nonEmpty(c.Symbol), Reason: fmt.Sprintf("element=%s ten_god=%s category=%s score=%.2f adjustment=%.2f", c.Element, c.TenGod, c.Category, c.Score, c.Adjustment)})
	}
	for _, r := range a.Relations {
		out = append(out, model.FactEvidence{RuleCode: a.RuleVersion + "." + r.Code, Source: "strength_relation", Pillars: append([]string{}, r.Positions...), Symbols: append([]string{}, r.Symbols...), Reason: r.Reason})
	}
	return out
}

func buildStrengthV2Evidence(a model.StrengthV2Analysis) []model.FactEvidence {
	out := make([]model.FactEvidence, 0, len(a.Trace)+1)
	out = append(out, model.FactEvidence{RuleCode: a.RuleVersion, Source: "strength_v2_summary", Reason: fmt.Sprintf("score=%.2f confidence=%.4f support=%.2f pressure=%.2f de_ling=%.2f de_di=%.2f de_shi=%.2f", a.Score, a.Confidence, a.Support, a.Pressure, a.DeLing, a.DeDi, a.DeShi)})
	for _, x := range a.Trace {
		out = append(out, model.FactEvidence{RuleCode: x.Rule, Source: "strength_v2_trace", Reason: x.Reason})
	}
	for _, p := range a.Patterns {
		out = append(out, model.FactEvidence{RuleCode: a.RuleVersion + ".pattern." + p.Type, Source: "pattern_candidate", Reason: fmt.Sprintf("matched=%t subtype=%s confidence=%.4f evidence=%s rejected=%s", p.Matched, p.Subtype, p.Confidence, strings.Join(p.Evidence, " | "), strings.Join(p.RejectedBy, " | "))})
	}
	return out
}

func projectRelations(relations []model.StrengthRelation) ([]model.RelationFact, []model.RelationFact) {
	stems := []model.RelationFact{}
	branches := []model.RelationFact{}
	for _, r := range relations {
		fact := model.RelationFact{Code: r.Code, Type: r.Type, Pillars: append([]string{}, r.Positions...), Symbols: append([]string{}, r.Symbols...), Evidence: []model.FactEvidence{{RuleCode: r.Code, Source: "strength_analysis", Pillars: append([]string{}, r.Positions...), Symbols: append([]string{}, r.Symbols...), Reason: fmt.Sprintf("%s; element=%s score=%.2f transformed=%t", r.Reason, r.Element, r.Score, r.Transformed)}}}
		if strings.HasPrefix(r.Type, "stem_") {
			stems = append(stems, fact)
		} else {
			branches = append(branches, fact)
		}
	}
	return stems, branches
}

func projectV2Relations(relations []model.ElementStructureEvidence) ([]model.RelationFact, []model.RelationFact) {
	stems, branches := []model.RelationFact{}, []model.RelationFact{}
	for _, r := range relations {
		e := model.FactEvidence{RuleCode: r.Code, Source: "element_power_structure", Pillars: append([]string{}, r.Positions...), Symbols: append([]string{}, r.Symbols...), Reason: fmt.Sprintf("state=%s target=%s confidence=%.2f transfer_rate=%.4f; %s", r.State, r.TargetElement, r.Confidence, r.TransferRate, r.Reason)}
		f := model.RelationFact{Code: r.Code, Type: r.Type, Pillars: append([]string{}, r.Positions...), Symbols: append([]string{}, r.Symbols...), Evidence: []model.FactEvidence{e}}
		if r.Type == "combination" {
			stems = append(stems, f)
		} else {
			branches = append(branches, f)
		}
	}
	return stems, branches
}

func buildPatternFacts(level string, a model.StrengthAnalysis) []model.ScoredFact {
	values := []string{a.Pattern}
	if a.PatternSubtype != "" {
		values = append(values, a.PatternSubtype)
	}
	score := a.SupportRatio * 100
	return []model.ScoredFact{{Code: "pattern." + a.Pattern, Level: level, Score: &score, Values: values, Evidence: []model.FactEvidence{{RuleCode: a.RuleVersion, Source: "strength_analysis", Reason: fmt.Sprintf("root=%s false_following=%t", a.RootLevel, a.FalseFollowing)}}, Conflicts: []string{}}}
}
func buildPatternFactsV2(a model.StrengthV2Analysis) []model.ScoredFact {
	out := []model.ScoredFact{}
	for _, p := range a.Patterns {
		score := p.Confidence * 100
		conflicts := append([]string{}, p.RejectedBy...)
		values := []string{p.Type}
		if p.Subtype != "" {
			values = append(values, p.Subtype)
		}
		if p.Alternative != "" {
			values = append(values, p.Alternative)
		}
		out = append(out, model.ScoredFact{Code: "pattern." + p.Type, Level: map[bool]string{true: "matched", false: "rejected"}[p.Matched], Score: &score, Values: values, Evidence: []model.FactEvidence{{RuleCode: a.RuleVersion + ".pattern." + p.Type, Source: "strength_v2", Reason: strings.Join(p.Evidence, " | ")}}, Conflicts: conflicts})
	}
	return out
}
func buildRuleMatches(a model.StrengthAnalysis) []model.RuleMatch {
	out := []model.RuleMatch{{RuleCode: a.RuleVersion, Module: strengthModule, Matched: true, Priority: 100, Evidence: []model.FactEvidence{{RuleCode: a.RuleVersion, Source: "strength_analysis", Reason: "day-master strength classification completed"}}}}
	for i, r := range a.Relations {
		out = append(out, model.RuleMatch{RuleCode: r.Code, Module: strengthModule, Matched: true, Priority: 90 - i, Evidence: []model.FactEvidence{{RuleCode: r.Code, Source: "strength_analysis", Pillars: append([]string{}, r.Positions...), Symbols: append([]string{}, r.Symbols...), Reason: r.Reason}}})
	}
	return out
}
func buildStrengthV2RuleMatches(a model.StrengthV2Analysis, structures []model.ElementStructureEvidence) []model.RuleMatch {
	out := []model.RuleMatch{{RuleCode: a.RuleVersion, Module: strengthModule, Matched: true, Priority: 100, Evidence: buildStrengthV2Evidence(a)}}
	for i, r := range structures {
		out = append(out, model.RuleMatch{RuleCode: r.Code, Module: "element_structure", Matched: true, Priority: 90 - i, Evidence: []model.FactEvidence{{RuleCode: r.Code, Source: "element_power_structure", Pillars: append([]string{}, r.Positions...), Symbols: append([]string{}, r.Symbols...), Reason: r.Reason}}})
	}
	return out
}
func strengthValues(a model.StrengthAnalysis) []string {
	v := []string{a.DayElement, a.RootLevel, a.Pattern}
	if a.PatternSubtype != "" {
		v = append(v, a.PatternSubtype)
	}
	if a.FalseFollowing {
		v = append(v, "false_following")
	}
	return v
}
func strengthV2Values(a model.StrengthV2Analysis) []string {
	v := []string{a.Level, fmt.Sprintf("score=%.2f", a.Score), fmt.Sprintf("confidence=%.4f", a.Confidence)}
	for _, p := range a.Patterns {
		if p.Matched {
			v = append(v, p.Type)
			if p.Subtype != "" {
				v = append(v, p.Subtype)
			}
		}
	}
	return v
}
func nonEmpty(v string) []string {
	if v == "" {
		return nil
	}
	return []string{v}
}

// CalculateHash reproduces the canonical hash stored with an immutable facts
// snapshot. It is exported so read-side integrity checks cannot drift from the
// write-side normalization rules.
func CalculateHash(f model.InterpretationFacts) (string, error) {
	f.FactsHash = ""
	// Prompt changes affect LLM call traces, not deterministic report facts.
	f.Versions.PromptVersion = ""
	return hashutil.CanonicalJSONSHA256(f)
}
