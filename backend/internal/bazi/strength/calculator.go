package strength

import (
	"fmt"
	"math"
	"sort"
)

type ledger struct {
	support, restraint float64
	contributions      []Contribution
	relations          []Relation
	categoryTotals     map[string]float64
}

func Evaluate(in Input, rules RuleSet) (Result, error) {
	if rules.Version == "" {
		return Result{}, fmt.Errorf("strength rule version is required")
	}
	for pos, p := range map[string]Pillar{"year": in.Year, "month": in.Month, "day": in.Day, "hour": in.Hour} {
		if _, ok := stemElement[p.Stem]; !ok {
			return Result{}, fmt.Errorf("invalid %s stem %q", pos, p.Stem)
		}
		if _, ok := rules.HiddenStems[p.Branch]; !ok {
			return Result{}, fmt.Errorf("invalid %s branch %q", pos, p.Branch)
		}
	}
	dayElement := stemElement[in.Day.Stem]
	l := &ledger{categoryTotals: map[string]float64{}}
	monthElement := stemElement[rules.HiddenStems[in.Month.Branch][0].Stem]
	monthScore := rules.MonthScore[monthElement][dayElement]
	l.add("month_command", "month_command", "month_branch", in.Month.Branch, monthElement, "", monthScore)

	for _, x := range []struct {
		pos string
		p   Pillar
	}{{"year_stem", in.Year}, {"month_stem", in.Month}, {"hour_stem", in.Hour}} {
		god := tenGod(in.Day.Stem, x.p.Stem)
		l.add("visible_stem", "stem", x.pos, x.p.Stem, stemElement[x.p.Stem], god, rules.PositionWeight[x.pos])
	}
	for _, x := range []struct {
		pos string
		p   Pillar
	}{{"year_branch", in.Year}, {"month_branch", in.Month}, {"day_branch", in.Day}, {"hour_branch", in.Hour}} {
		for i, hs := range rules.HiddenStems[x.p.Branch] {
			score := rules.PositionWeight[x.pos] * hs.Weight
			if x.pos == "month_branch" && i == 0 {
				score *= .3
			}
			l.add("hidden_stem", "hidden_stem", x.pos, hs.Stem, stemElement[hs.Stem], tenGod(in.Day.Stem, hs.Stem), score)
		}
	}
	applyRelations(in, rules, monthElement, dayElement, l)
	root := rootLevel(in, rules, dayElement, l)
	total := l.support + l.restraint
	ratio := .5
	if total > 0 {
		ratio = l.support / total
	}
	level, pattern, subtype, falseFollow := classify(in, rules, monthScore, ratio, root, l)
	return Result{RuleVersion: rules.Version, Level: level, DayElement: dayElement, MonthScore: monthScore,
		SupportScore: round(l.support), RestraintScore: round(l.restraint), SupportRatio: math.Round(ratio*10000) / 10000,
		RootLevel: root, Pattern: pattern, PatternSubtype: subtype, FalseFollowing: falseFollow,
		Contributions: l.contributions, Relations: l.relations}, nil
}

func (l *ledger) add(code, source, pos, symbol, element, god string, score float64) {
	cat := category(god)
	if god == "" {
		cat = "support"
		if score < 0 {
			cat = "restraint"
			score = -score
		}
	}
	if cat == "support" {
		l.support += score
	} else {
		l.restraint += score
	}
	l.categoryTotals[group(god)] += score
	l.contributions = append(l.contributions, Contribution{Code: code, Source: source, Position: pos, Symbol: symbol, Element: element, TenGod: god, Category: cat, Score: round(score)})
}
func (l *ledger) adjust(cat string, amount float64) {
	if cat == "support" {
		l.support = math.Max(0, l.support+amount)
	} else {
		l.restraint = math.Max(0, l.restraint+amount)
	}
}
func (l *ledger) relation(code, typ, element, reason string, positions, symbols []string, score float64, transformed bool, dayElement string) {
	cat := elementCategory(dayElement, element)
	l.adjust(cat, score)
	l.relations = append(l.relations, Relation{Code: code, Type: typ, Element: element, Reason: reason, Positions: positions, Symbols: symbols, Score: score, Transformed: transformed})
}
func group(god string) string {
	switch god {
	case "正印", "偏印":
		return "resource"
	case "比肩", "劫财":
		return "peer"
	case "正财", "偏财":
		return "wealth"
	case "正官", "七杀":
		return "officer"
	case "食神", "伤官":
		return "output"
	}
	return ""
}
func round(v float64) float64 { return math.Round(v*100) / 100 }

