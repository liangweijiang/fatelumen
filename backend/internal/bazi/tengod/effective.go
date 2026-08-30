package tengod

import (
	"fmt"
	"sort"
)

const EffectiveRuleVersionV2 = "ten-god-effective-v2.0"

type EffectiveInput struct {
	Raw           Result
	ElementRatios map[string]float64
}

// EvaluateEffective preserves the stem/hidden-stem yin-yang split from the
// raw analysis and allocates each effective five-element category by that raw
// split. It never reconstructs proper/偏 ten gods from aggregate elements.
func EvaluateEffective(in EffectiveInput) (Result, error) {
	if in.Raw.DayElement == "" || len(in.Raw.Gods) != 10 {
		return Result{}, fmt.Errorf("complete raw ten-god analysis is required")
	}
	categoryEffective, err := effectiveCategories(in.Raw.DayElement, in.ElementRatios)
	if err != nil {
		return Result{}, err
	}
	rawTotals := map[string]float64{}
	for _, g := range in.Raw.Gods {
		rawTotals[g.Category] += g.RawScore
	}
	out := in.Raw
	out.RuleVersion = EffectiveRuleVersionV2
	out.TotalScore = 0
	out.DominantGods, out.SecondaryGods, out.MissingGods = nil, nil, nil
	out.VisibleGods = append([]string{}, in.Raw.VisibleGods...)
	out.RootedGods = append([]string{}, in.Raw.RootedGods...)
	out.Evidence = append([]Evidence{}, in.Raw.Evidence...)
	for i := range out.Gods {
		g := &out.Gods[i]
		effective := 0.0
		if rawTotals[g.Category] > 0 {
			effective = categoryEffective[g.Category] * g.RawScore / rawTotals[g.Category]
		}
		g.EffectiveScore = round(effective)
		out.TotalScore += g.EffectiveScore
		out.Evidence = append(out.Evidence, Evidence{Code: EffectiveRuleVersionV2 + ".allocation", Source: "effective_element_power", TenGod: g.Name, Category: g.Category, RawScore: g.RawScore, Adjustment: round(g.EffectiveScore - g.RawScore), Reason: "effective category allocated by original stem/hidden-stem yin-yang share"})
	}
	for i := range out.Gods {
		if out.TotalScore > 0 {
			out.Gods[i].Ratio = round4(out.Gods[i].EffectiveScore / out.TotalScore)
		} else {
			out.Gods[i].Ratio = 0
		}
	}
	sort.SliceStable(out.Gods, func(i, j int) bool {
		if out.Gods[i].EffectiveScore == out.Gods[j].EffectiveScore {
			return out.Gods[i].Code < out.Gods[j].Code
		}
		return out.Gods[i].EffectiveScore > out.Gods[j].EffectiveScore
	})
	for i := range out.Gods {
		out.Gods[i].Rank = i + 1
		g := out.Gods[i]
		if g.EffectiveScore == 0 {
			out.MissingGods = append(out.MissingGods, g.Name)
		} else if i < 2 {
			out.DominantGods = append(out.DominantGods, g.Name)
		} else if i < 5 {
			out.SecondaryGods = append(out.SecondaryGods, g.Name)
		}
	}
	out.Categories = nil
	for _, category := range []string{"peer", "resource", "output", "wealth", "officer"} {
		out.Categories = append(out.Categories, CategoryScore{Category: category, RawScore: round(rawTotals[category]), EffectiveScore: round(categoryEffective[category]), Ratio: round4(categoryEffective[category] / 100)})
	}
	sort.SliceStable(out.Categories, func(i, j int) bool {
		if out.Categories[i].EffectiveScore == out.Categories[j].EffectiveScore {
			return out.Categories[i].Category < out.Categories[j].Category
		}
		return out.Categories[i].EffectiveScore > out.Categories[j].EffectiveScore
	})
	for i := range out.Categories {
		out.Categories[i].Rank = i + 1
	}
	if len(out.Gods) > 0 && out.TotalScore > 0 {
		out.Concentration = round4(out.Gods[0].EffectiveScore / out.TotalScore)
	}
	return out, nil
}

func effectiveCategories(day string, ratios map[string]float64) (map[string]float64, error) {
	cycle := map[string]string{"木": "火", "火": "土", "土": "金", "金": "水", "水": "木"}
	controls := map[string]string{"木": "土", "土": "水", "水": "火", "火": "金", "金": "木"}
	if cycle[day] == "" {
		return nil, fmt.Errorf("invalid day element %q", day)
	}
	out := map[string]float64{"peer": ratios[day], "output": ratios[cycle[day]], "wealth": ratios[controls[day]]}
	for e, target := range cycle {
		if target == day {
			out["resource"] = ratios[e]
		}
	}
	for e, target := range controls {
		if target == day {
			out["officer"] = ratios[e]
		}
	}
	return out, nil
}
