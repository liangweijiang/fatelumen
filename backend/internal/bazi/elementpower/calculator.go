package elementpower

import (
	"fmt"
	"math"
	"sort"

	"fatelumen/backend/internal/bazi/basedata"
)

var elements = []string{"木", "火", "土", "金", "水"}

type occurrence struct{ position, symbol, element string }
type pillarAt struct {
	name string
	p    Pillar
}

func Evaluate(in Input, rules RuleSet) (Result, error) {
	if rules.Version == "" {
		return Result{}, fmt.Errorf("element-power rule version is required")
	}
	base := basedata.V1()
	if _, ok := rules.SeasonMatrix[in.Month.Branch]; !ok {
		return Result{}, fmt.Errorf("invalid month branch %q", in.Month.Branch)
	}
	if in.SeasonProgress < 0 || in.SeasonProgress > 1 {
		return Result{}, fmt.Errorf("season progress %.4f outside 0..1", in.SeasonProgress)
	}
	pillars := []pillarAt{{"year", in.Year}, {"month", in.Month}, {"day", in.Day}, {"hour", in.Hour}}
	visible := map[string]int{}
	for _, x := range pillars {
		s, ok := base.Stem(x.p.Stem)
		if !ok {
			return Result{}, fmt.Errorf("invalid %s stem %q", x.name, x.p.Stem)
		}
		if _, ok := base.Branch(x.p.Branch); !ok {
			return Result{}, fmt.Errorf("invalid %s branch %q", x.name, x.p.Branch)
		}
		visible[s.Element]++
	}
	next := nextBranch(in.Month.Branch)
	coeff := interpolate(rules.SeasonMatrix[in.Month.Branch], rules.SeasonMatrix[next], in.SeasonProgress)
	result := Result{RuleVersion: rules.Version, Season: Season{Branch: in.Month.Branch, NextBranch: next, Progress: round4(in.SeasonProgress), Coefficients: mapVector(coeff)}}
	if !rules.StructureEnabled {
		result.Warnings = []string{"structural_transformations_pending_v2_b"}
	}
	raw, seasonal := emptyMap(), emptyMap()
	occurrences := []occurrence{}
	for _, x := range pillars {
		stemPos := x.name + "_stem"
		stem, _ := base.Stem(x.p.Stem)
		r := rules.PositionWeights[stemPos]
		season := r * coeff[stem.Element]
		result.Contributions = append(result.Contributions, Contribution{Code: "stem", Position: stemPos, Symbol: stem.Code, Element: stem.Element, BaseWeight: r, HiddenRatio: 1, RawPower: round4(r), SeasonCoefficient: coeff[stem.Element], VisibilityMultiplier: 1, SeasonalPower: round4(season)})
		raw[stem.Element] += r
		seasonal[stem.Element] += season
		occurrences = append(occurrences, occurrence{stemPos, stem.Code, stem.Element})
		branchPos := x.name + "_branch"
		branch, _ := base.Branch(x.p.Branch)
		for i, h := range branch.HiddenStems {
			ratio := hiddenRatio(branch.HiddenStems, i, rules)
			weight := rules.PositionWeights[branchPos] * ratio
			hs, _ := base.Stem(h.Stem)
			multiplier := visibilityMultiplier(branchPos, i, visible[hs.Element])
			season := weight * coeff[hs.Element] * multiplier
			result.Contributions = append(result.Contributions, Contribution{Code: "hidden_stem", Position: branchPos, Symbol: h.Stem, Element: hs.Element, HiddenLevel: h.Level, BaseWeight: rules.PositionWeights[branchPos], HiddenRatio: ratio, RawPower: round4(weight), SeasonCoefficient: coeff[hs.Element], VisibilityMultiplier: multiplier, SeasonalPower: round4(season)})
			raw[hs.Element] += weight
			seasonal[hs.Element] += season
			occurrences = append(occurrences, occurrence{branchPos, h.Stem, hs.Element})
		}
	}
	result.RawPower, result.SeasonalPower = mapVector(raw), mapVector(seasonal)
	result.Roots, result.RootPower = roots(in.Day.Stem, pillars, base, rules, coeff)
	effective := cloneMap(seasonal)
	effective, result.Interactions = interactionRound(effective, occurrences, base, rules, 1, 1, result.Interactions)
	if rules.StructureEnabled {
		effective, result.Structures = applyStructures(effective, pillars, result.Contributions, in.LongLifeStages, base, rules, coeff)
	}
	effective, result.Interactions = interactionRound(effective, occurrences, base, rules, 2, .5, result.Interactions)
	result.EffectivePower = mapVector(effective)
	result.EffectiveRatio = normalizedVector(effective)
	result.Trace = []Trace{
		{Rule: rules.Version + ".raw", Source: "pillars", Reason: "position weights and hidden-stem ratios", Values: roundedMap(raw)},
		{Rule: rules.Version + ".season", Source: "solar_terms", Reason: "continuous interpolation between month-branch coefficients", Values: roundedMap(coeff)},
		{Rule: rules.Version + ".structure", Source: "structure_engine", Reason: "versioned combinations, clashes and availability modifiers", Values: map[string]float64{"matches": float64(len(result.Structures))}},
		{Rule: rules.Version + ".effective", Source: "interaction_engine", Reason: "generation/control, structure adjustment, then half-rate second pass", Values: roundedMap(effective)},
	}
	return result, nil
}

