package basedata

func n(zh, en, ja, ko string) Names { return Names{zh, en, ja, ko} }

func relationName(typ string) Names {
	switch typ {
	case "combination":
		return n("天干五合", "Stem combination", "天干五合", "천간합")
	case "six_combination":
		return n("地支六合", "Six combination", "地支六合", "지지육합")
	case "three_harmony":
		return n("地支三合", "Three harmony", "地支三合", "지지삼합")
	case "half_harmony":
		return n("地支半合", "Half harmony", "地支半合", "지지반합")
	case "three_meeting":
		return n("地支三会", "Three meeting", "地支三会", "지지방합")
	case "clash":
		return n("地支相冲", "Branch clash", "地支相冲", "지지충")
	case "punishment":
		return n("地支相刑", "Branch punishment", "地支相刑", "지지형")
	case "harm":
		return n("地支相害", "Branch harm", "地支相害", "지지해")
	case "break":
		return n("地支相破", "Branch break", "地支相破", "지지파")
	}
	return n(typ, typ, typ, typ)
}

func catalogV1() Catalog {
	elements := []Element{{"木", n("木", "Wood", "木", "木"), "火", "土", "水", "金"}, {"火", n("火", "Fire", "火", "火"), "土", "金", "木", "水"}, {"土", n("土", "Earth", "土", "土"), "金", "水", "火", "木"}, {"金", n("金", "Metal", "金", "金"), "水", "木", "土", "火"}, {"水", n("水", "Water", "水", "水"), "木", "火", "金", "土"}}
	stems := []Stem{{"甲", "木", "yang", 1, n("甲", "Jia", "甲", "갑")}, {"乙", "木", "yin", 2, n("乙", "Yi", "乙", "을")}, {"丙", "火", "yang", 3, n("丙", "Bing", "丙", "병")}, {"丁", "火", "yin", 4, n("丁", "Ding", "丁", "정")}, {"戊", "土", "yang", 5, n("戊", "Wu", "戊", "무")}, {"己", "土", "yin", 6, n("己", "Ji", "己", "기")}, {"庚", "金", "yang", 7, n("庚", "Geng", "庚", "경")}, {"辛", "金", "yin", 8, n("辛", "Xin", "辛", "신")}, {"壬", "水", "yang", 9, n("壬", "Ren", "壬", "임")}, {"癸", "水", "yin", 10, n("癸", "Gui", "癸", "계")}}
	branches := []Branch{
		{"子", "水", "yang", 1, n("子", "Zi", "子", "자"), []HiddenStem{{"癸", 1, "main"}}}, {"丑", "土", "yin", 2, n("丑", "Chou", "丑", "축"), []HiddenStem{{"己", .6, "main"}, {"癸", .3, "middle"}, {"辛", .1, "residual"}}}, {"寅", "木", "yang", 3, n("寅", "Yin", "寅", "인"), []HiddenStem{{"甲", .6, "main"}, {"丙", .3, "middle"}, {"戊", .1, "residual"}}}, {"卯", "木", "yin", 4, n("卯", "Mao", "卯", "묘"), []HiddenStem{{"乙", 1, "main"}}}, {"辰", "土", "yang", 5, n("辰", "Chen", "辰", "진"), []HiddenStem{{"戊", .6, "main"}, {"乙", .3, "middle"}, {"癸", .1, "residual"}}}, {"巳", "火", "yin", 6, n("巳", "Si", "巳", "사"), []HiddenStem{{"丙", .6, "main"}, {"戊", .3, "middle"}, {"庚", .1, "residual"}}}, {"午", "火", "yang", 7, n("午", "Wu", "午", "오"), []HiddenStem{{"丁", .7, "main"}, {"己", .3, "middle"}}}, {"未", "土", "yin", 8, n("未", "Wei", "未", "미"), []HiddenStem{{"己", .6, "main"}, {"丁", .3, "middle"}, {"乙", .1, "residual"}}}, {"申", "金", "yang", 9, n("申", "Shen", "申", "신"), []HiddenStem{{"庚", .6, "main"}, {"壬", .3, "middle"}, {"戊", .1, "residual"}}}, {"酉", "金", "yin", 10, n("酉", "You", "酉", "유"), []HiddenStem{{"辛", 1, "main"}}}, {"戌", "土", "yang", 11, n("戌", "Xu", "戌", "술"), []HiddenStem{{"戊", .6, "main"}, {"辛", .3, "middle"}, {"丁", .1, "residual"}}}, {"亥", "水", "yin", 12, n("亥", "Hai", "亥", "해"), []HiddenStem{{"壬", .7, "main"}, {"甲", .3, "middle"}}},
	}
	relations := []Relation{}
	add := func(scope, typ, element string, score float64, pairs ...[]string) {
		for _, m := range pairs {
			code := scope + "." + typ + "."
			for _, x := range m {
				code += x
			}
			relations = append(relations, Relation{code, scope, typ, element, m, score, relationName(typ)})
		}
	}
	add("stem", "combination", "土", 10, []string{"甲", "己"})
	add("stem", "combination", "金", 10, []string{"乙", "庚"})
	add("stem", "combination", "水", 10, []string{"丙", "辛"})
	add("stem", "combination", "木", 10, []string{"丁", "壬"})
	add("stem", "combination", "火", 10, []string{"戊", "癸"})
	for _, x := range []struct {
		m []string
		e string
	}{{[]string{"子", "丑"}, "土"}, {[]string{"寅", "亥"}, "木"}, {[]string{"卯", "戌"}, "火"}, {[]string{"辰", "酉"}, "金"}, {[]string{"巳", "申"}, "水"}, {[]string{"午", "未"}, "土"}} {
		add("branch", "six_combination", x.e, 8, x.m)
	}
	for _, x := range []struct {
		m []string
		e string
	}{{[]string{"申", "子", "辰"}, "水"}, {[]string{"亥", "卯", "未"}, "木"}, {[]string{"寅", "午", "戌"}, "火"}, {[]string{"巳", "酉", "丑"}, "金"}} {
		add("branch", "three_harmony", x.e, 15, x.m)
	}
	for _, x := range []struct {
		m []string
		e string
	}{{[]string{"申", "子"}, "水"}, {[]string{"子", "辰"}, "水"}, {[]string{"申", "辰"}, "水"}, {[]string{"亥", "卯"}, "木"}, {[]string{"卯", "未"}, "木"}, {[]string{"亥", "未"}, "木"}, {[]string{"寅", "午"}, "火"}, {[]string{"午", "戌"}, "火"}, {[]string{"寅", "戌"}, "火"}, {[]string{"巳", "酉"}, "金"}, {[]string{"酉", "丑"}, "金"}, {[]string{"巳", "丑"}, "金"}} {
		add("branch", "half_harmony", x.e, 7.5, x.m)
	}
	for _, x := range []struct {
		m []string
		e string
	}{{[]string{"寅", "卯", "辰"}, "木"}, {[]string{"巳", "午", "未"}, "火"}, {[]string{"申", "酉", "戌"}, "金"}, {[]string{"亥", "子", "丑"}, "水"}} {
		add("branch", "three_meeting", x.e, 20, x.m)
	}
	for _, m := range [][]string{{"子", "午"}, {"丑", "未"}, {"寅", "申"}, {"卯", "酉"}, {"辰", "戌"}, {"巳", "亥"}} {
		add("branch", "clash", "", -8, m)
	}
	add("branch", "punishment", "", -4, []string{"子", "卯"}, []string{"辰", "辰"}, []string{"午", "午"}, []string{"酉", "酉"}, []string{"亥", "亥"})
	add("branch", "punishment", "", -6, []string{"寅", "巳", "申"}, []string{"丑", "戌", "未"})
	for _, m := range [][]string{{"子", "未"}, {"丑", "午"}, {"寅", "巳"}, {"卯", "辰"}, {"申", "亥"}, {"酉", "戌"}} {
		add("branch", "harm", "", -3, m)
	}
	for _, m := range [][]string{{"子", "酉"}, {"卯", "午"}, {"辰", "丑"}, {"戌", "未"}, {"寅", "亥"}, {"巳", "申"}} {
		add("branch", "break", "", 0, m)
	}
	tg := []TenGodRule{{"peer.same", "same", "same", "support", n("比肩", "Friend", "比肩", "비견")}, {"peer.opposite", "same", "opposite", "support", n("劫财", "Rob Wealth", "劫財", "겁재")}, {"resource.same", "generates_me", "same", "support", n("偏印", "Indirect Resource", "偏印", "편인")}, {"resource.opposite", "generates_me", "opposite", "support", n("正印", "Direct Resource", "正印", "정인")}, {"output.same", "i_generate", "same", "restraint", n("食神", "Eating God", "食神", "식신")}, {"output.opposite", "i_generate", "opposite", "restraint", n("伤官", "Hurting Officer", "傷官", "상관")}, {"wealth.same", "i_control", "same", "restraint", n("偏财", "Indirect Wealth", "偏財", "편재")}, {"wealth.opposite", "i_control", "opposite", "restraint", n("正财", "Direct Wealth", "正財", "정재")}, {"officer.same", "controls_me", "same", "restraint", n("七杀", "Seven Killings", "七殺", "편관")}, {"officer.opposite", "controls_me", "opposite", "restraint", n("正官", "Direct Officer", "正官", "정관")}}
	weights := []PositionWeight{{"year_stem", 8, n("年干", "Year stem", "年干", "연간")}, {"month_stem", 15, n("月干", "Month stem", "月干", "월간")}, {"day_stem", 0, n("日干", "Day stem", "日干", "일간")}, {"hour_stem", 12, n("时干", "Hour stem", "時干", "시간")}, {"year_branch", 10, n("年支", "Year branch", "年支", "연지")}, {"month_branch", 30, n("月支", "Month branch", "月支", "월지")}, {"day_branch", 25, n("日支", "Day branch", "日支", "일지")}, {"hour_branch", 18, n("时支", "Hour branch", "時支", "시지")}}
	month := map[string]map[string]float64{"木": {"木": 40, "火": -20, "土": -30, "金": -40, "水": 30}, "火": {"木": 30, "火": 40, "土": -20, "金": -30, "水": -40}, "土": {"木": -40, "火": 30, "土": 40, "金": -20, "水": -30}, "金": {"木": -30, "火": -40, "土": 30, "金": 40, "水": -20}, "水": {"木": -20, "火": -30, "土": -40, "金": 30, "水": 40}}
	thresholds := []Threshold{{"strong", .65, n("身强阈值", "Strong threshold", "身強しきい値", "신강 기준")}, {"weak", .35, n("身弱阈值", "Weak threshold", "身弱しきい値", "신약 기준")}, {"follow_strong", .85, n("从强阈值", "Follow-strong threshold", "従強しきい値", "종강 기준")}, {"follow_weak", .15, n("从弱阈值", "Follow-weak threshold", "従弱しきい値", "종약 기준")}, {"month_main_factor", .3, n("月支本气系数", "Month-main factor", "月支本気係数", "월지 본기 계수")}}
	SortRelations(relations)
	return Catalog{Version: VersionV1, Elements: elements, Stems: stems, Branches: branches, Relations: relations, TenGodRules: tg, PositionWeights: weights, MonthScores: month, Thresholds: thresholds}
}
