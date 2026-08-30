package tengod

import (
	"fmt"
	"math"
	"sort"

	"fatelumen/backend/internal/bazi/basedata"
)

type godState struct {
	rule            basedata.TenGodRule
	raw             float64
	visible, rooted bool
}

func Evaluate(in Input) (Result, error) {
	base := basedata.V1()
	if _, ok := base.Stem(in.DayStem); !ok {
		return Result{}, fmt.Errorf("invalid day stem %q", in.DayStem)
	}
	if in.DayElement == "" {
		return Result{}, fmt.Errorf("day element is required")
	}
	states := make(map[string]*godState, len(base.TenGodRules))
	nameToCode := make(map[string]string, len(base.TenGodRules))
	for _, rule := range base.TenGodRules {
		copy := rule
		states[rule.Code] = &godState{rule: copy}
		nameToCode[rule.Names.ZH] = rule.Code
	}
	evidence := make([]Evidence, 0, len(in.Contributions)+len(in.Relations))
	categoryRaw := map[string]float64{}
	categoryAdjustment := map[string]float64{}
	for _, c := range in.Contributions {
		if c.TenGod == "" { // month-command summary is not a ten-god occurrence.
			continue
		}
		code, ok := nameToCode[c.TenGod]
		if !ok {
			return Result{}, fmt.Errorf("unknown ten god %q at %s", c.TenGod, c.Position)
		}
		state := states[code]
		group := godCategory(state.rule.Code)
		state.raw += c.Score
		if c.Source == "stem" {
			state.visible = true
		}
		if c.Source == "hidden_stem" {
			state.rooted = true
		}
		categoryRaw[group] += c.Score
		evidence = append(evidence, Evidence{Code: RuleVersionV1 + ".occurrence", Source: c.Source, Position: c.Position, Symbols: []string{c.Symbol}, TenGod: c.TenGod, Category: group, RawScore: round(c.Score), Reason: "position weight and hidden-stem level projected from strength analysis"})
	}
	for _, rel := range in.Relations {
		if rel.Element == "" || rel.Score == 0 {
			continue
		}
		category, ok := categoryForElement(base, in.DayElement, rel.Element)
		if !ok {
			return Result{}, fmt.Errorf("cannot map relation element %q", rel.Element)
		}
		categoryAdjustment[category] += rel.Score
		evidence = append(evidence, Evidence{Code: rel.Code, Source: "strength_relation", Position: joinPositions(rel.Positions), Symbols: append([]string{}, rel.Symbols...), Category: category, Adjustment: round(rel.Score), Reason: rel.Reason})
	}

	gods := make([]GodScore, 0, len(base.TenGodRules))
	totalRaw := 0.0
	for _, state := range states {
		totalRaw += state.raw
	}
	for _, rule := range base.TenGodRules {
		state := states[rule.Code]
		ratio := 0.0
		if totalRaw > 0 {
			ratio = state.raw / totalRaw
		}
		gods = append(gods, GodScore{Code: rule.Code, Name: rule.Names.ZH, Category: godCategory(rule.Code), RawScore: round(state.raw), EffectiveScore: round(state.raw), Ratio: round4(ratio), Visible: state.visible, Rooted: state.rooted})
	}
	sort.SliceStable(gods, func(i, j int) bool {
		if gods[i].EffectiveScore == gods[j].EffectiveScore {
			return gods[i].Code < gods[j].Code
		}
		return gods[i].EffectiveScore > gods[j].EffectiveScore
	})
	for i := range gods {
		gods[i].Rank = i + 1
	}

	categories := make([]CategoryScore, 0, 5)
	for _, category := range []string{"peer", "resource", "output", "wealth", "officer"} {
		effective := math.Max(0, categoryRaw[category]+categoryAdjustment[category])
		categories = append(categories, CategoryScore{Category: category, RawScore: round(categoryRaw[category]), RelationAdjustment: round(categoryAdjustment[category]), EffectiveScore: round(effective)})
	}
	totalEffective := 0.0
	for _, c := range categories {
		totalEffective += c.EffectiveScore
	}
	sort.SliceStable(categories, func(i, j int) bool {
		if categories[i].EffectiveScore == categories[j].EffectiveScore {
			return categories[i].Category < categories[j].Category
		}
		return categories[i].EffectiveScore > categories[j].EffectiveScore
	})
	for i := range categories {
		categories[i].Rank = i + 1
		if totalEffective > 0 {
			categories[i].Ratio = round4(categories[i].EffectiveScore / totalEffective)
		}
	}

	result := Result{RuleVersion: RuleVersionV1, DayStem: in.DayStem, DayElement: in.DayElement, TotalScore: round(totalEffective), Gods: gods, Categories: categories, Evidence: evidence}
	for _, god := range gods {
		switch {
		case god.RawScore == 0:
			result.MissingGods = append(result.MissingGods, god.Name)
		case god.Rank <= 2:
			result.DominantGods = append(result.DominantGods, god.Name)
		case god.Rank <= 5:
			result.SecondaryGods = append(result.SecondaryGods, god.Name)
		}
		if god.Visible {
			result.VisibleGods = append(result.VisibleGods, god.Name)
		}
		if god.Rooted {
			result.RootedGods = append(result.RootedGods, god.Name)
		}
	}
	if len(gods) > 0 && totalRaw > 0 {
		result.Concentration = round4(gods[0].RawScore / totalRaw)
	}
	return result, nil
}

func categoryForElement(base basedata.Catalog, dayElement, other string) (string, bool) {
	for _, e := range base.Elements {
		if e.Code != dayElement {
			continue
		}
		switch other {
		case e.Code:
			return "peer", true
		case e.GeneratedBy:
			return "resource", true
		case e.Generates:
			return "output", true
		case e.Controls:
			return "wealth", true
		case e.ControlledBy:
			return "officer", true
		}
	}
	return "", false
}

func joinPositions(v []string) string {
	if len(v) == 0 {
		return ""
	}
	out := v[0]
	for _, x := range v[1:] {
		out += "," + x
	}
	return out
}
func godCategory(code string) string {
	for i, r := range code {
		if r == '.' {
			return code[:i]
		}
	}
	return code
}
func round(v float64) float64  { return math.Round(v*100) / 100 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
