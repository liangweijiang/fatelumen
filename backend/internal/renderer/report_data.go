package renderer

import (
	"fmt"
	"strings"

	"fatelumen/backend/internal/model"
)

// ---------- 五行摘要 ----------

type elementSummaryItem struct {
	Element string
	Color   string
	Count   int
	Width   int // percentage for bar chart
}

// ---------- 流年 ----------

type fortuneItem struct {
	Year int
	Note string
}

// ReportChapterSection is one readable module within a frozen chapter body.
// The executor stores modules as "title\ncontent" blocks separated by blank
// lines; keeping that boundary here avoids flattening the PDF into one wall of
// text without changing the immutable report payload.
type ReportChapterSection struct {
	Title      string
	Paragraphs []string
}

type ReportChapterPDFData struct {
	No       int
	Key      string
	Title    string
	Sections []ReportChapterSection
	Cycles   []model.CycleNote
	Years    []model.YearNote
}

// ---------- 四柱 ----------

// ---------- 报告 PDF 模板数据 ----------

// ReportPDFData 深度报告 PDF 模板数据。
// 严格遵守 P2：所有文案/标题不出现 "AI"。
type ReportPDFData struct {
	Brand             string
	ProfileName       string
	Locale            string
	ReportTitle       string
	DayMasterLabel    string
	StrengthLevel     string
	ElementBalance    string
	GenDate           string
	SolarDate         string
	LunarDate         string
	Pillars           []PillarDisplay
	Elements          []elementSummaryItem
	Content           model.ReportContent
	FortuneItems      []fortuneItem
	HasTenYearChapter bool
	Suggestions       []string
	Chapters          []ReportChapterPDFData
	SectionLabels     map[string]string
}

var strengthLabels = map[string]map[string]string{
	"zh": {"extremely_strong": "极强", "strong": "身强", "slightly_strong": "身偏强", "balanced": "中和", "slightly_weak": "身偏弱", "weak": "身弱", "extremely_weak": "极弱"},
	"en": {"extremely_strong": "Extremely strong", "strong": "Strong", "slightly_strong": "Slightly strong", "balanced": "Balanced", "slightly_weak": "Slightly weak", "weak": "Weak", "extremely_weak": "Extremely weak"},
	"ja": {"extremely_strong": "極旺", "strong": "身旺", "slightly_strong": "やや身旺", "balanced": "中和", "slightly_weak": "やや身弱", "weak": "身弱", "extremely_weak": "極弱"},
	"ko": {"extremely_strong": "극신강", "strong": "신강", "slightly_strong": "약간 신강", "balanced": "중화", "slightly_weak": "약간 신약", "weak": "신약", "extremely_weak": "극신약"},
}

