package usefulgod

import (
	"fmt"
	"math"
	"sort"
)

const RuleVersionV1 = "useful-god-v1.1"

var Elements = []string{"木", "火", "土", "金", "水"}

type Weights struct{ FuYi, Disease, Pattern, Climate, Flow, Availability float64 }
type RuleSet struct {
	Version      string
	Delta        float64
	NeutralLimit float64
	Weights      Weights
}
type Disease struct {
	Code              string
	Severity          float64
	CandidateElements []string
}
type State struct {
	DayElement                                                                         string
	StrengthScore, Temperature, Moisture, FlowScore, PatternIntegrity, DiseaseSeverity float64
	PrimaryDisease                                                                     *Disease
	ElementRatio                                                                       map[string]float64
	SpecialPattern, PatternAmbiguous                                                   bool
}
type Simulator interface {
	Simulate(element string, delta float64) (State, error)
}
type ScoreBreakdown struct{ FuYi, Disease, Pattern, Climate, Flow, Availability, MarginalUtility, SideEffects float64 }
type Simulation struct {
	Delta, Health, MarginalUtility                                                     float64
	StrengthScore, Temperature, Moisture, FlowScore, PatternIntegrity, DiseaseSeverity float64
}
type Candidate struct {
	Element, Role                   string
	Score, Confidence, Availability float64
	Valid, ResolvesPrimary          bool
	PrimaryRequired                 bool
	Breakdown                       ScoreBreakdown
	Simulations                     []Simulation
	Reasons, RejectReasons          []string
	OptimalRange                    [2]float64
}
type Result struct {
	RuleVersion, Primary, Secondary  string
	Favorable, Taboo, Enemy, Neutral []string
	Confidence                       float64
	Candidates                       []Candidate
	Warnings                         []string
}

func RuleV1() RuleSet {
	return RuleSet{Version: RuleVersionV1, Delta: 8, NeutralLimit: 15, Weights: Weights{FuYi: 25, Disease: 20, Pattern: 20, Climate: 15, Flow: 10, Availability: 10}}
}

func Evaluate(base State, simulator Simulator, rules RuleSet) (Result, error) {
	if simulator == nil {
		return Result{}, fmt.Errorf("simulator is required")
	}
	weights := dynamicWeights(base, rules.Weights)
	baseHealth := health(base, weights)
	out := Result{RuleVersion: rules.Version}
	for _, element := range Elements {
		c := Candidate{Element: element, Valid: true, PrimaryRequired: base.PrimaryDisease != nil, Availability: round(base.ElementRatio[element])}
		c.Breakdown.FuYi = fuyiScore(element, base.DayElement, base.StrengthScore, weights.FuYi)
		c.Breakdown.Climate = climateScore(element, base, weights.Climate)
		c.Breakdown.Disease, c.ResolvesPrimary = diseaseScore(element, base.PrimaryDisease, weights.Disease)
		c.Breakdown.Flow = flowStatic(element, base, weights.Flow)
		c.Breakdown.Pattern = patternScore(element, base, weights.Pattern)
		c.Breakdown.Availability = availabilityScore(c.Availability, weights.Availability)
		for _, delta := range []float64{5, 10, 15, 20} {
			next, err := simulator.Simulate(element, delta)
			if err != nil {
				return Result{}, fmt.Errorf("simulate %s +%.0f: %w", element, delta, err)
			}
			nextHealth := health(next, weights)
			u := nextHealth - baseHealth
			c.Simulations = append(c.Simulations, Simulation{delta, round(nextHealth), round(u), round(next.StrengthScore), round(next.Temperature), round(next.Moisture), round(next.FlowScore), round(next.PatternIntegrity), round(next.DiseaseSeverity)})
		}
		selected := closestSimulation(c.Simulations, rules.Delta)
		c.Breakdown.MarginalUtility = selected.MarginalUtility
		c.Breakdown.SideEffects = sideEffects(base, selected)
		c.OptimalRange = optimalRange(base.ElementRatio[element], c.Simulations)
		if severeClimateWorsened(base, selected) {
			c.Valid = false
			c.RejectReasons = append(c.RejectReasons, "severe_climate_worsened")
		}
		if base.SpecialPattern && selected.PatternIntegrity < base.PatternIntegrity-20 {
			c.Valid = false
			c.RejectReasons = append(c.RejectReasons, "confirmed_special_pattern_damaged")
		}
		if selected.DiseaseSeverity > base.DiseaseSeverity+20 {
			c.Valid = false
			c.RejectReasons = append(c.RejectReasons, "new_major_imbalance")
		}
		c.Score = round(c.Breakdown.FuYi + c.Breakdown.Disease + c.Breakdown.Pattern + c.Breakdown.Climate + c.Breakdown.Flow + c.Breakdown.Availability + c.Breakdown.MarginalUtility - c.Breakdown.SideEffects)
		c.Confidence = candidateConfidence(c, base)
		c.Reasons = buildReasons(c)
		out.Candidates = append(out.Candidates, c)
	}
	sort.SliceStable(out.Candidates, func(i, j int) bool { return out.Candidates[i].Score > out.Candidates[j].Score })
	selectRoles(&out, rules)
	out.Confidence = resultConfidence(out, base)
	out.Warnings = []string{"candidate_simulation_uses_rule_aligned_dynamic_health_weights", "availability_root_protection_coefficients_require_domain_calibration"}
	return out, nil
}

