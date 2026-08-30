package strength

import (
	"fatelumen/backend/internal/bazi/basedata"
	"fmt"
)

type positionedBranch struct{ pos, symbol string }

func applyRelations(in Input, r RuleSet, monthElement, dayElement string, l *ledger) {
	base := basedata.V1()
	stems := []struct{ pos, s string }{{"year_stem", in.Year.Stem}, {"month_stem", in.Month.Stem}, {"day_stem", in.Day.Stem}, {"hour_stem", in.Hour.Stem}}
	stemCombos := relationsBy(base, "stem", "combination")
	usedStem := map[int]bool{}
	for _, c := range stemCombos {
		for i := 0; i < len(stems); i++ {
			for j := i + 1; j < len(stems); j++ {
				if usedStem[i] || usedStem[j] || !pair(stems[i].s, stems[j].s, c.Members[0], c.Members[1]) {
					continue
				}
				usedStem[i], usedStem[j] = true, true
				if monthSupports(monthElement, c.Element) {
					l.relation("stem_combination", "stem_combination", c.Element, "month command supports transformation", []string{stems[i].pos, stems[j].pos}, []string{stems[i].s, stems[j].s}, c.Score, true, dayElement)
				} else {
					for _, x := range []struct{ pos, s string }{stems[i], stems[j]} {
						if x.pos == "day_stem" {
							continue
						}
						cat := category(tenGod(in.Day.Stem, x.s))
						l.adjust(cat, -5)
					}
					l.relations = append(l.relations, Relation{Code: "stem_combination", Type: "stem_combination", Positions: []string{stems[i].pos, stems[j].pos}, Symbols: []string{stems[i].s, stems[j].s}, Element: c.Element, Score: -c.Score, Reason: "combination without transformation"})
				}
			}
		}
	}
	b := []positionedBranch{{"year_branch", in.Year.Branch}, {"month_branch", in.Month.Branch}, {"day_branch", in.Day.Branch}, {"hour_branch", in.Hour.Branch}}
	sixCombinations := relationsBy(base, "branch", "six_combination")
	used := map[int]bool{}
	for _, c := range sixCombinations {
		for i := 0; i < len(b); i++ {
			for j := i + 1; j < len(b); j++ {
				if used[i] || used[j] || !pair(b[i].symbol, b[j].symbol, c.Members[0], c.Members[1]) {
					continue
				}
				used[i], used[j] = true, true
				if monthSupports(monthElement, c.Element) {
					l.relation("branch_six_combination", "branch_combination", c.Element, "month command supports transformation", []string{b[i].pos, b[j].pos}, []string{b[i].symbol, b[j].symbol}, c.Score, true, dayElement)
				} else {
					deductBranch(b[i], 4, r, in.Day.Stem, l)
					deductBranch(b[j], 4, r, in.Day.Stem, l)
					l.relations = append(l.relations, Relation{Code: "branch_six_combination", Type: "branch_combination", Positions: []string{b[i].pos, b[j].pos}, Symbols: []string{b[i].symbol, b[j].symbol}, Element: c.Element, Score: -c.Score, Reason: "combination without transformation"})
				}
			}
		}
	}
	groups := append(relationsBy(base, "branch", "three_harmony"), relationsBy(base, "branch", "three_meeting")...)
	for _, g := range groups {
		found := findDistinct(b, g.Members)
		if len(found) == 3 {
			l.relation("branch_"+g.Type, g.Type, g.Element, "complete group", positions(found), symbols(found), g.Score, true, dayElement)
		} else if len(found) == 2 && g.Type == "three_harmony" && monthSupports(monthElement, g.Element) {
			l.relation("branch_three_harmony_half", g.Type, g.Element, "half group supported by month command", positions(found), symbols(found), g.Score/2, true, dayElement)
		}
	}
	applyPairs("branch_clash", "clash", relationPairs(base, "branch", "clash"), b, func(x positionedBranch) float64 {
		if x.pos == "month_branch" {
			return 4
		}
		return 8
	}, r, in.Day.Stem, l)
	for _, punishment := range relationsBy(base, "branch", "punishment") {
		if len(punishment.Members) == 3 {
			applyTriplePunishment(punishment.Members, b, r, in.Day.Stem, l)
		} else if len(punishment.Members) == 2 && punishment.Members[0] != punishment.Members[1] {
			applyPairs("branch_punishment", "punishment", [][2]string{{punishment.Members[0], punishment.Members[1]}}, b, func(positionedBranch) float64 { return 4 }, r, in.Day.Stem, l)
		}
	}
	for _, punishment := range relationsBy(base, "branch", "punishment") {
		if len(punishment.Members) != 2 || punishment.Members[0] != punishment.Members[1] {
			continue
		}
		z := punishment.Members[0]
		idx := []positionedBranch{}
		for _, x := range b {
			if x.symbol == z {
				idx = append(idx, x)
			}
		}
		if len(idx) >= 2 {
			deductBranch(idx[0], 4, r, in.Day.Stem, l)
			l.relations = append(l.relations, Relation{Code: "branch_self_punishment", Type: "punishment", Positions: positions(idx[:2]), Symbols: symbols(idx[:2]), Score: -4, Reason: "self punishment"})
		}
	}
	applyPairs("branch_harm", "harm", relationPairs(base, "branch", "harm"), b, func(positionedBranch) float64 { return 3 }, r, in.Day.Stem, l)
}