var reportSectionLabels = map[string]map[string]string{
	"en": {
		"report_title":           "Bazi Deep Reading Report",
		"summary":                "Destiny Overview",
		"personality":            "Personality",
		"career":                 "Career & Wealth",
		"relationship":           "Love & Marriage",
		"health":                 "Health & Wellness",
		"yearly_fortune":         "Yearly Fortune",
		"suggestions":            "Guidance & Suggestions",
		"day_master":             "Day Master",
		"strength":               "Strength",
		"five_elements":          "Five Elements",
		"generated":              "Generated",
		"birth_info":             "Birth Information",
		"page":                   "Page",
		"year":                   "Year",
		"analysis":               "Analysis",
		"contents":               "Contents",
		"chart_overview":         "Four Pillars Overview",
		"private_report":         "Personal Bazi Reading",
		"chapter_chart_detail":   "Refined Chart Reading",
		"chapter_destiny_depth":  "In-Depth Destiny Reading",
		"chapter_ten_gods_full":  "Full Ten-Gods Panorama",
		"chapter_luck_cycle":     "Lifelong Luck-Cycle Trend",
		"chapter_ten_year_years": "Next Ten Years, Year by Year",
		"chapter_career_depth":   "Career Deep Dive",
		"chapter_wealth_depth":   "Wealth Deep Dive",
		"chapter_love_depth":     "Relationship Deep Dive",
		"chapter_health_depth":   "Health Deep Dive",
		"chapter_element_tuning": "Five-Element Climate Balance",
		"chapter_remedies":       "Remedies for Near-Term Challenges",
		"chapter_fortune_guide":  "Five-Element Fortune Guide",
		"chapter_life_plan":      "Lifelong Guidance & Planning",
	},
	"zh": {
		"report_title":           "八字深度解读报告",
		"summary":                "命格总论",
		"personality":            "性格特质",
		"career":                 "事业财运",
		"relationship":           "感情婚姻",
		"health":                 "健康提示",
		"yearly_fortune":         "流年运势",
		"suggestions":            "开运建议",
		"day_master":             "日主",
		"strength":               "强弱",
		"five_elements":          "五行",
		"generated":              "生成时间",
		"birth_info":             "出生信息",
		"page":                   "页",
		"year":                   "流年",
		"analysis":               "解读",
		"contents":               "目录",
		"chart_overview":         "四柱命盘概览",
		"private_report":         "专属八字命理解读",
		"chapter_chart_detail":   "精细排盘",
		"chapter_destiny_depth":  "命格深度解读",
		"chapter_ten_gods_full":  "十神全象分析",
		"chapter_luck_cycle":     "终身大运走势",
		"chapter_ten_year_years": "未来十年逐流年详解",
		"chapter_career_depth":   "事业深度剖析",
		"chapter_wealth_depth":   "财富深度剖析",
		"chapter_love_depth":     "情感深度剖析",
		"chapter_health_depth":   "健康深度剖析",
		"chapter_element_tuning": "五行调候",
		"chapter_remedies":       "近期挑战化解方案",
		"chapter_fortune_guide":  "五行开运指南",
		"chapter_life_plan":      "终身建议与定制规划",
	},
	"ja": {
		"report_title":           "四柱推命 詳細鑑定書",
		"summary":                "命式総論",
		"personality":            "性格の特質",
		"career":                 "仕事と財運",
		"relationship":           "恋愛と結婚",
		"health":                 "健康のヒント",
		"yearly_fortune":         "年運の流れ",
		"suggestions":            "開運のヒント",
		"day_master":             "日主",
		"strength":               "強弱",
		"five_elements":          "五行",
		"generated":              "作成日時",
		"birth_info":             "生年月日情報",
		"page":                   "ページ",
		"year":                   "流年",
		"analysis":               "解説",
		"contents":               "目次",
		"chart_overview":         "四柱命式概要",
		"private_report":         "四柱推命 個人鑑定",
		"chapter_chart_detail":   "詳細命式",
		"chapter_destiny_depth":  "命式の深層解読",
		"chapter_ten_gods_full":  "十神総象分析",
		"chapter_luck_cycle":     "生涯大運の流れ",
		"chapter_ten_year_years": "今後十年・年運詳解",
		"chapter_career_depth":   "仕事の深層分析",
		"chapter_wealth_depth":   "財運の深層分析",
		"chapter_love_depth":     "恋愛の深層分析",
		"chapter_health_depth":   "健康の深層分析",
		"chapter_element_tuning": "五行の調候",
		"chapter_remedies":       "近期の課題への対処法",
		"chapter_fortune_guide":  "五行開運ガイド",
		"chapter_life_plan":      "生涯の指針と設計",
	},
	"ko": {
		"report_title":           "사주 심층 해석 리포트",
		"summary":                "명격 총론",
		"personality":            "성격 특질",
		"career":                 "직업과 재물운",
		"relationship":           "연애와 결혼",
		"health":                 "건강 조언",
		"yearly_fortune":         "세운 흐름",
		"suggestions":            "개운 조언",
		"day_master":             "일주",
		"strength":               "강약",
		"five_elements":          "오행",
		"generated":              "생성 일시",
		"birth_info":             "출생 정보",
		"page":                   "페이지",
		"year":                   "유년",
		"analysis":               "해석",
		"contents":               "목차",
		"chart_overview":         "사주 명식 개요",
		"private_report":         "개인 사주 심층 해석",
		"chapter_chart_detail":   "정밀 명식",
		"chapter_destiny_depth":  "명격 심층 해석",
		"chapter_ten_gods_full":  "십신 전상 분석",
		"chapter_luck_cycle":     "평생 대운 흐름",
		"chapter_ten_year_years": "향후 10년 세운 상세",
		"chapter_career_depth":   "직업 심층 분석",
		"chapter_wealth_depth":   "재물 심층 분석",
		"chapter_love_depth":     "연애 심층 분석",
		"chapter_health_depth":   "건강 심층 분석",
		"chapter_element_tuning": "오행 조후",
		"chapter_remedies":       "당면 과제 해소 방안",
		"chapter_fortune_guide":  "오행 개운 가이드",
		"chapter_life_plan":      "평생 지침과 설계",
	},
}

