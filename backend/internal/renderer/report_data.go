package renderer

import (
	"fmt"

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

// ---------- 四柱 ----------

// ---------- 报告 PDF 模板数据 ----------

// ReportPDFData 深度报告 PDF 模板数据。
// 严格遵守 P2：所有文案/标题不出现 "AI"。
type ReportPDFData struct {
	Brand               string
	Locale              string
	ReportTitle         string
	DayMasterLabel      string
	StrengthLevel       string
	ElementBalance      string
	GenDate             string
	SolarDate           string
	LunarDate           string
	Pillars             []PillarDisplay
	Elements            []elementSummaryItem
	Content             model.ReportContent
	FortuneItems        []fortuneItem
	Suggestions         []string
	SectionLabels       map[string]string
}

var reportSectionLabels = map[string]map[string]string{
	"en": {
		"report_title":    "Bazi Deep Reading Report",
		"summary":         "Destiny Overview",
		"personality":     "Personality",
		"career":          "Career & Wealth",
		"relationship":    "Love & Marriage",
		"health":          "Health & Wellness",
		"yearly_fortune":  "Yearly Fortune",
		"suggestions":     "Guidance & Suggestions",
		"day_master":      "Day Master",
		"strength":        "Strength",
		"five_elements":   "Five Elements",
		"generated":       "Generated",
		"birth_info":      "Birth Information",
		"page":            "Page",
		"year":            "Year",
		"analysis":        "Analysis",
	},
	"zh": {
		"report_title":    "八字深度解读报告",
		"summary":         "命格总论",
		"personality":     "性格特质",
		"career":          "事业财运",
		"relationship":    "感情婚姻",
		"health":          "健康提示",
		"yearly_fortune":  "流年运势",
		"suggestions":     "开运建议",
		"day_master":      "日主",
		"strength":        "强弱",
		"five_elements":   "五行",
		"generated":       "生成时间",
		"birth_info":      "出生信息",
		"page":            "页",
		"year":            "流年",
		"analysis":        "解读",
	},
	"ja": {
		"report_title":    "四柱推命 詳細鑑定書",
		"summary":         "命式総論",
		"personality":     "性格の特質",
		"career":          "仕事と財運",
		"relationship":    "恋愛と結婚",
		"health":          "健康のヒント",
		"yearly_fortune":  "年運の流れ",
		"suggestions":     "開運のヒント",
		"day_master":      "日主",
		"strength":        "強弱",
		"five_elements":   "五行",
		"generated":       "作成日時",
		"birth_info":      "生年月日情報",
		"page":            "ページ",
		"year":            "流年",
		"analysis":        "解説",
	},
	"ko": {
		"report_title":    "사주 심층 해석 리포트",
		"summary":         "명격 총론",
		"personality":     "성격 특질",
		"career":          "직업과 재물운",
		"relationship":    "연애와 결혼",
		"health":          "건강 조언",
		"yearly_fortune":  "세운 흐름",
		"suggestions":     "개운 조언",
		"day_master":      "일주",
		"strength":        "강약",
		"five_elements":   "오행",
		"generated":       "생성 일시",
		"birth_info":      "출생 정보",
		"page":            "페이지",
		"year":            "유년",
		"analysis":        "해석",
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

	dayMasterLabel := chart.DayMaster.Stem + " · " + chart.DayMaster.Element + " " + chart.DayMaster.YinYang

	return &ReportPDFData{
		Brand:          "FateLumen",
		Locale:         locale,
		ReportTitle:    labels["report_title"],
		DayMasterLabel: dayMasterLabel,
		StrengthLevel:  chart.Strength.Level,
		ElementBalance: summarizeElements(chart.FiveElementsCount),
		GenDate:        genDate,
		SolarDate:      chart.Meta.SolarDate,
		LunarDate:      chart.Meta.LunarDate,
		Pillars:        displays,
		Elements:       elements,
		Content:        content,
		FortuneItems:   fortuneItems,
		Suggestions:    content.Suggestions,
		SectionLabels:  labels,
	}
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