// SimulateIncrease reruns the deterministic natal engine, injects a
// standardized element increment, and reruns generation/control interactions.
// Natal structures are already reflected in the baseline effective vector.
func SimulateIncrease(in Input, rules RuleSet, element string, delta float64) (Vector, error) {
	baseResult, err := Evaluate(in, rules)
	if err != nil {
		return Vector{}, err
	}
	found := false
	for _, e := range elements {
		if e == element {
			found = true
		}
	}
	if !found || delta < 0 {
		return Vector{}, fmt.Errorf("invalid simulation element %q or delta %.4f", element, delta)
	}
	power := vectorMap(baseResult.EffectiveRatio)
	power[element] += delta
	catalog := basedata.V1()
	pillars := []pillarAt{{"year", in.Year}, {"month", in.Month}, {"day", in.Day}, {"hour", in.Hour}}
	occurrences := make([]occurrence, 0, 16)
	for _, p := range pillars {
		stem, _ := catalog.Stem(p.p.Stem)
		occurrences = append(occurrences, occurrence{p.name + "_stem", p.p.Stem, stem.Element})
		branch, _ := catalog.Branch(p.p.Branch)
		for _, hidden := range branch.HiddenStems {
			hs, _ := catalog.Stem(hidden.Stem)
			occurrences = append(occurrences, occurrence{p.name + "_branch", hidden.Stem, hs.Element})
		}
	}
	power, _ = interactionRound(power, occurrences, catalog, rules, 1, 1, nil)
	power, _ = interactionRound(power, occurrences, catalog, rules, 2, .5, nil)
	return normalizedVector(power), nil
}

func hiddenRatio(v []basedata.HiddenStem, i int, r RuleSet) float64 {
	if len(v) == 1 {
		return r.HiddenRatios["single"]
	}
	if len(v) == 2 {
		if i == 0 {
			return r.HiddenRatios["double_main"]
		}
		return r.HiddenRatios["double_residual"]
	}
	switch i {
	case 0:
		return r.HiddenRatios["main"]
	case 1:
		return r.HiddenRatios["middle"]
	default:
		return r.HiddenRatios["residual"]
	}
}
func visibilityMultiplier(position string, hiddenIndex, visibleCount int) float64 {
	if visibleCount == 0 {
		return 1
	}
	m := 1.05
	if position == "month_branch" {
		m = 1.12
		if hiddenIndex == 0 {
			m = 1.15
		}
	}
	if m > 1.2 {
		return 1.2
	}
	return m
}
func roots(dayStem string, pillars []pillarAt, base basedata.Catalog, rules RuleSet, coeff map[string]float64) ([]Root, float64) {
	day, _ := base.Stem(dayStem)
	out := []Root{}
	total := 0.0
	for _, x := range pillars {
		branch, _ := base.Branch(x.p.Branch)
		for i, h := range branch.HiddenStems {
			hs, _ := base.Stem(h.Stem)
			if hs.Element != day.Element {
				continue
			}
			ratio := hiddenRatio(branch.HiddenStems, i, rules)
			quality := .35
			if i == 0 {
				quality = .9
				if hs.YinYang == day.YinYang {
					quality = 1
				}
			} else if i == 1 {
				quality = .6
			}
			if x.name == "month" {
				quality *= 1.15
			}
			if x.name == "day" {
				quality *= 1.10
			}
			power := rules.PositionWeights[x.name+"_branch"] * ratio * coeff[day.Element] * quality
			out = append(out, Root{Position: x.name + "_branch", Branch: branch.Code, Stem: h.Stem, Level: h.Level, HiddenRatio: ratio, Quality: round4(quality), Power: round4(power)})
			total += power
		}
	}
	return out, round4(total)
}