// BuildReportPDFData 将排盘 + LLM 报告内容组装为模板数据。
func BuildReportPDFData(chart *model.ChartData, content model.ReportContent, genDate string) *ReportPDFData {
	locale := content.Locale
	if locale == "" {
		locale = "en"
	}
	labels, ok := reportSectionLabels[locale]
	if !ok {
		labels = reportSectionLabels["en"]
	}

	pLabels, ok := pillarLabels[locale]
	if !ok {
		pLabels = pillarLabels["en"]
	}

	pillars := []struct {
		pos   string
		p     model.Pillar
		label string
	}{
		{"year", chart.Pillars.Year, pLabels["year"]},
		{"month", chart.Pillars.Month, pLabels["month"]},
		{"day", chart.Pillars.Day, pLabels["day"]},
		{"hour", chart.Pillars.Hour, pLabels["hour"]},
	}

	displays := make([]PillarDisplay, 0, 4)
	for _, pp := range pillars {
		displays = append(displays, PillarDisplay{
			PositionLabel: pp.label,
			Stem:          pp.p.Stem,
			Branch:        pp.p.Branch,
			ElementColor:  elementColors[pp.p.StemElement],
		})
	}

	// 五行数量摘要
	var elements []elementSummaryItem
	totalElements := 0
	for _, count := range chart.FiveElementsCount {
		totalElements += count
	}
	elementOrder := []string{"木", "火", "土", "金", "水"}
	for _, el := range elementOrder {
		count := chart.FiveElementsCount[el]
		pct := 0
		if totalElements > 0 {
			pct = count * 100 / totalElements
		}
		elements = append(elements, elementSummaryItem{
			Element: el,
			Color:   elementColors[el],
			Count:   count,
			Width:   pct,
		})
	}

	fortuneItems := make([]fortuneItem, len(content.YearlyFortune))
	for i, yf := range content.YearlyFortune {
		fortuneItems[i] = fortuneItem{Year: yf.Year, Note: yf.Note}
	}

	hasTenYearChapter := false
	chapters := make([]ReportChapterPDFData, 0, len(content.Chapters))
	for _, chapter := range content.Chapters {
		if chapter.Key == "ten_year_years" {
			hasTenYearChapter = true
		}
		chapters = append(chapters, ReportChapterPDFData{
			No: chapter.No, Key: chapter.Key, Title: chapter.Title,
			Sections: splitChapterSections(chapter.Body),
			Cycles:   chapter.Cycles, Years: chapter.Years,
		})
	}

	dayMasterLabel := chart.DayMaster.Stem + " · " + chart.DayMaster.Element + " " + chart.DayMaster.YinYang

	return &ReportPDFData{
		Brand:             "FateLumen",
		Locale:            locale,
		ReportTitle:       labels["report_title"],
		DayMasterLabel:    dayMasterLabel,
		StrengthLevel:     localizedStrength(currentStrengthLevel(chart), locale),
		ElementBalance:    summarizeElements(chart.FiveElementsCount),
		GenDate:           genDate,
		SolarDate:         chart.Meta.SolarDate,
		LunarDate:         chart.Meta.LunarDate,
		Pillars:           displays,
		Elements:          elements,
		Content:           content,
		FortuneItems:      fortuneItems,
		HasTenYearChapter: hasTenYearChapter,
		Suggestions:       content.Suggestions,
		Chapters:          chapters,
		SectionLabels:     labels,
	}
}

func currentStrengthLevel(chart *model.ChartData) string {
	// New charts keep the current conclusion in Strength. StrengthV2 is read
	// first so historical snapshots created before that unification still render
	// the same conclusion that their report facts and prompts consumed.
	if chart.StrengthV2 != nil && chart.StrengthV2.Level != "" {
		return chart.StrengthV2.Level
	}
	return chart.Strength.Level
}

func localizedStrength(level, locale string) string {
	if labels := strengthLabels[locale]; labels != nil {
		if value := labels[level]; value != "" {
			return value
		}
	}
	return level
}

func splitChapterSections(body string) []ReportChapterSection {
	normalized := strings.ReplaceAll(body, "\r\n", "\n")
	blocks := strings.Split(strings.TrimSpace(normalized), "\n\n")
	sections := make([]ReportChapterSection, 0, len(blocks))
	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		clean := lines[:0]
		for _, line := range lines {
			if value := strings.TrimSpace(line); value != "" {
				clean = append(clean, value)
			}
		}
		if len(clean) == 0 {
			continue
		}
		section := ReportChapterSection{}
		if len(clean) > 1 && len([]rune(clean[0])) <= 32 {
			section.Title = clean[0]
			section.Paragraphs = clean[1:]
		} else {
			section.Paragraphs = clean
		}
		sections = append(sections, section)
	}
	return sections
}

func summarizeElements(counts map[string]int) string {
	order := []string{"木", "火", "土", "金", "水"}
	var parts []string
	for _, el := range order {
		c := counts[el]
		if c > 0 {
			parts = append(parts, el+"×"+fmt.Sprint(c))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}
	return result
}
