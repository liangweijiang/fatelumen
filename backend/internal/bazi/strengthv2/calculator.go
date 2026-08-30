package strengthv2

import (
	"fmt"
	"math"
	"sort"
)

func Evaluate(in Input) (Result, error) {
	if in.DayElement == "" || in.SeasonCoefficient <= 0 {
		return Result{}, fmt.Errorf("day element and season coefficient are required")
	}
	c := in.Categories
	support := c.Peer + c.Resource*.85
	pressure := c.Output*.70 + c.Wealth*.75 + c.Officer*.95
	denom := support + pressure
	base := 0.0
	if denom > 0 {
		base = 100 * (support - pressure) / denom
	}
	deling := deLing(in.SeasonCoefficient)
	dedi := clamp(in.RootPower/2.5*100, 0, 100)
	deshi := 0.0
	total := c.Peer + c.Resource + c.Output + c.Wealth + c.Officer
	if total > 0 {
		deshi = (c.Peer + c.Resource*.85) / total * 100
	}
	final := base*.70 + (deling-50)*2*.10 + (dedi-50)*2*.12 + (deshi-50)*2*.08
	final = clamp(final, -100, 100)
	level := level(final)
	confidence := confidence(final, in)
	r := Result{RuleVersion: RuleVersionV2, Level: level, BaseScore: round(base), Score: round(final), Confidence: round4(confidence), Support: round(support), Pressure: round(pressure), DeLing: round(deling), DeDi: round(dedi), DeShi: round(deshi), RootPower: round4(in.RootPower)}
	if denom > 0 {
		r.SupportRatio = round4(support / denom)
	}
	r.Patterns = patterns(in, r)
	r.Trace = []Trace{
		{Rule: RuleVersionV2 + ".support_pressure", Result: level, Reason: "peer*1 + resource*.85 versus output*.70 + wealth*.75 + officer*.95", Values: map[string]float64{"support": r.Support, "pressure": r.Pressure}},
		{Rule: RuleVersionV2 + ".base", Result: level, Reason: "100*(support-pressure)/(support+pressure)", Score: r.BaseScore},
		{Rule: RuleVersionV2 + ".stability", Result: level, Reason: "base*.70 + deLing*.10 + deDi*.12 + deShi*.08 after normalization", Score: r.Score, Values: map[string]float64{"de_ling": r.DeLing, "de_di": r.DeDi, "de_shi": r.DeShi}},
	}
	return r, nil
}

func deLing(v float64) float64 {
	switch {
	case v <= .5:
		return 10
	case v <= .75:
		return 25
	case v <= 1:
		return 45
	case v <= 1.25:
		return 60
	case v <= 1.5:
		return 80
	default:
		return 100
	}
}
func level(v float64) string {
	switch {
	case v < -75:
		return "extremely_weak"
	case v < -45:
		return "weak"
	case v < -15:
		return "slightly_weak"
	case v <= 15:
		return "balanced"
	case v <= 45:
		return "slightly_strong"
	case v <= 75:
		return "strong"
	default:
		return "extremely_strong"
	}
}
func confidence(score float64, in Input) float64 {
	boundaries := []float64{-75, -45, -15, 15, 45, 75}
	dist := 200.0
	for _, b := range boundaries {
		d := math.Abs(score - b)
		if d < dist {
			dist = d
		}
	}
	c := .55 + math.Min(dist/30, .35)
	if in.SeasonProgress < .03 || in.SeasonProgress > .97 {
		c -= .10
	}
	if in.HourUnknown {
		c -= .10
	}
	for _, s := range in.Structures {
		if s.Confidence >= 50 && s.Confidence < 75 {
			c -= .15
			break
		}
	}
	return clamp(c, .2, .95)
}
func patterns(in Input, r Result) []PatternCandidate {
	out := []PatternCandidate{}
	c := in.Categories
	total := c.Peer + c.Resource + c.Output + c.Wealth + c.Officer
	dominantName, dominant := dominantCategory(c)
	dominantRatio := 0.0
	if total > 0 {
		dominantRatio = dominant / total
	}
	follow := PatternCandidate{Type: "following", Matched: r.Score <= -75 && r.DeDi <= 15 && r.SupportRatio <= .15 && dominantRatio >= .65, Confidence: round4(math.Min(1, dominantRatio)), Evidence: []string{fmt.Sprintf("strength=%.2f root=%.2f support_ratio=%.4f dominant=%s:%.4f", r.Score, r.DeDi, r.SupportRatio, dominantName, dominantRatio)}}
	if follow.Matched {
		switch dominantName {
		case "wealth":
			follow.Subtype = "follow_wealth"
		case "officer":
			follow.Subtype = "follow_officer"
		case "output":
			follow.Subtype = "follow_output"
		default:
			follow.Subtype = "follow_momentum"
		}
		if r.DeDi > 10 {
			follow.Alternative = "extremely_weak_normal"
			follow.Confidence *= .8
		}
	}
	if r.DeDi > 15 {
		follow.RejectedBy = append(follow.RejectedBy, "effective_root_present")
	}
	if r.SupportRatio > .15 {
		follow.RejectedBy = append(follow.RejectedBy, "support_not_minimal")
	}
	out = append(out, follow)
	dominantPattern := PatternCandidate{Type: "dominant", Matched: r.Score >= 75 && r.SupportRatio >= .75, Confidence: round4(math.Max(r.SupportRatio, dominantRatio)), Subtype: dominantSubtype(in.DayElement), Evidence: []string{fmt.Sprintf("strength=%.2f support_ratio=%.4f", r.Score, r.SupportRatio)}}
	out = append(out, dominantPattern)
	specialMatched := follow.Matched || dominantPattern.Matched
	for _, s := range in.Structures {
		if s.Type == "combination" && s.State == "full" && in.ElementRatios[s.TargetElement] >= 65 {
			out = append(out, PatternCandidate{Type: "transformation", Subtype: "transform_" + s.TargetElement, Matched: true, Confidence: round4(s.Confidence / 100), Evidence: []string{s.Code, fmt.Sprintf("target_ratio=%.4f", in.ElementRatios[s.TargetElement])}})
			specialMatched = true
		}
	}
	if !specialMatched {
		out = append(out, PatternCandidate{Type: "normal", Matched: true, Confidence: r.Confidence, Evidence: []string{"no confirmed special pattern"}})
	}
	return out
}
func dominantCategory(c CategoryPower) (string, float64) {
	v := []struct {
		n string
		x float64
	}{{"peer", c.Peer}, {"resource", c.Resource}, {"output", c.Output}, {"wealth", c.Wealth}, {"officer", c.Officer}}
	sort.SliceStable(v, func(i, j int) bool { return v[i].x > v[j].x })
	return v[0].n, v[0].x
}
func dominantSubtype(e string) string {
	return map[string]string{"木": "curved_straight", "火": "flame_upward", "土": "earthwork", "金": "metal_reform", "水": "water_downward"}[e]
}
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func round(v float64) float64  { return math.Round(v*100) / 100 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