func rootLevel(in Input, r RuleSet, dayElement string, l *ledger) string {
	best := 0.0
	for _, x := range []struct {
		pos string
		p   Pillar
	}{{"year_branch", in.Year}, {"month_branch", in.Month}, {"day_branch", in.Day}, {"hour_branch", in.Hour}} {
		for _, h := range r.HiddenStems[x.p.Branch] {
			if stemElement[h.Stem] == dayElement && h.Weight > best {
				best = h.Weight
			}
		}
	}
	if best >= .6 {
		return "strong"
	}
	if best >= .3 {
		return "medium"
	}
	if best >= .1 {
		return "weak"
	}
	return "none"
}

func classify(in Input, r RuleSet, monthScore, ratio float64, root string, l *ledger) (level, pattern, subtype string, falseFollow bool) {
	hasOfficer, hasSupport := false, false
	for _, p := range []Pillar{in.Year, in.Month, in.Hour} {
		g := tenGod(in.Day.Stem, p.Stem)
		if g == "正官" || g == "七杀" {
			hasOfficer = true
		}
		if category(g) == "support" {
			hasSupport = true
		}
	}
	officerTransformed := allVisibleMatchedByTransformation(in, l, func(g string) bool { return g == "正官" || g == "七杀" })
	supportTransformed := allVisibleMatchedByTransformation(in, l, func(g string) bool { return category(g) == "support" })
	total := l.support + l.restraint
	maxRes := math.Max(l.categoryTotals["wealth"], math.Max(l.categoryTotals["officer"], l.categoryTotals["output"]))
	maxSup := math.Max(l.categoryTotals["resource"], l.categoryTotals["peer"])
	if ratio >= r.FollowStrongThreshold && (root == "strong" || root == "medium") && monthScore > 0 && l.restraint <= total*.15 && maxRes <= 8 && (!hasOfficer || officerTransformed) {
		if l.categoryTotals["resource"] > l.categoryTotals["peer"] {
			subtype = "follow_resource"
		} else {
			subtype = "follow_peer"
		}
		return "follow_strong", "following", subtype, hasOfficer && officerTransformed
	}
	if ratio <= r.FollowWeakThreshold && (root == "weak" || root == "none") && monthScore < 0 && l.support <= total*.15 && maxSup <= 8 && (!hasSupport || supportTransformed) {
		pairs := []struct {
			name string
			v    float64
		}{{"follow_wealth", l.categoryTotals["wealth"]}, {"follow_officer", l.categoryTotals["officer"]}, {"follow_output", l.categoryTotals["output"]}}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].v > pairs[j].v })
		subtype = pairs[0].name
		if len(pairs) > 1 && math.Abs(pairs[0].v-pairs[1].v) < 1 {
			subtype = "follow_momentum"
		}
		return "follow_weak", "following", subtype, (hasSupport && supportTransformed) || (root == "weak" && weakRootWasClashed(in, r, l))
	}
	if ratio >= r.StrongThreshold {
		return "strong", "normal", "", false
	}
	if ratio <= r.WeakThreshold {
		return "weak", "normal", "", false
	}
	return "balanced", "normal", "", false
}

func allVisibleMatchedByTransformation(in Input, l *ledger, match func(string) bool) bool {
	required := map[string]bool{}
	for pos, p := range map[string]Pillar{"year_stem": in.Year, "month_stem": in.Month, "hour_stem": in.Hour} {
		if match(tenGod(in.Day.Stem, p.Stem)) {
			required[pos] = true
		}
	}
	if len(required) == 0 {
		return false
	}
	for _, rel := range l.relations {
		if rel.Type != "stem_combination" || !rel.Transformed {
			continue
		}
		for _, pos := range rel.Positions {
			delete(required, pos)
		}
	}
	return len(required) == 0
}

func weakRootWasClashed(in Input, r RuleSet, l *ledger) bool {
	dayElement := stemElement[in.Day.Stem]
	weakPositions := map[string]bool{}
	for pos, p := range map[string]Pillar{"year_branch": in.Year, "month_branch": in.Month, "day_branch": in.Day, "hour_branch": in.Hour} {
		for _, h := range r.HiddenStems[p.Branch] {
			if h.Weight == .1 && stemElement[h.Stem] == dayElement {
				weakPositions[pos] = true
			}
		}
	}
	for _, rel := range l.relations {
		if rel.Type == "clash" {
			for _, pos := range rel.Positions {
				if weakPositions[pos] {
					return true
				}
			}
		}
	}
	return false
}
