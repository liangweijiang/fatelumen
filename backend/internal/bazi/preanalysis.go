package bazi

import (
	"fmt"
	"strings"

	climatecalc "fatelumen/backend/internal/bazi/climate"
	diseasecalc "fatelumen/backend/internal/bazi/disease"
	elementcalc "fatelumen/backend/internal/bazi/elementpower"
	mediationcalc "fatelumen/backend/internal/bazi/mediation"
	patterncalc "fatelumen/backend/internal/bazi/pattern"
	strengthv2calc "fatelumen/backend/internal/bazi/strengthv2"
	usefulcalc "fatelumen/backend/internal/bazi/usefulgod"
	"fatelumen/backend/internal/model"
)

func evaluatePreanalysis(chart *model.ChartData) error {
	if chart.ElementPower == nil || chart.StrengthV2 == nil || chart.TenGodEffective == nil {
		return fmt.Errorf("v2 element, strength and ten-god analyses are required")
	}
	elements := modelElementMap(chart.ElementPower.EffectiveRatio)
	climate, err := climatecalc.Evaluate(climatecalc.Input{MonthBranch: chart.Pillars.Month.Branch, ElementRatio: elements}, climatecalc.RuleV1())
	if err != nil {
		return fmt.Errorf("evaluate climate: %w", err)
	}
	mediation := mediationcalc.Evaluate(mediationcalc.Input{ElementRatio: elements}, mediationcalc.RuleV1())
	categories, gods := tenGodMaps(*chart.TenGodEffective)
	existing := make([]patterncalc.Candidate, 0, len(chart.StrengthV2.Patterns))
	for _, p := range chart.StrengthV2.Patterns {
		state := "candidate"
		if p.Matched {
			state = "confirmed"
		}
		existing = append(existing, patterncalc.Candidate{Code: p.Type + ":" + p.Subtype, State: state, Confidence: p.Confidence, RejectedBy: append([]string{}, p.RejectedBy...), Evidence: append([]string{}, p.Evidence...)})
	}
	pattern := patterncalc.Evaluate(patterncalc.Input{StrengthScore: chart.StrengthV2.Score, RootPower: chart.StrengthV2.RootPower, Gods: gods, Categories: categories, Existing: existing}, patterncalc.RuleV1())
	disease := diseasecalc.Evaluate(diseasecalc.Input{DayElement: chart.DayMaster.Element, StrengthScore: chart.StrengthV2.Score, Temperature: climate.Temperature, Moisture: climate.Moisture, Categories: categories, ConflictCount: len(mediation.Conflicts)}, diseasecalc.RuleV1())
	chart.Climate = toModelClimate(climate)
	chart.Pattern = toModelPattern(pattern)
	chart.Disease = toModelDisease(disease)
	chart.Mediation = toModelMediation(mediation)
	simulator := &chartStateSimulator{chart: chart, elementInput: elementInputFromChart(chart)}
	baseState := simulator.state(elements, chart.StrengthV2.Score, climate, pattern, disease, mediation)
	useful, err := usefulcalc.Evaluate(baseState, simulator, usefulcalc.RuleV1())
	if err != nil {
		return fmt.Errorf("evaluate useful god: %w", err)
	}
	chart.UsefulGod = toModelUsefulGod(useful)
	return nil
}

type chartStateSimulator struct {
	chart        *model.ChartData
	elementInput elementcalc.Input
}

