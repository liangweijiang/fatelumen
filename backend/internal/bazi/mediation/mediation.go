package mediation

import "math"

const RuleVersionV1 = "mediation-v1.0"

type RuleSet struct {
	Version                                                         string
	MinimumConflictPower, MinRatio, MaxRatio, SufficientBridgeRatio float64
}
type Input struct{ ElementRatio map[string]float64 }
type Conflict struct {
	Controller, Controlled, Bridge, State                            string
	ControllerPower, ControlledPower, BridgePower, Ratio, Confidence float64
	Evidence                                                         []string
}
type Result struct {
	RuleVersion string
	FlowScore   float64
	Conflicts   []Conflict
	Warnings    []string
}

func RuleV1() RuleSet { return RuleSet{RuleVersionV1, 15, .60, 1.67, .50} }
func Evaluate(in Input, r RuleSet) Result {
	pairs := [][3]string{{"金", "木", "水"}, {"木", "土", "火"}, {"土", "水", "金"}, {"水", "火", "木"}, {"火", "金", "土"}}
	out := Result{RuleVersion: r.Version}
	flow := 0.0
	for _, p := range pairs {
		a, b, bridgePower := in.ElementRatio[p[0]], in.ElementRatio[p[1]], in.ElementRatio[p[2]]
		if a <= r.MinimumConflictPower || b <= r.MinimumConflictPower {
			continue
		}
		ratio := a / b
		if ratio < r.MinRatio || ratio > r.MaxRatio {
			continue
		}
		need := math.Min(a, b)
		state := "missing"
		if bridgePower >= need*r.SufficientBridgeRatio {
			state = "present"
			flow += 10
		} else if bridgePower > 0 {
			state = "insufficient"
			flow -= 5
		} else {
			flow -= 10
		}
		out.Conflicts = append(out.Conflicts, Conflict{p[0], p[1], p[2], state, round(a), round(b), round(bridgePower), round4(ratio), confidence(ratio), []string{"both sides exceed minimum conflict power", "ratio is within 0.60..1.67"}})
	}
	out.FlowScore = clamp(flow, -100, 100)
	out.Warnings = []string{"minimum_conflict_and_bridge_thresholds_require_domain_calibration"}
	return out
}
func confidence(r float64) float64 {
	d := math.Abs(math.Log(r))
	return round4(clamp(.9-d*.35, .5, .95))
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