func relationsBy(c basedata.Catalog, scope, typ string) []basedata.Relation {
	out := []basedata.Relation{}
	for _, v := range c.Relations {
		if v.Scope == scope && v.Type == typ {
			out = append(out, v)
		}
	}
	return out
}
func relationPairs(c basedata.Catalog, scope, typ string) [][2]string {
	out := [][2]string{}
	for _, v := range relationsBy(c, scope, typ) {
		if len(v.Members) == 2 {
			out = append(out, [2]string{v.Members[0], v.Members[1]})
		}
	}
	return out
}

func deductBranch(x positionedBranch, points float64, r RuleSet, day string, l *ledger) {
	for _, h := range r.HiddenStems[x.symbol] {
		cat := category(tenGod(day, h.Stem))
		v := points * h.Weight
		l.adjust(cat, -v)
	}
}
func applyPairs(code, typ string, pairs [][2]string, b []positionedBranch, points func(positionedBranch) float64, r RuleSet, day string, l *ledger) {
	for _, p := range pairs {
		for i := 0; i < len(b); i++ {
			for j := i + 1; j < len(b); j++ {
				if pair(b[i].symbol, b[j].symbol, p[0], p[1]) {
					a, c := points(b[i]), points(b[j])
					deductBranch(b[i], a, r, day, l)
					deductBranch(b[j], c, r, day, l)
					l.relations = append(l.relations, Relation{Code: code, Type: typ, Positions: []string{b[i].pos, b[j].pos}, Symbols: []string{b[i].symbol, b[j].symbol}, Score: -(a + c), Reason: fmt.Sprintf("%s adjustment", typ)})
				}
			}
		}
	}
}
func applyTriplePunishment(m []string, b []positionedBranch, r RuleSet, day string, l *ledger) {
	f := findDistinct(b, m)
	if len(f) < 2 {
		return
	}
	v := 3.0
	if len(f) == 3 {
		v = 6
	}
	for _, x := range f {
		deductBranch(x, v, r, day, l)
	}
	l.relations = append(l.relations, Relation{Code: "branch_three_punishment", Type: "punishment", Positions: positions(f), Symbols: symbols(f), Score: -v * float64(len(f)), Reason: "three punishment adjustment"})
}
func pair(a, b, x, y string) bool { return a == x && b == y || a == y && b == x }
func findDistinct(all []positionedBranch, members []string) []positionedBranch {
	out := []positionedBranch{}
	used := map[int]bool{}
	for _, m := range members {
		for i, x := range all {
			if !used[i] && x.symbol == m {
				used[i] = true
				out = append(out, x)
				break
			}
		}
	}
	return out
}
func positions(v []positionedBranch) []string {
	o := make([]string, len(v))
	for i, x := range v {
		o[i] = x.pos
	}
	return o
}
func symbols(v []positionedBranch) []string {
	o := make([]string, len(v))
	for i, x := range v {
		o[i] = x.symbol
	}
	return o
}
