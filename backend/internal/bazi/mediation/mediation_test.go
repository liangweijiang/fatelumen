package mediation

import "testing"

func TestEvaluateConflictAndBridge(t *testing.T) {
	r := Evaluate(Input{ElementRatio: map[string]float64{"金": 25, "木": 20, "水": 12}}, RuleV1())
	if len(r.Conflicts) != 1 || r.Conflicts[0].Bridge != "水" || r.Conflicts[0].State != "present" {
		t.Fatalf("result=%+v", r)
	}
}
func TestEvaluateRejectsUnbalancedPair(t *testing.T) {
	r := Evaluate(Input{ElementRatio: map[string]float64{"金": 50, "木": 10, "水": 20}}, RuleV1())
	if len(r.Conflicts) != 0 {
		t.Fatalf("result=%+v", r)
	}
}
