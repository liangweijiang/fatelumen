package climate

import "testing"

func TestEvaluateWinterAndSummer(t *testing.T) {
	winter, err := Evaluate(Input{MonthBranch: "子", ElementRatio: map[string]float64{"火": 10, "水": 40}}, RuleV1())
	if err != nil {
		t.Fatal(err)
	}
	if winter.Temperature >= -70 || winter.TemperatureLevel != "severe_cold" {
		t.Fatalf("winter=%+v", winter)
	}
	summer, err := Evaluate(Input{MonthBranch: "午", ElementRatio: map[string]float64{"火": 40, "水": 10}}, RuleV1())
	if err != nil {
		t.Fatal(err)
	}
	if summer.Temperature <= 70 || summer.TemperatureLevel != "severe_hot" {
		t.Fatalf("summer=%+v", summer)
	}
}
