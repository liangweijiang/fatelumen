package pattern

import "testing"

func TestEvaluateKeepsRejectedSpecialCandidate(t *testing.T) {
	r := Evaluate(Input{
		StrengthScore: -20,
		Categories:    map[string]float64{"officer": 25, "resource": 20},
		Gods:          map[string]float64{"正官": 12},
		Existing:      []Candidate{{Code: "following:follow_wealth", RejectedBy: []string{"effective_root_present"}}},
	}, RuleV1())
	foundRejected := false
	for _, candidate := range r.Candidates {
		if candidate.Code == "following:follow_wealth" && candidate.State == "rejected" {
			foundRejected = true
		}
	}
	if !foundRejected {
		t.Fatalf("result=%+v", r)
	}
	if r.Primary == "following:follow_wealth" {
		t.Fatal("rejected pattern became primary")
	}
}

func TestEvaluateOnlyConfirmedPatternCanBecomePrimary(t *testing.T) {
	r := Evaluate(Input{Existing: []Candidate{
		{Code: "dominant:flame_upward", State: "candidate", Confidence: .9},
		{Code: "normal:", State: "confirmed", Confidence: .8},
	}}, RuleV1())
	if r.Primary != "normal:" {
		t.Fatalf("candidate must not become primary: %+v", r)
	}
}
