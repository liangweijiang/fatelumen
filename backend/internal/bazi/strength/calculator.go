package strength

import (
	"fmt"
	"math"
)

type ledger struct {
	support, restraint float64
	contributions      []Contribution
	relations          []Relation
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
	l := &ledger{}
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
	return Result{RuleVersion: rules.Version, DayElement: dayElement, MonthScore: monthScore,
		SupportScore: round(l.support), RestraintScore: round(l.restraint), SupportRatio: math.Round(ratio*10000) / 10000,
		RootLevel:     root,
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