func (s *chartStateSimulator) Simulate(element string, delta float64) (usefulcalc.State, error) {
	v, err := elementcalc.SimulateIncrease(s.elementInput, elementcalc.RuleV2(), element, delta)
	if err != nil {
		return usefulcalc.State{}, err
	}
	ratios := map[string]float64{"木": v.Wood, "火": v.Fire, "土": v.Earth, "金": v.Metal, "水": v.Water}
	categories := categoriesFromElements(s.chart.DayMaster.Element, ratios)
	structures := make([]strengthv2calc.Structure, 0, len(s.chart.ElementPower.Structures))
	for _, x := range s.chart.ElementPower.Structures {
		structures = append(structures, strengthv2calc.Structure{Code: x.Code, Type: x.Type, TargetElement: x.TargetElement, State: x.State, Confidence: x.Confidence})
	}
	strength, err := strengthv2calc.Evaluate(strengthv2calc.Input{DayElement: s.chart.DayMaster.Element, MonthBranch: s.chart.Pillars.Month.Branch, SeasonCoefficient: elementValue(s.chart.ElementPower.Season.Coefficients, s.chart.DayMaster.Element), SeasonProgress: s.chart.ElementPower.Season.Progress, RootPower: s.chart.ElementPower.RootPower, Categories: categories, ElementRatios: ratios, Structures: structures, HourUnknown: s.chart.HourUnknown})
	if err != nil {
		return usefulcalc.State{}, err
	}
	categoryMap := map[string]float64{"peer": categories.Peer, "resource": categories.Resource, "output": categories.Output, "wealth": categories.Wealth, "officer": categories.Officer}
	gods := scaledGods(*s.chart.TenGodEffective, categoryMap)
	existing := make([]patterncalc.Candidate, 0, len(strength.Patterns))
	for _, p := range strength.Patterns {
		state := "candidate"
		if p.Matched {
			state = "confirmed"
		}
		existing = append(existing, patterncalc.Candidate{Code: p.Type + ":" + p.Subtype, State: state, Confidence: p.Confidence, RejectedBy: p.RejectedBy, Evidence: p.Evidence})
	}
	pattern := patterncalc.Evaluate(patterncalc.Input{StrengthScore: strength.Score, RootPower: strength.RootPower, Gods: gods, Categories: categoryMap, Existing: existing}, patterncalc.RuleV1())
	climate, err := climatecalc.Evaluate(climatecalc.Input{MonthBranch: s.chart.Pillars.Month.Branch, ElementRatio: ratios}, climatecalc.RuleV1())
	if err != nil {
		return usefulcalc.State{}, err
	}
	mediation := mediationcalc.Evaluate(mediationcalc.Input{ElementRatio: ratios}, mediationcalc.RuleV1())
	disease := diseasecalc.Evaluate(diseasecalc.Input{DayElement: s.chart.DayMaster.Element, StrengthScore: strength.Score, Temperature: climate.Temperature, Moisture: climate.Moisture, Categories: categoryMap, ConflictCount: len(mediation.Conflicts)}, diseasecalc.RuleV1())
	return s.state(ratios, strength.Score, climate, pattern, disease, mediation), nil
}
func (s *chartStateSimulator) state(ratios map[string]float64, strength float64, climate climatecalc.Result, pattern patterncalc.Result, disease diseasecalc.Result, mediation mediationcalc.Result) usefulcalc.State {
	integrity := 75.0
	if pattern.Primary != "" {
		integrity = 90
	}
	special, ambiguous := false, false
	for _, c := range pattern.Candidates {
		isSpecial := strings.HasPrefix(c.Code, "following:") || strings.HasPrefix(c.Code, "dominant:") || strings.HasPrefix(c.Code, "transformation:")
		if isSpecial {
			if c.State == "confirmed" {
				special = true
			}
			if c.State == "candidate" && c.Confidence >= .5 {
				ambiguous = true
			}
		}
	}
	var primary *usefulcalc.Disease
	if disease.Primary != nil {
		primary = &usefulcalc.Disease{Code: disease.Primary.Code, Severity: disease.Primary.Severity, CandidateElements: append([]string{}, disease.Primary.CandidateElements...)}
	}
	return usefulcalc.State{DayElement: s.chart.DayMaster.Element, StrengthScore: strength, Temperature: climate.Temperature, Moisture: climate.Moisture, FlowScore: mediation.FlowScore, PatternIntegrity: integrity, DiseaseSeverity: maxDiseaseSeverity(disease), PrimaryDisease: primary, ElementRatio: ratios, SpecialPattern: special, PatternAmbiguous: ambiguous}
}