func interactionRound(power map[string]float64, occurrences []occurrence, base basedata.Catalog, rules RuleSet, iteration int, factor float64, traces []Interaction) (map[string]float64, []Interaction) {
	delta := emptyMap()
	for _, e := range base.Elements {
		source, target := e.Code, e.Generates
		demand := 1 - power[target]/math.Max(total(power), .0001)
		if demand < .1 {
			demand = .1
		}
		contact := contactFactor(source, target, occurrences)
		transfer := power[source] * rules.GenerationRate * factor * demand * contact
		delta[source] -= transfer
		delta[target] += transfer * rules.GenerationEfficiency
		traces = append(traces, Interaction{Iteration: iteration, Type: "generation", SourceElement: source, TargetElement: target, SourceBefore: round4(power[source]), TargetBefore: round4(power[target]), Amount: round4(transfer), Efficiency: rules.GenerationEfficiency, ContactFactor: contact, RatioFactor: 1})
	}
	for _, e := range base.Elements {
		attacker, defender := e.Code, e.Controls
		contact := contactFactor(attacker, defender, occurrences)
		ratio := power[attacker] / math.Max(power[defender], .0001)
		rf := ratioFactor(ratio)
		damage := power[attacker] * rules.ControlRate * factor * contact * rf
		if damage > power[defender]+delta[defender] {
			damage = math.Max(0, power[defender]+delta[defender])
		}
		delta[defender] -= damage
		traces = append(traces, Interaction{Iteration: iteration, Type: "control", SourceElement: attacker, TargetElement: defender, SourceBefore: round4(power[attacker]), TargetBefore: round4(power[defender]), Amount: round4(damage), Efficiency: 1, ContactFactor: contact, RatioFactor: rf})
	}
	out := emptyMap()
	for _, e := range elements {
		out[e] = math.Max(0, power[e]+delta[e])
	}
	return out, traces
}

func applyStructures(power map[string]float64, pillars []pillarAt, contributions []Contribution, longLife map[string]string, base basedata.Catalog, rules RuleSet, coeff map[string]float64) (map[string]float64, []StructureRelation) {
	stemPos, branchPos := symbolPositions(pillars)
	rooted := map[string]bool{}
	for _, c := range contributions {
		if c.Code == "hidden_stem" {
			rooted[c.Element] = true
		}
	}
	matchedFull := map[string]bool{}
	relations := append([]basedata.Relation{}, base.Relations...)
	sort.SliceStable(relations, func(i, j int) bool {
		return structurePriority(relations[i].Type) < structurePriority(relations[j].Type)
	})
	for _, rel := range relations {
		if rel.Type != "three_harmony" {
			continue
		}
		if _, ok := matchMembers(rel.Members, branchPos); ok {
			matchedFull[rel.Element] = true
		}
	}
	out := cloneMap(power)
	traces := []StructureRelation{}
	for _, rel := range relations {
		positions, ok := matchMembers(rel.Members, branchPos)
		if rel.Scope == "stem" {
			positions, ok = matchMembers(rel.Members, stemPos)
		}
		if !ok || (rel.Type == "half_harmony" && matchedFull[rel.Element]) {
			continue
		}
		before := mapVector(out)
		switch rel.Type {
		case "combination", "six_combination", "three_harmony", "half_harmony", "three_meeting":
			confidence, reason := transformationConfidence(rel, out, stemPos, rooted, coeff, base)
			state, rate := "untransformed", 1-rules.UntransformedRetention
			if confidence >= rules.TransformationFull*100 {
				state, rate = "full", .70
			} else if confidence >= rules.TransformationPartial*100 {
				state, rate = "partial", .35
			}
			if rel.Type == "half_harmony" && !containsMiddle(rel.Members) {
				state += "_arch"
			}
			out = applyTransformation(out, rel, positions, contributions, base, state, rate)
			traces = append(traces, StructureRelation{Code: rel.Code, Type: rel.Type, TargetElement: rel.Element, State: state, Positions: positions, Symbols: append([]string{}, rel.Members...), Confidence: confidence, TransferRate: rate, Before: before, After: mapVector(out), Reason: reason})
		case "clash":
			out, traces = applyClash(out, rel, positions, contributions, base, traces, before)
		case "punishment", "harm", "break":
			retention := rules.PunishmentRetention
			if rel.Type == "harm" {
				retention = rules.HarmRetention
			}
			if rel.Type == "break" {
				retention = rules.BreakRetention
			}
			out = applyAvailability(out, positions, contributions, retention)
			traces = append(traces, StructureRelation{Code: rel.Code, Type: rel.Type, State: "availability_reduced", Positions: positions, Symbols: append([]string{}, rel.Members...), Confidence: 100, TransferRate: 1 - retention, Before: before, After: mapVector(out), Reason: fmt.Sprintf("availability retention %.2f", retention)})
		}
	}
	for _, p := range pillars {
		if p.p.Branch == "辰" || p.p.Branch == "戌" || p.p.Branch == "丑" || p.p.Branch == "未" {
			traces = append(traces, StructureRelation{Code: "tomb_storage." + p.name, Type: "tomb_storage", State: "stored_qi_preserved", Positions: []string{p.name + "_branch"}, Symbols: []string{p.p.Branch}, Confidence: 100, Before: mapVector(out), After: mapVector(out), Reason: "hidden stems already carry power; tomb storage creates no extra energy"})
		}
		if stage := longLife[p.name]; stage != "" {
			traces = append(traces, StructureRelation{Code: "long_life." + p.name, Type: "long_life", State: "secondary_neutral", Positions: []string{p.name + "_branch"}, Symbols: []string{stage}, Confidence: 100, Before: mapVector(out), After: mapVector(out), Reason: "stage retained as evidence; neutral 1.00 modifier until calibrated"})
		}
	}
	return out, traces
}

