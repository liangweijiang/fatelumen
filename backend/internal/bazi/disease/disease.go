package disease

import (
	"math"
	"sort"
)

const RuleVersionV1 = "disease-v1.0"

type RuleSet struct {
	Version                           string
	ExcessThreshold, ClimateThreshold float64
}
type Input struct {
	DayElement                           string
	StrengthScore, Temperature, Moisture float64
	Categories                           map[string]float64
	ConflictCount                        int
}
type Item struct {
	Code, Level                 string
	Severity                    float64
	CandidateElements, Evidence []string
}
type Result struct {
	RuleVersion      string
	Primary          *Item
	Secondary, Minor []Item
	All              []Item
	Warnings         []string
}

func RuleV1() RuleSet { return RuleSet{RuleVersionV1, 30, 40} }
func Evaluate(in Input, r RuleSet) Result {
	items := []Item{}
	add := func(code string, sev float64, candidates []string, evidence string) {
		if sev <= 0 {
			return
		}
		items = append(items, Item{code, level(sev), round(sev), candidates, []string{evidence}})
	}
	if in.StrengthScore <= -15 {
		add("DAY_MASTER_WEAK", math.Abs(in.StrengthScore), []string{resourceOf(in.DayElement), in.DayElement}, "strength score below -15")
	}
	if in.StrengthScore >= 15 {
		add("DAY_MASTER_STRONG", math.Abs(in.StrengthScore), []string{outputOf(in.DayElement), wealthOf(in.DayElement), officerOf(in.DayElement)}, "strength score above 15")
	}
	for _, x := range []struct{ k, c string }{{"peer", "PEER_EXCESS"}, {"resource", "RESOURCE_EXCESS"}, {"output", "OUTPUT_EXCESS"}, {"wealth", "WEALTH_EXCESS"}, {"officer", "OFFICER_KILLING_EXCESS"}} {
		if v := in.Categories[x.k]; v >= r.ExcessThreshold {
			add(x.c, (v-r.ExcessThreshold)/(100-r.ExcessThreshold)*100, remedyForCategory(x.k, in.DayElement), "effective category exceeds threshold")
		}
	}
	if in.Temperature <= -r.ClimateThreshold {
		add("COLD", math.Abs(in.Temperature), []string{"火"}, "temperature below climate threshold")
	}
	if in.Temperature >= r.ClimateThreshold {
		add("HOT", math.Abs(in.Temperature), []string{"水"}, "temperature above climate threshold")
	}
	if in.Moisture <= -r.ClimateThreshold {
		add("DRY", math.Abs(in.Moisture), []string{"水"}, "moisture below climate threshold")
	}
	if in.Moisture >= r.ClimateThreshold {
		add("WET", math.Abs(in.Moisture), []string{"火"}, "moisture above climate threshold")
	}
	if in.ConflictCount > 0 {
		add("ELEMENT_CONFLICT", math.Min(100, float64(in.ConflictCount)*25), nil, "mediation conflict detected")
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Severity > items[j].Severity })
	out := Result{RuleVersion: r.Version, All: items, Warnings: []string{"disease_severity_coefficients_require_domain_calibration"}}
	if len(items) > 0 {
		v := items[0]
		out.Primary = &v
	}
	if len(items) > 1 {
		out.Secondary = append(out.Secondary, items[1])
	}
	if len(items) > 2 {
		out.Minor = append(out.Minor, items[2:]...)
	}
	return out
}
func level(v float64) string {
	if v > 70 {
		return "severe"
	}
	if v >= 40 {
		return "obvious"
	}
	return "minor"
}
func round(v float64) float64 { return math.Round(v*100) / 100 }
func outputOf(e string) string {
	return map[string]string{"木": "火", "火": "土", "土": "金", "金": "水", "水": "木"}[e]
}
func resourceOf(e string) string {
	return map[string]string{"木": "水", "火": "木", "土": "火", "金": "土", "水": "金"}[e]
}
func wealthOf(e string) string {
	return map[string]string{"木": "土", "火": "金", "土": "水", "金": "木", "水": "火"}[e]
}
func officerOf(e string) string {
	return map[string]string{"木": "金", "火": "水", "土": "木", "金": "火", "水": "土"}[e]
}
func remedyForCategory(category, day string) []string {
	switch category {
	case "peer":
		return []string{officerOf(day), outputOf(day)}
	case "resource":
		return []string{wealthOf(day)}
	case "output":
		return []string{resourceOf(day)}
	case "wealth":
		return []string{day}
	case "officer":
		return []string{resourceOf(day)}
	}
	return nil
}
