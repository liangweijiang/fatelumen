package displaydict

import "strings"

const Version = "bazi-display-dictionary-v2"

type Names struct {
	ZH string `json:"zh"`
	EN string `json:"en"`
	JA string `json:"ja"`
	KO string `json:"ko"`
}

type Term struct {
	Code     string       `json:"code"`
	Category string       `json:"category"`
	Names    Names        `json:"names"`
	Review   ReviewStatus `json:"review"`
}

type ReviewStatus struct {
	ZH string `json:"zh"`
	EN string `json:"en"`
	JA string `json:"ja"`
	KO string `json:"ko"`
}
type Coverage struct {
	Locale   string  `json:"locale"`
	Total    int     `json:"total"`
	Approved int     `json:"approved"`
	Draft    int     `json:"draft"`
	Missing  int     `json:"missing"`
	Rate     float64 `json:"rate"`
	Ready    bool    `json:"ready"`
}

type GlossaryEntry struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

var jaNames = map[string]string{
	"wood": "木", "fire": "火", "earth": "土", "metal": "金", "water": "水", "day_master": "日主",
	"bi_jian": "比肩", "jie_cai": "劫財", "shi_shen": "食神", "shang_guan": "傷官", "zheng_cai": "正財", "pian_cai": "偏財", "zheng_guan": "正官", "qi_sha": "偏官（七殺）", "zheng_yin": "印綬", "pian_yin": "偏印",
	"extremely_strong": "極旺", "strong": "身旺", "slightly_strong": "やや身旺", "balanced": "中和", "slightly_weak": "やや身弱", "weak": "身弱", "extremely_weak": "極弱",
	"normal": "普通格", "false_following": "仮従格", "following": "従格候補", "dominant": "専旺格候補", "transformation": "化気格候補",
	"candidate": "候補", "confirmed": "確定", "matched": "該当", "rejected": "非該当", "pending": "判定待ち", "missing": "欠落", "present": "あり", "rooted": "通根", "visible_and_rooted": "透干かつ通根",
	"resource": "印星", "peer": "比劫", "output": "食傷", "wealth": "財星", "officer": "官殺", "supporting": "扶助", "secondary": "副次", "moderate": "中程度", "relatively_strong": "偏旺", "relatively_weak": "偏弱",
	"combination": "干合", "stem_combination": "干合", "branch_combination": "支合", "six_combination": "六合", "clash": "冲", "punishment": "刑", "harm": "害", "break": "破", "three_harmony": "三合", "three_meeting": "方合", "half_harmony": "半会", "long_life": "十二運",
	"primary": "第一候補", "secondary_useful": "第二用神", "favorable": "喜神", "taboo": "忌神", "enemy": "仇神", "neutral": "中立",
	"severe_cold": "極寒", "cold": "寒性", "balanced_temperature": "寒熱中和", "hot": "熱性", "severe_hot": "極熱", "severe_dry": "極燥", "dry": "燥性", "balanced_moisture": "燥湿中和", "wet": "湿性", "severe_wet": "極湿",
	"element.power.wood": "木の力量", "element.power.fire": "火の力量", "element.power.earth": "土の力量", "element.power.metal": "金の力量", "element.power.water": "水の力量",
	"pattern.following": "従格候補", "pattern.dominant": "専旺格候補", "pattern.normal": "普通格", "ten_god.structure": "通変星構成",
	"ten_god.peer.same": "同性の比肩", "ten_god.peer.opposite": "異性の劫財", "ten_god.output.same": "同性の食神", "ten_god.output.opposite": "異性の傷官", "ten_god.resource.same": "同性の偏印", "ten_god.resource.opposite": "異性の印綬", "ten_god.officer.same": "同性の偏官", "ten_god.officer.opposite": "異性の正官", "ten_god.wealth.same": "同性の偏財", "ten_god.wealth.opposite": "異性の正財",
	"ten_god.category.peer": "比劫合計", "ten_god.category.output": "食傷合計", "ten_god.category.resource": "印星合計", "ten_god.category.officer": "官殺合計", "ten_god.category.wealth": "財星合計",
}

