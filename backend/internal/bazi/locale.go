package bazi

type Locale string

const (
	LocaleZH Locale = "zh"
	LocaleEN Locale = "en"
	LocaleJA Locale = "ja"
	LocaleKO Locale = "ko"
)

var stemLocale = map[Locale][]string{
	LocaleZH: {"", "甲", "乙", "丙", "丁", "戊", "己", "庚", "辛", "壬", "癸"},
	LocaleEN: {"", "Jia", "Yi", "Bing", "Ding", "Wu", "Ji", "Geng", "Xin", "Ren", "Gui"},
	LocaleJA: {"", "甲", "乙", "丙", "丁", "戊", "己", "庚", "辛", "壬", "癸"},
	LocaleKO: {"", "갑", "을", "병", "정", "무", "기", "경", "신", "임", "계"},
}

var branchLocale = map[Locale][]string{
	LocaleZH: {"", "子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"},
	LocaleEN: {"", "Zi", "Chou", "Yin", "Mao", "Chen", "Si", "Wu", "Wei", "Shen", "You", "Xu", "Hai"},
	LocaleJA: {"", "子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"},
	LocaleKO: {"", "자", "축", "인", "묘", "진", "사", "오", "미", "신", "유", "술", "해"},
}

var elementLocale = map[Locale]map[string]string{
	LocaleZH: {"木": "木", "火": "火", "土": "土", "金": "金", "水": "水"},
	LocaleEN: {"木": "Wood", "火": "Fire", "土": "Earth", "金": "Metal", "水": "Water"},
	LocaleJA: {"木": "木", "火": "火", "土": "土", "金": "金", "水": "水"},
	LocaleKO: {"木": "목", "火": "화", "土": "토", "金": "금", "水": "수"},
}

var tenGodLocale = map[Locale]map[string]string{
	LocaleZH: {
		"比肩": "比肩", "劫财": "劫财", "食神": "食神", "伤官": "伤官", "偏财": "偏财",
		"正财": "正财", "七杀": "七杀", "正官": "正官", "偏印": "偏印", "正印": "正印",
	},
	LocaleEN: {
		"比肩": "Friend", "劫财": "Rob Wealth", "食神": "Eating God", "伤官": "Hurting Officer", "偏财": "Indirect Wealth",
		"正财": "Direct Wealth", "七杀": "Killing", "正官": "Officer", "偏印": "Indirect Resource", "正印": "Direct Resource",
	},
	LocaleJA: {
		"比肩": "比肩", "劫财": "劫財", "食神": "食神", "伤官": "傷官", "偏财": "偏財",
		"正财": "正財", "七杀": "七殺", "正官": "正官", "偏印": "偏印", "正印": "正印",
	},
	LocaleKO: {
		"比肩": "비견", "劫财": "겁재", "食神": "식신", "伤官": "상관", "偏财": "편재",
		"正财": "정재", "七杀": "칠살", "正官": "정관", "偏印": "편인", "正印": "정인",
	},
}

func TranslateStem(loc Locale, idx int) string {
	if m, ok := stemLocale[loc]; ok && idx >= 0 && idx < len(m) {
		return m[idx]
	}
	return ""
}

func TranslateBranch(loc Locale, idx int) string {
	if m, ok := branchLocale[loc]; ok && idx >= 0 && idx < len(m) {
		return m[idx]
	}
	return ""
}

func TranslateElement(loc Locale, e string) string {
	if m, ok := elementLocale[loc]; ok {
		if v, ok2 := m[e]; ok2 {
			return v
		}
	}
	return e
}

func TranslateTenGod(loc Locale, tg string) string {
	if m, ok := tenGodLocale[loc]; ok {
		if v, ok2 := m[tg]; ok2 {
			return v
		}
	}
	return tg
}