func dynamicWeights(s State, w Weights) Weights {
	if math.Abs(s.Temperature) > 70 || math.Abs(s.Moisture) > 70 {
		return rebalance(w, "climate", 35)
	}
	if s.DiseaseSeverity > 70 {
		return rebalance(w, "disease", 35)
	}
	if s.FlowScore < -20 {
		return rebalance(w, "flow", 30)
	}
	return w
}
func rebalance(w Weights, target string, value float64) Weights {
	old := map[string]float64{"climate": w.Climate, "disease": w.Disease, "flow": w.Flow}[target]
	remaining := 100 - value
	scale := remaining / (100 - old)
	w.FuYi *= scale
	w.Disease *= scale
	w.Pattern *= scale
	w.Climate *= scale
	w.Flow *= scale
	w.Availability *= scale
	if target == "climate" {
		w.Climate = value
	}
	if target == "disease" {
		w.Disease = value
	}
	if target == "flow" {
		w.Flow = value
	}
	return w
}
func health(s State, w Weights) float64 {
	// Availability is a natal-chart feasibility score rather than a simulated
	// state, so it is excluded from health and the other rule weights are
	// normalized. Climate retains the documented 60/40 temperature/moisture split.
	total := w.FuYi + w.Disease + w.Pattern + w.Climate + w.Flow
	if total <= 0 {
		return 0
	}
	return (100-math.Abs(s.StrengthScore))*(w.FuYi/total) +
		(100-math.Abs(s.Temperature))*(w.Climate*.60/total) +
		(100-math.Abs(s.Moisture))*(w.Climate*.40/total) +
		s.PatternIntegrity*(w.Pattern/total) +
		(s.FlowScore+100)/2*(w.Flow/total) +
		(100-s.DiseaseSeverity)*(w.Disease/total)
}
func fuyiScore(e, day string, strength, w float64) float64 {
	generatedBy := map[string]string{"木": "水", "火": "木", "土": "火", "金": "土", "水": "金"}
	output := map[string]string{"木": "火", "火": "土", "土": "金", "金": "水", "水": "木"}
	controls := map[string]string{"木": "土", "火": "金", "土": "水", "金": "木", "水": "火"}
	controlledBy := map[string]string{"木": "金", "火": "水", "土": "木", "金": "火", "水": "土"}
	if math.Abs(strength) <= 15 {
		return 0
	}
	if strength < 0 {
		if e == day {
			return w
		}
		if e == generatedBy[day] {
			return w * .85
		}
		return -w * .35
	}
	if e == output[day] {
		return w
	}
	if e == controls[day] {
		return w * .85
	}
	if e == controlledBy[day] {
		return w * .75
	}
	return -w * .35
}
func climateScore(e string, s State, w float64) float64 {
	score := 0.0
	if s.Temperature <= -40 && e == "火" {
		score += w
	}
	if s.Temperature >= 40 && e == "水" {
		score += w
	}
	if s.Moisture <= -40 && e == "水" {
		score += w
	}
	if s.Moisture >= 40 && e == "火" {
		score += w
	}
	return math.Min(w, score)
}
func diseaseScore(e string, d *Disease, w float64) (float64, bool) {
	if d == nil {
		return 0, false
	}
	for _, x := range d.CandidateElements {
		if x == e {
			return w, true
		}
	}
	return 0, false
}
func flowStatic(e string, s State, w float64) float64 {
	if s.FlowScore >= 0 {
		return 0
	}
	return w * .25
}
func patternScore(e string, s State, w float64) float64 {
	if s.SpecialPattern {
		return w * .25
	}
	return 0
}
func availabilityScore(p, w float64) float64 { return w * math.Min(1, p/20) }
func closestSimulation(v []Simulation, d float64) Simulation {
	best := v[0]
	for _, x := range v {
		if math.Abs(x.Delta-d) < math.Abs(best.Delta-d) {
			best = x
		}
	}
	return best
}
func optimalRange(current float64, v []Simulation) [2]float64 {
	lo, hi := current, current
	for _, x := range v {
		if x.MarginalUtility > 0 {
			if lo == current {
				lo = current + x.Delta
			}
			hi = current + x.Delta
		}
	}
	return [2]float64{round(lo), round(hi)}
}
func severeClimateWorsened(base State, s Simulation) bool {
	return (math.Abs(base.Temperature) > 70 && math.Abs(s.Temperature) > math.Abs(base.Temperature)+5) || (math.Abs(base.Moisture) > 70 && math.Abs(s.Moisture) > math.Abs(base.Moisture)+5)
}
func sideEffects(base State, s Simulation) float64 {
	penalty := math.Max(0, math.Abs(s.Temperature)-math.Abs(base.Temperature)) * .20
	penalty += math.Max(0, math.Abs(s.Moisture)-math.Abs(base.Moisture)) * .15
	penalty += math.Max(0, s.DiseaseSeverity-base.DiseaseSeverity) * .30
	penalty += math.Max(0, base.PatternIntegrity-s.PatternIntegrity) * .25
	return round(penalty)
}
func candidateConfidence(c Candidate, s State) float64 {
	v := .55 + math.Min(math.Abs(c.Score)/100, .30)
	if s.PatternAmbiguous {
		v -= .15
	}
	if !c.Valid {
		v -= .2
	}
	return round4(clamp(v, .2, .95))
}
func buildReasons(c Candidate) []string {
	r := []string{}
	if c.Breakdown.Climate > 0 {
		r = append(r, "improves_climate")
	}
	if c.Breakdown.Disease > 0 {
		r = append(r, "addresses_primary_disease")
	}
	if c.Breakdown.MarginalUtility > 0 {
		r = append(r, "positive_marginal_utility")
	}
	if c.Breakdown.Availability > 0 {
		r = append(r, "present_in_natal_chart")
	}
	return r
}
func selectRoles(out *Result, r RuleSet) {
	for i := range out.Candidates {
		c := &out.Candidates[i]
		if c.Valid && c.Breakdown.MarginalUtility <= 0 {
			c.Reasons = appendUnique(c.Reasons, "non_positive_marginal_utility")
		}
		if c.Valid && c.PrimaryRequired && !c.ResolvesPrimary {
			c.Reasons = appendUnique(c.Reasons, "does_not_resolve_primary_disease")
		}
		switch {
		case !c.Valid:
			c.Role = "taboo"
			out.Taboo = append(out.Taboo, c.Element)
		case out.Primary == "" && c.Score > 0 && c.Breakdown.MarginalUtility > 0 && (!c.PrimaryRequired || c.ResolvesPrimary):
			out.Primary = c.Element
			c.Role = "primary_useful"
		case c.Valid && out.Primary != "" && out.Secondary == "" && c.Score > r.NeutralLimit && c.Breakdown.MarginalUtility > 0 && synergizes(c.Element, out.Primary):
			out.Secondary = c.Element
			c.Role = "secondary_useful"
		case c.Score > r.NeutralLimit && c.Breakdown.MarginalUtility > 0 && (!c.PrimaryRequired || c.ResolvesPrimary):
			c.Role = "favorable"
			out.Favorable = append(out.Favorable, c.Element)
		case c.Score > r.NeutralLimit:
			c.Role = "pending_review"
		case c.Score < -r.NeutralLimit:
			c.Role = "taboo"
			out.Taboo = append(out.Taboo, c.Element)
		default:
			c.Role = "neutral"
			out.Neutral = append(out.Neutral, c.Element)
		}
	}
	for i := range out.Candidates {
		c := &out.Candidates[i]
		if c.Role == "neutral" && helpsAny(c.Element, out.Taboo) {
			c.Role = "enemy"
			out.Enemy = append(out.Enemy, c.Element)
			out.Neutral = remove(out.Neutral, c.Element)
		}
	}
}
func appendUnique(v []string, x string) []string {
	for _, item := range v {
		if item == x {
			return v
		}
	}
	return append(v, x)
}
func synergizes(e, primary string) bool { return generates(e, primary) || generates(primary, e) }
func helpsAny(e string, v []string) bool {
	for _, x := range v {
		if generates(e, x) {
			return true
		}
	}
	return false
}
func generates(a, b string) bool {
	return map[string]string{"木": "火", "火": "土", "土": "金", "金": "水", "水": "木"}[a] == b
}
func resultConfidence(r Result, s State) float64 {
	if r.Primary == "" {
		return .25
	}
	top := r.Candidates[0].Score
	second := -100.0
	if len(r.Candidates) > 1 {
		second = r.Candidates[1].Score
	}
	v := .55 + math.Min(math.Max(0, top-second)/50, .30)
	if s.PatternAmbiguous {
		v -= .15
	}
	return round4(clamp(v, .2, .95))
}
func remove(v []string, x string) []string {
	o := []string{}
	for _, e := range v {
		if e != x {
			o = append(o, e)
		}
	}
	return o
}
func clamp(v, a, b float64) float64 {
	if v < a {
		return a
	}
	if v > b {
		return b
	}
	return v
}
func round(v float64) float64  { return math.Round(v*100) / 100 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