var koNames = map[string]string{
	"wood": "목", "fire": "화", "earth": "토", "metal": "금", "water": "수", "day_master": "일간",
	"bi_jian": "비견", "jie_cai": "겁재", "shi_shen": "식신", "shang_guan": "상관", "zheng_cai": "정재", "pian_cai": "편재", "zheng_guan": "정관", "qi_sha": "편관(칠살)", "zheng_yin": "정인", "pian_yin": "편인",
	"extremely_strong": "극신강", "strong": "신강", "slightly_strong": "약간 신강", "balanced": "중화", "slightly_weak": "약간 신약", "weak": "신약", "extremely_weak": "극신약",
	"normal": "일반격", "false_following": "가종격", "following": "종격 후보", "dominant": "전왕격 후보", "transformation": "화기격 후보",
	"candidate": "후보", "confirmed": "확정", "matched": "해당", "rejected": "비해당", "pending": "판정 대기", "missing": "누락", "present": "존재", "rooted": "통근", "visible_and_rooted": "투간 및 통근",
	"resource": "인성", "peer": "비겁", "output": "식상", "wealth": "재성", "officer": "관성", "supporting": "생조", "secondary": "보조", "moderate": "중간", "relatively_strong": "비교적 강함", "relatively_weak": "비교적 약함",
	"combination": "천간합", "stem_combination": "천간합", "branch_combination": "지지육합", "six_combination": "육합", "clash": "충", "punishment": "형", "harm": "해", "break": "파", "three_harmony": "삼합", "three_meeting": "방합", "half_harmony": "반합", "long_life": "십이운성",
	"primary": "제1후보", "secondary_useful": "제2용신", "favorable": "희신", "taboo": "기신", "enemy": "구신", "neutral": "중립",
	"severe_cold": "극한", "cold": "한", "balanced_temperature": "한열 중화", "hot": "열", "severe_hot": "극열", "severe_dry": "극조", "dry": "조", "balanced_moisture": "조습 중화", "wet": "습", "severe_wet": "극습",
	"element.power.wood": "목의 세력", "element.power.fire": "화의 세력", "element.power.earth": "토의 세력", "element.power.metal": "금의 세력", "element.power.water": "수의 세력",
	"pattern.following": "종격 후보", "pattern.dominant": "전왕격 후보", "pattern.normal": "일반격", "ten_god.structure": "십신 구조",
	"ten_god.peer.same": "동음양 비견", "ten_god.peer.opposite": "이음양 겁재", "ten_god.output.same": "동음양 식신", "ten_god.output.opposite": "이음양 상관", "ten_god.resource.same": "동음양 편인", "ten_god.resource.opposite": "이음양 정인", "ten_god.officer.same": "동음양 편관", "ten_god.officer.opposite": "이음양 정관", "ten_god.wealth.same": "동음양 편재", "ten_god.wealth.opposite": "이음양 정재",
	"ten_god.category.peer": "비겁 합계", "ten_god.category.output": "식상 합계", "ten_god.category.resource": "인성 합계", "ten_god.category.officer": "관성 합계", "ten_god.category.wealth": "재성 합계",
}

var jaNamesByCategory = map[string]string{"power_state\x00dominant": "主導"}
var koNamesByCategory = map[string]string{"power_state\x00dominant": "주도"}

var terms = []Term{
	term("wood", "element", "木", "Wood"), term("fire", "element", "火", "Fire"), term("earth", "element", "土", "Earth"), term("metal", "element", "金", "Metal"), term("water", "element", "水", "Water"),
	term("day_master", "bazi_term", "日主", "Day Master"), term("bi_jian", "ten_god", "比肩", "Companion"), term("jie_cai", "ten_god", "劫财", "Rob Wealth"), term("shi_shen", "ten_god", "食神", "Eating God"), term("shang_guan", "ten_god", "伤官", "Hurting Officer"), term("zheng_cai", "ten_god", "正财", "Direct Wealth"), term("pian_cai", "ten_god", "偏财", "Indirect Wealth"), term("zheng_guan", "ten_god", "正官", "Direct Officer"), term("qi_sha", "ten_god", "七杀", "Seven Killings"), term("zheng_yin", "ten_god", "正印", "Direct Resource"), term("pian_yin", "ten_god", "偏印", "Indirect Resource"),
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
	ja, ko := jaNames[code], koNames[code]
	if value := jaNamesByCategory[category+"\x00"+code]; value != "" {
		ja = value
	}
	if value := koNamesByCategory[category+"\x00"+code]; value != "" {
		ko = value
	}
	jaStatus, koStatus := "missing", "missing"
	if ja != "" {
		jaStatus = "draft"
	}
	if ko != "" {
		koStatus = "draft"
	}
	return Term{Code: code, Category: category, Names: Names{ZH: zh, EN: en, JA: ja, KO: ko}, Review: ReviewStatus{ZH: "approved", EN: "draft", JA: jaStatus, KO: koStatus}}
}

func Terms() []Term { return append([]Term(nil), terms...) }

// GlossaryForText returns only reviewed/catalogued translations actually used
// by one chapter. Missing translations are intentionally omitted so the model
// can translate them contextually without blocking the call.
func GlossaryForText(text, locale string) []GlossaryEntry {
	if strings.ToLower(strings.TrimSpace(locale)) == "zh" {
		return nil
	}
	seen := map[string]bool{}
	out := []GlossaryEntry{}
	for _, item := range terms {
		source := strings.TrimSpace(item.Names.ZH)
		target := strings.TrimSpace(nameForLocale(item.Names, locale))
		if source == "" || target == "" || !strings.Contains(text, source) {
			continue
		}
		key := source + "\x00" + target
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, GlossaryEntry{Source: source, Target: target})
	}
	return out
}

func nameForLocale(names Names, locale string) string {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "en":
		return names.EN
	case "ja":
		return names.JA
	case "ko":
		return names.KO
	default:
		return names.ZH
	}
}

func LocaleCoverage(locale string) Coverage {
	locale = strings.ToLower(strings.TrimSpace(locale))
	coverage := Coverage{Locale: locale, Total: len(terms)}
	for _, item := range terms {
		switch reviewForLocale(item.Review, locale) {
		case "approved":
			coverage.Approved++
		case "draft":
			coverage.Draft++
		default:
			coverage.Missing++
		}
	}
	if coverage.Total > 0 {
		coverage.Rate = float64(coverage.Approved) / float64(coverage.Total)
	}
	coverage.Ready = coverage.Approved == coverage.Total
	return coverage
}

func reviewForLocale(review ReviewStatus, locale string) string {
	switch locale {
	case "en":
		return review.EN
	case "ja":
		return review.JA
	case "ko":
		return review.KO
	default:
		return review.ZH
	}
}

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