func elementInputFromChart(c *model.ChartData) elementcalc.Input {
	return elementcalc.Input{Year: elementcalc.Pillar{Stem: c.Pillars.Year.Stem, Branch: c.Pillars.Year.Branch}, Month: elementcalc.Pillar{Stem: c.Pillars.Month.Stem, Branch: c.Pillars.Month.Branch}, Day: elementcalc.Pillar{Stem: c.Pillars.Day.Stem, Branch: c.Pillars.Day.Branch}, Hour: elementcalc.Pillar{Stem: c.Pillars.Hour.Stem, Branch: c.Pillars.Hour.Branch}, SeasonProgress: c.ElementPower.Season.Progress}
}
func categoriesFromElements(day string, r map[string]float64) strengthv2calc.CategoryPower {
	generation := map[string]string{"木": "火", "火": "土", "土": "金", "金": "水", "水": "木"}
	control := map[string]string{"木": "土", "火": "金", "土": "水", "金": "木", "水": "火"}
	resource, officer := "", ""
	for e, x := range generation {
		if x == day {
			resource = e
		}
	}
	for e, x := range control {
		if x == day {
			officer = e
		}
	}
	return strengthv2calc.CategoryPower{Peer: r[day], Resource: r[resource], Output: r[generation[day]], Wealth: r[control[day]], Officer: r[officer]}
}
func elementValue(v model.ElementPowerVector, e string) float64 {
	return map[string]float64{"木": v.Wood, "火": v.Fire, "土": v.Earth, "金": v.Metal, "水": v.Water}[e]
}
func scaledGods(raw model.TenGodAnalysis, categories map[string]float64) map[string]float64 {
	old := map[string]float64{}
	for _, c := range raw.Categories {
		old[c.Category] = c.EffectiveScore
	}
	out := map[string]float64{}
	for _, g := range raw.Gods {
		if old[g.Category] > 0 {
			out[g.Name] = g.EffectiveScore * categories[g.Category] / old[g.Category]
		}
	}
	return out
}
func maxDiseaseSeverity(v diseasecalc.Result) float64 {
	m := 0.0
	for _, x := range v.All {
		if x.Severity > m {
			m = x.Severity
		}
	}
	return m
}

