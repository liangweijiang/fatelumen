package displaydict

import "strings"

const Version = "bazi-display-dictionary-v1"

type Names struct {
	ZH string `json:"zh"`
	EN string `json:"en"`
	JA string `json:"ja"`
	KO string `json:"ko"`
}

type Term struct {
	Code     string `json:"code"`
	Category string `json:"category"`
	Names    Names  `json:"names"`
}

var terms = []Term{
	term("wood", "element", "木", "Wood"), term("fire", "element", "火", "Fire"), term("earth", "element", "土", "Earth"), term("metal", "element", "金", "Metal"), term("water", "element", "水", "Water"),
	term("extremely_strong", "strength", "极强", "Extremely strong"), term("strong", "strength", "身强", "Strong"), term("slightly_strong", "strength", "身偏强", "Slightly strong"), term("balanced", "strength", "中和", "Balanced"), term("slightly_weak", "strength", "身偏弱", "Slightly weak"), term("weak", "strength", "身弱", "Weak"), term("extremely_weak", "strength", "极弱", "Extremely weak"),
	term("normal", "pattern", "普通格", "Normal pattern"), term("false_following", "pattern", "假从格", "False-following pattern"), term("following", "pattern", "从格候选", "Following candidate"), term("dominant", "pattern", "专旺格候选", "Dominant candidate"), term("transformation", "pattern", "化气格候选", "Transformation candidate"), term("candidate", "state", "候选", "Candidate"), term("confirmed", "state", "已确认", "Confirmed"), term("matched", "state", "命中", "Matched"), term("rejected", "state", "未命中", "Rejected"), term("pending", "state", "待判定", "Pending"), term("missing", "state", "缺失", "Missing"), term("present", "state", "已存在", "Present"), term("rooted", "state", "有根", "Rooted"), term("visible_and_rooted", "state", "透干且有根", "Visible and rooted"),
	term("resource", "ten_god_category", "印星", "Resource"), term("peer", "ten_god_category", "比劫", "Peer"), term("output", "ten_god_category", "食伤", "Output"), term("wealth", "ten_god_category", "财星", "Wealth"), term("officer", "ten_god_category", "官杀", "Officer"), term("supporting", "power_state", "扶助", "Supporting"), term("secondary", "power_state", "次要", "Secondary"), term("moderate", "power_state", "中等", "Moderate"), term("dominant", "power_state", "主导", "Dominant"), term("relatively_strong", "power_state", "偏强", "Relatively strong"), term("relatively_weak", "power_state", "偏弱", "Relatively weak"),
	term("combination", "relation", "天干五合", "Stem combination"), term("stem_combination", "relation", "天干五合", "Stem combination"), term("branch_combination", "relation", "地支六合", "Branch combination"), term("six_combination", "relation", "地支六合", "Six combination"), term("clash", "relation", "相冲", "Clash"), term("punishment", "relation", "相刑", "Punishment"), term("harm", "relation", "相害", "Harm"), term("break", "relation", "相破", "Break"), term("three_harmony", "relation", "三合", "Three harmony"), term("three_meeting", "relation", "三会", "Three meeting"), term("half_harmony", "relation", "半合", "Half harmony"), term("long_life", "relation", "十二长生", "Twelve life stages"),
	term("primary", "useful_god", "首选", "Primary"), term("secondary_useful", "useful_god", "次选用神", "Secondary useful element"), term("favorable", "useful_god", "喜神", "Favorable"), term("taboo", "useful_god", "忌神", "Taboo"), term("enemy", "useful_god", "仇神", "Enemy"), term("neutral", "useful_god", "中性", "Neutral"),
	term("severe_cold", "climate", "极寒", "Severely cold"), term("cold", "climate", "偏寒", "Cold"), term("balanced_temperature", "climate", "寒热适中", "Balanced temperature"), term("hot", "climate", "偏热", "Hot"), term("severe_hot", "climate", "极热", "Severely hot"), term("severe_dry", "climate", "极燥", "Severely dry"), term("dry", "climate", "偏燥", "Dry"), term("balanced_moisture", "climate", "燥湿适中", "Balanced moisture"), term("wet", "climate", "偏湿", "Wet"), term("severe_wet", "climate", "极湿", "Severely wet"),
	term("element.power.wood", "fact", "木力量", "Wood power"), term("element.power.fire", "fact", "火力量", "Fire power"), term("element.power.earth", "fact", "土力量", "Earth power"), term("element.power.metal", "fact", "金力量", "Metal power"), term("element.power.water", "fact", "水力量", "Water power"),
	term("pattern.following", "fact", "从格候选", "Following candidate"), term("pattern.dominant", "fact", "专旺格候选", "Dominant candidate"), term("pattern.normal", "fact", "普通格", "Normal pattern"),
	term("ten_god.structure", "fact", "十神结构", "Ten-god structure"), term("ten_god.peer.same", "fact", "同阴阳比肩", "Same-polarity peer"), term("ten_god.peer.opposite", "fact", "异阴阳劫财", "Opposite-polarity peer"), term("ten_god.output.same", "fact", "同阴阳食神", "Same-polarity output"), term("ten_god.output.opposite", "fact", "异阴阳伤官", "Opposite-polarity output"), term("ten_god.resource.same", "fact", "同阴阳偏印", "Same-polarity resource"), term("ten_god.resource.opposite", "fact", "异阴阳正印", "Opposite-polarity resource"), term("ten_god.officer.same", "fact", "同阴阳七杀", "Same-polarity officer"), term("ten_god.officer.opposite", "fact", "异阴阳正官", "Opposite-polarity officer"), term("ten_god.wealth.same", "fact", "同阴阳偏财", "Same-polarity wealth"), term("ten_god.wealth.opposite", "fact", "异阴阳正财", "Opposite-polarity wealth"),
	term("ten_god.category.peer", "fact", "比劫合计", "Peer category"), term("ten_god.category.output", "fact", "食伤合计", "Output category"), term("ten_god.category.resource", "fact", "印星合计", "Resource category"), term("ten_god.category.officer", "fact", "官杀合计", "Officer category"), term("ten_god.category.wealth", "fact", "财星合计", "Wealth category"),
}

func term(code, category, zh, en string) Term {
	return Term{Code: code, Category: category, Names: Names{ZH: zh, EN: en}}
}

func Terms() []Term { return append([]Term(nil), terms...) }

func Translate(code, locale string) string {
	return TranslateIn("", code, locale)
}

func TranslateIn(category, code, locale string) string {
	value := strings.TrimSpace(code)
	if value == "" {
		return "—"
	}
	for _, item := range terms {
		if strings.EqualFold(item.Code, value) && (category == "" || item.Category == category) {
			switch strings.ToLower(locale) {
			case "en":
				if item.Names.EN != "" {
					return item.Names.EN
				}
			case "ja":
				if item.Names.JA != "" {
					return item.Names.JA
				}
			case "ko":
				if item.Names.KO != "" {
					return item.Names.KO
				}
			default:
				if item.Names.ZH != "" {
					return item.Names.ZH
				}
			}
		}
	}
	if category != "" {
		return TranslateIn("", value, locale)
	}
	return value
}

func ZH(code string) string             { return Translate(code, "zh") }
func ZHIn(category, code string) string { return TranslateIn(category, code, "zh") }