func structurePriority(t string) int {
	switch t {
	case "three_meeting":
		return 10
	case "three_harmony":
		return 20
	case "half_harmony":
		return 30
	case "combination":
		return 40
	case "six_combination":
		return 50
	case "clash":
		return 60
	case "punishment":
		return 70
	case "harm":
		return 80
	case "break":
		return 90
	}
	return 100
}

func symbolPositions(pillars []pillarAt) (map[string][]string, map[string][]string) {
	stems, branches := map[string][]string{}, map[string][]string{}
	for _, p := range pillars {
		stems[p.p.Stem] = append(stems[p.p.Stem], p.name+"_stem")
		branches[p.p.Branch] = append(branches[p.p.Branch], p.name+"_branch")
	}
	return stems, branches
}
func matchMembers(members []string, positions map[string][]string) ([]string, bool) {
	used := map[string]int{}
	out := make([]string, 0, len(members))
	for _, symbol := range members {
		list := positions[symbol]
		i := used[symbol]
		if i >= len(list) {
			return nil, false
		}
		out = append(out, list[i])
		used[symbol]++
	}
	return out, true
}
func transformationConfidence(rel basedata.Relation, power map[string]float64, visible map[string][]string, rooted map[string]bool, coeff map[string]float64, base basedata.Catalog) (float64, string) {
	score := 35.0
	switch rel.Type {
	case "three_harmony":
		score = 65
	case "half_harmony":
		score = 30
		if containsMiddle(rel.Members) {
			score += 15
		}
	case "three_meeting":
		score = 70
	}
	if coeff[rel.Element] >= 1.2 {
		score += 15
	} else if coeff[rel.Element] >= 1 {
		score += 8
	}
	if power[rel.Element]/math.Max(total(power), .0001) >= .20 {
		score += 10
	}
	if rooted[rel.Element] {
		score += 10
	}
	for _, s := range base.Stems {
		if s.Element == rel.Element && len(visible[s.Code]) > 0 {
			score += 10
			break
		}
	}
	controller := ""
	for _, e := range base.Elements {
		if e.Controls == rel.Element {
			controller = e.Code
			break
		}
	}
	if controller != "" && power[controller] > power[rel.Element]*1.2 {
		score -= 15
	}
	for _, m := range rel.Members {
		if len(visible[m]) > 1 {
			score -= 10
			break
		}
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return round4(score), fmt.Sprintf("base relation=%s; season=%.4f target_ratio=%.4f rooted=%t controller=%s", rel.Type, coeff[rel.Element], power[rel.Element]/math.Max(total(power), .0001), rooted[rel.Element], controller)
}
func containsMiddle(members []string) bool {
	for _, x := range members {
		if x == "子" || x == "卯" || x == "午" || x == "酉" {
			return true
		}
	}
	return false
}

func applyTransformation(power map[string]float64, rel basedata.Relation, positions []string, contributions []Contribution, base basedata.Catalog, state string, rate float64) map[string]float64 {
	out := cloneMap(power)
	moved := 0.0
	for _, pos := range positions {
		for _, c := range contributions {
			if c.Position != pos {
				continue
			}
			amount := c.SeasonalPower * rate
			if state == "untransformed" {
				out[c.Element] = math.Max(0, out[c.Element]-amount)
				continue
			}
			if c.Element == rel.Element {
				continue
			}
			amount = math.Min(amount, out[c.Element])
			out[c.Element] -= amount
			moved += amount
		}
	}
	if state != "untransformed" {
		out[rel.Element] += moved
	}
	return out
}
func applyAvailability(power map[string]float64, positions []string, contributions []Contribution, retention float64) map[string]float64 {
	out := cloneMap(power)
	for _, pos := range positions {
		for _, c := range contributions {
			if c.Position == pos {
				out[c.Element] = math.Max(0, out[c.Element]-c.SeasonalPower*(1-retention))
			}
		}
	}
	return out
}
func applyClash(power map[string]float64, rel basedata.Relation, positions []string, contributions []Contribution, base basedata.Catalog, traces []StructureRelation, before Vector) (map[string]float64, []StructureRelation) {
	if len(rel.Members) != 2 {
		return power, traces
	}
	b1, _ := base.Branch(rel.Members[0])
	b2, _ := base.Branch(rel.Members[1])
	p1, p2 := power[b1.Element], power[b2.Element]
	ratio := p1 / math.Max(p2, .0001)
	r1, r2 := .78, .78
	if ratio > 1.5 {
		r1, r2 = .90, .60
	} else if ratio < .67 {
		r1, r2 = .60, .90
	}
	out := cloneMap(power)
	out = applyAvailability(out, positions[:1], contributions, r1)
	out = applyAvailability(out, positions[1:], contributions, r2)
	state := "balanced_clash"
	if r1 != r2 {
		state = "strong_weak_clash"
	}
	traces = append(traces, StructureRelation{Code: rel.Code, Type: rel.Type, State: state, Positions: positions, Symbols: append([]string{}, rel.Members...), Confidence: 100, TransferRate: round4(1 - (r1+r2)/2), Before: before, After: mapVector(out), Reason: fmt.Sprintf("retention %.2f/%.2f ratio %.4f", r1, r2, ratio)})
	return out, traces
}

func contactFactor(a, b string, v []occurrence) float64 {
	best := .45
	for _, x := range v {
		if x.element == a {
			for _, y := range v {
				if y.element == b {
					f := positionContact(x.position, y.position)
					if f > best {
						best = f
					}
				}
			}
		}
	}
	return best
}
func positionContact(a, b string) float64 {
	pa, sa := positionIndex(a)
	pb, sb := positionIndex(b)
	if pa == pb {
		return 1
	}
	d := pa - pb
	if d < 0 {
		d = -d
	}
	if d == 1 {
		return .9
	}
	if d == 2 {
		return .65
	}
	if sa == sb {
		return .45
	}
	return .45
}
func positionIndex(v string) (int, string) {
	names := []string{"year", "month", "day", "hour"}
	for i, n := range names {
		if len(v) >= len(n) && v[:len(n)] == n {
			return i, v[len(n):]
		}
	}
	return 0, ""
}
func ratioFactor(v float64) float64 {
	switch {
	case v < .5:
		return .4
	case v < .8:
		return .7
	case v <= 1.2:
		return 1
	case v <= 2:
		return 1.2
	default:
		return 1.35
	}
}
func nextBranch(v string) string {
	order := []string{"寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥", "子", "丑"}
	for i, x := range order {
		if x == v {
			return order[(i+1)%len(order)]
		}
	}
	return ""
}
func interpolate(a, b map[string]float64, p float64) map[string]float64 {
	out := emptyMap()
	for _, e := range elements {
		out[e] = a[e]*(1-p) + b[e]*p
	}
	return out
}
func emptyMap() map[string]float64 {
	return map[string]float64{"木": 0, "火": 0, "土": 0, "金": 0, "水": 0}
}
func cloneMap(v map[string]float64) map[string]float64 {
	out := emptyMap()
	for k, x := range v {
		out[k] = x
	}
	return out
}
func total(v map[string]float64) float64 {
	n := 0.0
	for _, e := range elements {
		n += v[e]
	}
	return n
}
func mapVector(v map[string]float64) Vector {
	return Vector{round4(v["木"]), round4(v["火"]), round4(v["土"]), round4(v["金"]), round4(v["水"])}
}
func vectorMap(v Vector) map[string]float64 {
	return map[string]float64{"木": v.Wood, "火": v.Fire, "土": v.Earth, "金": v.Metal, "水": v.Water}
}
func normalizedVector(v map[string]float64) Vector {
	n := total(v)
	if n <= 0 {
		return Vector{}
	}
	return Vector{round4(v["木"] / n * 100), round4(v["火"] / n * 100), round4(v["土"] / n * 100), round4(v["金"] / n * 100), round4(v["水"] / n * 100)}
}
func roundedMap(v map[string]float64) map[string]float64 {
	out := emptyMap()
	keys := append([]string{}, elements...)
	sort.Strings(keys)
	for _, k := range keys {
		out[k] = round4(v[k])
	}
	return out
}
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