func modelElementMap(v model.ElementPowerVector) map[string]float64 {
	return map[string]float64{"木": v.Wood, "火": v.Fire, "土": v.Earth, "金": v.Metal, "水": v.Water}
}
func tenGodMaps(v model.TenGodAnalysis) (map[string]float64, map[string]float64) {
	c := map[string]float64{}
	g := map[string]float64{}
	for _, x := range v.Categories {
		c[x.Category] = x.EffectiveScore
	}
	for _, x := range v.Gods {
		g[x.Name] = x.EffectiveScore
	}
	return c, g
}
func toModelClimate(v climatecalc.Result) *model.ClimateAnalysis {
	e := make([]model.AnalysisEvidence, 0, len(v.Evidence))
	for _, x := range v.Evidence {
		e = append(e, model.AnalysisEvidence{Rule: x.Rule, Reason: x.Reason, Values: x.Values})
	}
	return &model.ClimateAnalysis{RuleVersion: v.RuleVersion, Temperature: v.Temperature, Moisture: v.Moisture, TemperatureLevel: v.TemperatureLevel, MoistureLevel: v.MoistureLevel, Candidates: v.Candidates, Evidence: e, Warnings: v.Warnings}
}
func toModelPattern(v patterncalc.Result) *model.PatternAnalysis {
	c := make([]model.PatternReviewCandidate, 0, len(v.Candidates))
	for _, x := range v.Candidates {
		c = append(c, model.PatternReviewCandidate{Code: x.Code, State: x.State, Confidence: x.Confidence, Requirements: x.Requirements, RejectedBy: x.RejectedBy, Evidence: x.Evidence})
	}
	return &model.PatternAnalysis{RuleVersion: v.RuleVersion, Primary: v.Primary, Candidates: c, Warnings: v.Warnings}
}
func toModelDisease(v diseasecalc.Result) *model.DiseaseAnalysis {
	convert := func(x diseasecalc.Item) model.DiseaseItem {
		return model.DiseaseItem{Code: x.Code, Level: x.Level, Severity: x.Severity, CandidateElements: x.CandidateElements, Evidence: x.Evidence}
	}
	all := make([]model.DiseaseItem, 0, len(v.All))
	for _, x := range v.All {
		all = append(all, convert(x))
	}
	secondary := make([]model.DiseaseItem, 0, len(v.Secondary))
	for _, x := range v.Secondary {
		secondary = append(secondary, convert(x))
	}
	minor := make([]model.DiseaseItem, 0, len(v.Minor))
	for _, x := range v.Minor {
		minor = append(minor, convert(x))
	}
	var primary *model.DiseaseItem
	if v.Primary != nil {
		x := convert(*v.Primary)
		primary = &x
	}
	return &model.DiseaseAnalysis{RuleVersion: v.RuleVersion, Primary: primary, Secondary: secondary, Minor: minor, All: all, Warnings: v.Warnings}
}
func toModelMediation(v mediationcalc.Result) *model.MediationAnalysis {
	c := make([]model.MediationConflict, 0, len(v.Conflicts))
	for _, x := range v.Conflicts {
		c = append(c, model.MediationConflict{Controller: x.Controller, Controlled: x.Controlled, Bridge: x.Bridge, State: x.State, ControllerPower: x.ControllerPower, ControlledPower: x.ControlledPower, BridgePower: x.BridgePower, Ratio: x.Ratio, Confidence: x.Confidence, Evidence: x.Evidence})
	}
	return &model.MediationAnalysis{RuleVersion: v.RuleVersion, FlowScore: v.FlowScore, Conflicts: c, Warnings: v.Warnings}
}
func toModelUsefulGod(v usefulcalc.Result) *model.UsefulGodAnalysis {
	c := make([]model.UsefulGodCandidate, 0, len(v.Candidates))
	for _, x := range v.Candidates {
		sims := make([]model.UsefulGodSimulation, 0, len(x.Simulations))
		for _, s := range x.Simulations {
			sims = append(sims, model.UsefulGodSimulation{Delta: s.Delta, Health: s.Health, MarginalUtility: s.MarginalUtility, StrengthScore: s.StrengthScore, Temperature: s.Temperature, Moisture: s.Moisture, FlowScore: s.FlowScore, PatternIntegrity: s.PatternIntegrity, DiseaseSeverity: s.DiseaseSeverity})
		}
		c = append(c, model.UsefulGodCandidate{Element: x.Element, Role: x.Role, Score: x.Score, Confidence: x.Confidence, Availability: x.Availability, Valid: x.Valid, ResolvesPrimary: x.ResolvesPrimary, PrimaryRequired: x.PrimaryRequired, Breakdown: model.UsefulGodScoreBreakdown{FuYi: x.Breakdown.FuYi, Disease: x.Breakdown.Disease, Pattern: x.Breakdown.Pattern, Climate: x.Breakdown.Climate, Flow: x.Breakdown.Flow, Availability: x.Breakdown.Availability, MarginalUtility: x.Breakdown.MarginalUtility, SideEffects: x.Breakdown.SideEffects}, Simulations: sims, Reasons: x.Reasons, RejectReasons: x.RejectReasons, OptimalRange: x.OptimalRange})
	}
	return &model.UsefulGodAnalysis{RuleVersion: v.RuleVersion, Primary: v.Primary, Secondary: v.Secondary, Favorable: v.Favorable, Taboo: v.Taboo, Enemy: v.Enemy, Neutral: v.Neutral, Confidence: v.Confidence, Candidates: c, Warnings: v.Warnings}
}
