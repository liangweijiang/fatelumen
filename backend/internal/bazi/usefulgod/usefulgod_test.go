package usefulgod

import "testing"

type fakeSimulator struct{ base State }

func (f fakeSimulator) Simulate(element string, delta float64) (State, error) {
	s := f.base
	s.ElementRatio = copyRatios(f.base.ElementRatio)
	s.ElementRatio[element] += delta
	if element == "火" {
		s.Temperature += delta * 3
		s.DiseaseSeverity -= delta * 2
	} else if element == "水" {
		s.Temperature -= delta * 3
		s.DiseaseSeverity += delta
	}
	return s, nil
}
func copyRatios(v map[string]float64) map[string]float64 {
	o := map[string]float64{}
	for k, x := range v {
		o[k] = x
	}
	return o
}
func TestEvaluateSimulatesAllElementsAndSelectsColdRemedy(t *testing.T) {
	d := &Disease{Code: "COLD", Severity: 80, CandidateElements: []string{"火"}}
	base := State{DayElement: "木", StrengthScore: -30, Temperature: -80, Moisture: 20, FlowScore: 0, PatternIntegrity: 90, DiseaseSeverity: 80, PrimaryDisease: d, ElementRatio: map[string]float64{"木": 20, "火": 10, "土": 20, "金": 20, "水": 30}}
	got, err := Evaluate(base, fakeSimulator{base}, RuleV1())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Candidates) != 5 {
		t.Fatalf("candidates=%d", len(got.Candidates))
	}
	if got.Primary != "火" {
		t.Fatalf("primary=%s result=%+v", got.Primary, got)
	}
}
func TestEvaluateHardRejectsWorsenedSevereClimate(t *testing.T) {
	base := State{DayElement: "木", Temperature: -80, PatternIntegrity: 90, ElementRatio: map[string]float64{"木": 20, "火": 10, "土": 20, "金": 20, "水": 30}}
	got, err := Evaluate(base, fakeSimulator{base}, RuleV1())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got.Candidates {
		if c.Element == "水" && (c.Valid || c.Role != "taboo") {
			t.Fatalf("water should be rejected and taboo: %+v", c)
		}
	}
}

func TestSelectRolesDoesNotPresentFailedPrimaryConditionsAsFavorable(t *testing.T) {
	out := Result{Candidates: []Candidate{
		{Element: "火", Valid: true, Score: 50, PrimaryRequired: true, ResolvesPrimary: true, Breakdown: ScoreBreakdown{MarginalUtility: -1}},
		{Element: "土", Valid: true, Score: 30, PrimaryRequired: true, ResolvesPrimary: false, Breakdown: ScoreBreakdown{MarginalUtility: 1}},
	}}
	selectRoles(&out, RuleV1())
	if out.Primary != "" || len(out.Favorable) != 0 {
		t.Fatalf("failed primary conditions must not be favorable: %+v", out)
	}
	for _, candidate := range out.Candidates {
		if candidate.Role != "pending_review" {
			t.Fatalf("candidate %s role=%s", candidate.Element, candidate.Role)
		}
	}
}

type wetColdSimulator struct{ base State }

func (f wetColdSimulator) Simulate(element string, delta float64) (State, error) {
	s := f.base
	s.ElementRatio = copyRatios(f.base.ElementRatio)
	s.ElementRatio[element] += delta
	if element == "火" {
		s.StrengthScore += delta * 1.37
		s.Temperature += delta * .59
		s.Moisture -= delta * .51
		s.DiseaseSeverity -= delta * .51
	}
	return s, nil
}

func TestEvaluateUsesSameDynamicWeightsForSevereWetColdSimulation(t *testing.T) {
	d := &Disease{Code: "WET", Severity: 73.83, CandidateElements: []string{"火"}}
	base := State{
		DayElement: "火", StrengthScore: 16.96, Temperature: -71.97, Moisture: 73.83,
		FlowScore: 0, PatternIntegrity: 50, DiseaseSeverity: 73.83, PrimaryDisease: d,
		ElementRatio: map[string]float64{"木": 22, "火": 17, "土": 19, "金": 20, "水": 22},
	}
	got, err := Evaluate(base, wetColdSimulator{base: base}, RuleV1())
	if err != nil {
		t.Fatal(err)
	}
	if got.Primary != "火" {
		t.Fatalf("severe wet/cold remedy should be selected after positive rule-aligned simulation: primary=%q result=%+v", got.Primary, got)
	}
	if got.RuleVersion != "useful-god-v1.1" {
		t.Fatalf("rule version=%q", got.RuleVersion)
	}
}
