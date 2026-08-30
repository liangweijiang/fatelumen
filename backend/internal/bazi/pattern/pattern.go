package pattern

import (
	"math"
	"sort"
)

const RuleVersionV1 = "pattern-review-v1.0"

type RuleSet struct {
	Version                           string
	PresentThreshold, StrongThreshold float64
}
type Candidate struct {
	Code, State                        string
	Confidence                         float64
	Requirements, RejectedBy, Evidence []string
}
type Input struct {
	StrengthScore, RootPower float64
	Gods, Categories         map[string]float64
	Existing                 []Candidate
}
type Result struct {
	RuleVersion string
	Primary     string
	Candidates  []Candidate
	Warnings    []string
}

func RuleV1() RuleSet { return RuleSet{RuleVersionV1, 8, 18} }
func Evaluate(in Input, r RuleSet) Result {
	out := Result{RuleVersion: r.Version, Warnings: []string{"pattern_presence_thresholds_require_domain_calibration"}}
	for _, c := range in.Existing {
		if len(c.RejectedBy) > 0 {
			c.State = "rejected"
		} else if c.State != "confirmed" {
			c.State = "candidate"
		}
		out.Candidates = append(out.Candidates, c)
	}
	for _, g := range []struct{ name, code string }{{"正官", "proper_officer"}, {"七杀", "seven_killings"}, {"正财", "proper_wealth"}, {"偏财", "indirect_wealth"}, {"正印", "proper_resource"}, {"偏印", "indirect_resource"}, {"食神", "food_god"}, {"伤官", "hurting_officer"}} {
		if p := in.Gods[g.name]; p >= r.PresentThreshold {
			out.Candidates = append(out.Candidates, Candidate{g.code, "candidate", confidence(p), []string{"effective ten-god present"}, nil, []string{g.name}})
		}
	}
	combos := []struct {
		code, a, b, godA, godB string
		weak                   bool
	}{{"officer_resource_cycle", "officer", "resource", "正官", "正印", false}, {"killing_resource_cycle", "officer", "resource", "七杀", "正印", true}, {"food_controls_killing", "output", "officer", "食神", "七杀", false}, {"output_resource_balance", "output", "resource", "伤官", "正印", false}, {"output_generates_wealth", "output", "wealth", "伤官", "", false}, {"food_generates_wealth", "output", "wealth", "食神", "", false}, {"wealth_generates_officer", "wealth", "officer", "", "正官", false}}
	for _, x := range combos {
		a, b := in.Categories[x.a], in.Categories[x.b]
		godAPresent := x.godA == "" || in.Gods[x.godA] >= r.PresentThreshold
		godBPresent := x.godB == "" || in.Gods[x.godB] >= r.PresentThreshold
		if a >= r.PresentThreshold && b >= r.PresentThreshold && godAPresent && godBPresent {
			reject := []string{}
			if x.weak && in.StrengthScore > 15 {
				reject = append(reject, "day_master_not_weak")
			}
			state := "candidate"
			if len(reject) > 0 {
				state = "rejected"
			}
			out.Candidates = append(out.Candidates, Candidate{x.code, state, round4(math.Min(a, b) / 100), []string{x.a + " present", x.b + " present", x.godA + "/" + x.godB + " evidence"}, reject, []string{"effective category and ten-god powers"}})
		}
	}
	sort.SliceStable(out.Candidates, func(i, j int) bool { return out.Candidates[i].Confidence > out.Candidates[j].Confidence })
	for _, c := range out.Candidates {
		if c.State == "confirmed" {
			out.Primary = c.Code
			break
		}
	}
	return out
}
func confidence(v float64) float64 { return round4(math.Min(.95, .45+v/100)) }
func round4(v float64) float64     { return math.Round(v*10000) / 10000 }
