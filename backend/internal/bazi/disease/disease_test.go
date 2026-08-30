package disease

import "testing"

func TestEvaluateRanksSevereClimate(t *testing.T) {
	r := Evaluate(Input{StrengthScore: -30, Temperature: -80, Moisture: 20, Categories: map[string]float64{}}, RuleV1())
	if r.Primary == nil || r.Primary.Code != "COLD" {
		t.Fatalf("result=%+v", r)
	}
}
func TestMissingElementIsNotDisease(t *testing.T) {
	r := Evaluate(Input{Categories: map[string]float64{"peer": 20}}, RuleV1())
	for _, x := range r.All {
		if x.Code == "ELEMENT_MISSING" {
			t.Fatal("missing element must not be disease")
		}
	}
}
