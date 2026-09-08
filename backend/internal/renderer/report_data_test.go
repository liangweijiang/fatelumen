package renderer

import (
	"bytes"
	"strings"
	"testing"

	"fatelumen/backend/internal/bazi"
	"fatelumen/backend/internal/model"
)

func sampleReportChart(t *testing.T) *model.ChartData {
	t.Helper()
	chart, err := bazi.Calculate(bazi.BirthInput{
		Gender:       1,
		CalendarType: 0,
		Year:         1990,
		Month:        8,
		Day:          15,
		Hour:         14,
		Minute:       30,
	})
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	return chart
}

func sampleReportContent() model.ReportContent {
	content := model.ReportContent{
		Locale:       "en",
		SummaryLine:  "A bright Wood rising at dawn, destined to illuminate.",
		Summary:      "Your chart reveals a dynamic balance of Wood and Fire elements, creating a personality that is both ambitious and charismatic. The day master sits on a strong foundation, supported by favorable elements in the month and year pillars.",
		Personality:  "Your Yang Wood day master gives you a natural leadership quality — you are the tall tree in the forest, visible and inspiring. Combined with Fire in the month branch, your creative energy is formidable.",
		Career:       "The presence of Earth in your hour pillar suggests strong wealth potential realized through steady, systematic effort. Your favorable Metal element indicates success in structured, analytical fields.",
		Relationship: "Your day branch reveals a harmonious Peach Blossom configuration, suggesting warm and balanced romantic relationships. The spouse palace is occupied by a resource star, indicating a supportive partner.",
		Health:       "The strong Wood element in your chart relates to liver and detoxification functions in traditional elemental theory. This is a general wellness observation and should not be taken as medical advice.",
		YearlyFortune: []model.YearlyFortuneItem{
			{Year: 2026, Note: "This year's Fire element supports your Wood day master, bringing career visibility and recognition."},
			{Year: 2027, Note: "Earth element stabilization — focus on consolidating achievements and relationship building."},
			{Year: 2028, Note: "Metal year activates your wealth star through the Monkey branch. Strategic decisions ahead."},
		},
		Suggestions: []string{
			"Channel Wood energy into daily creative practice.",
			"Cultivate patience during Earth-dominant months.",
			"Build a network of Metal-element mentors for career guidance.",
			"Practice mindfulness to balance Fire's intensity.",
			"Incorporate Water-element activities for elemental harmony.",
		},
	}
	chapterBodies := []struct{ key, title, body string }{
		{"destiny_depth", "In-Depth Destiny Reading", "Core Pattern\nThe chart shows a clear and balanced structure.\n\nPractical Focus\nUse steady action to turn potential into results."},
		{"ten_gods_full", "Full Ten-Gods Panorama", "Relationships\nThe ten-god pattern emphasizes cooperation and discernment."},
		{"luck_cycle", "Lifelong Luck-Cycle Trend", "Major Cycles\nEach phase has a distinct rhythm and focus."},
		{"ten_year_years", "Next Ten Years, Year by Year", "Ten-Year View\nThe coming decade rewards patient preparation."},
		{"career_depth", "Career Deep Dive", "Career Direction\nStructured work creates durable progress."},
		{"wealth_depth", "Wealth Deep Dive", "Wealth Pattern\nBuild reserves before expanding commitments."},
		{"love_depth", "Relationship Deep Dive", "Partnership\nClear communication supports lasting trust."},
		{"health_depth", "Health Deep Dive", "Wellness\nKeep a sustainable daily rhythm."},
		{"element_tuning", "Five-Element Climate Balance", "Balance\nUse rest and movement to restore balance."},
		{"life_plan", "Lifelong Guidance & Planning", "Long-Term Plan\nReview priorities at each major transition."},
	}
	for i, item := range chapterBodies {
		content.Chapters = append(content.Chapters, model.Chapter{No: i + 1, Key: item.key, Title: item.title, Body: item.body})
	}
	return content
}

func TestBuildReportPDFData_Fields(t *testing.T) {
	chart := sampleReportChart(t)
	content := sampleReportContent()
	data := BuildReportPDFData(chart, content, "2026-06-11")

	if data.Brand != "FateLumen" {
		t.Errorf("expected Brand FateLumen, got %s", data.Brand)
	}
	if data.Locale != "en" {
		t.Errorf("expected locale en, got %s", data.Locale)
	}
	if len(data.Pillars) != 4 {
		t.Fatalf("expected 4 pillars, got %d", len(data.Pillars))
	}
	if len(data.Elements) != 5 {
		t.Fatalf("expected 5 elements, got %d", len(data.Elements))
	}
	if data.Pillars[0].ElementColor == "" {
		t.Error("pillar element color should not be empty")
	}
	if data.DayMasterLabel == "" {
		t.Error("day master label should not be empty")
	}
	if data.StrengthLevel == "" {
		t.Error("strength level should not be empty")
	}
	if data.ElementBalance == "" {
		t.Error("element balance should not be empty")
	}
	if len(data.FortuneItems) != 3 {
		t.Fatalf("expected 3 fortune items, got %d", len(data.FortuneItems))
	}
	if len(data.Suggestions) != 5 {
		t.Fatalf("expected 5 suggestions, got %d", len(data.Suggestions))
	}
	if data.SectionLabels["summary"] == "" {
		t.Error("SectionLabels summary should not be empty")
	}
}

func TestReportPDFTemplate_HidesLegacyYearlyFortuneWhenTenYearChapterExists(t *testing.T) {
	chart := sampleReportChart(t)
	content := sampleReportContent()
	data := BuildReportPDFData(chart, content, "2026-06-11")

	if !data.HasTenYearChapter {
		t.Fatal("expected ten-year chapter marker")
	}

	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, data); err != nil {
		t.Fatalf("template execute: %v", err)
	}
	html := buf.String()

	if strings.Contains(html, content.YearlyFortune[0].Note) {
		t.Error("legacy yearly fortune content should not be rendered in the ten-chapter report")
	}
	if !strings.Contains(html, "Next Ten Years, Year by Year") {
		t.Error("ten-year chapter should still be rendered")
	}
}

func TestBuildReportPDFData_Locales(t *testing.T) {
	chart := sampleReportChart(t)
	chart.Strength.Level = "balanced"
	wantStrength := map[string]string{"en": "Balanced", "zh": "中和", "ja": "中和", "ko": "중화"}
	for _, loc := range []string{"en", "zh", "ja", "ko"} {
		content := sampleReportContent()
		content.Locale = loc
		data := BuildReportPDFData(chart, content, "2026-06-11")
		if data.Locale != loc {
			t.Errorf("expected locale %s, got %s", loc, data.Locale)
		}
		if data.StrengthLevel != wantStrength[loc] {
			t.Errorf("locale %s: expected strength %s, got %s", loc, wantStrength[loc], data.StrengthLevel)
		}
	}
}

func TestReportPDFTemplate_Render(t *testing.T) {
	chart := sampleReportChart(t)
	content := sampleReportContent()
	data := BuildReportPDFData(chart, content, "2026-06-11")

	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, data); err != nil {
		t.Fatalf("template execute: %v", err)
	}
	html := buf.String()

	// 关键内容检查（html/template 会转义 &）
	for _, kw := range []string{
		"FateLumen", "Bazi Deep Reading Report",
		"Four Pillars Overview", "Contents", "Day Master",
		"In-Depth Destiny Reading", "Lifelong Guidance", "Core Pattern",
	} {
		if !strings.Contains(html, kw) {
			t.Errorf("rendered HTML missing: %s", kw)
		}
	}

	// 封面日期信息
	if !strings.Contains(html, "2026-06-11") {
		t.Error("rendered HTML missing generation date")
	}

	// 样式
	if !strings.Contains(html, "page-break") {
		t.Error("rendered HTML missing page breaks")
	}
	if !strings.Contains(html, "#f5f0e6") {
		t.Error("rendered HTML missing bg color token")
	}
	if !strings.Contains(html, "#9a761e") {
		t.Error("rendered HTML missing gold token")
	}

	// 四柱
	for _, ps := range data.Pillars {
		if !strings.Contains(html, ps.Stem) || !strings.Contains(html, ps.Branch) {
			t.Errorf("rendered HTML missing pillar: %s%s", ps.Stem, ps.Branch)
		}
	}

	if strings.Contains(html, "suggestion-list") {
		t.Error("legacy suggestion section should not be rendered")
	}

	// @page 尺寸
	if !strings.Contains(html, "size:A4") {
		t.Error("rendered HTML missing A4 page size")
	}
}

func TestReportPDFNoAI(t *testing.T) {
	chart := sampleReportChart(t)
	content := sampleReportContent()
	data := BuildReportPDFData(chart, content, "2026-06-11")

	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, data); err != nil {
		t.Fatalf("template execute: %v", err)
	}
	html := buf.String()

	lower := strings.ToLower(html)
	if strings.Contains(lower, " ai ") || strings.Contains(lower, "ai.") || strings.Contains(lower, "ai,") {
		t.Error("rendered HTML contains 'AI' — P2 violation")
	}
	if strings.Contains(lower, "artificial") {
		t.Error("rendered HTML contains 'artificial' — P2 violation")
	}
}

func TestReportPDFTemplate_ChapterContent(t *testing.T) {
	chart := sampleReportChart(t)
	content := sampleReportContent()
	data := BuildReportPDFData(chart, content, "2026-06-11")

	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, data); err != nil {
		t.Fatalf("template execute: %v", err)
	}
	html := buf.String()

	// 十章正文应出现在 HTML 中，旧摘要字段不再重复渲染。
	for _, text := range []string{
		content.Chapters[0].Body,
		content.Chapters[9].Body,
	} {
		firstLine := strings.Split(text, "\n")[0]
		if !strings.Contains(html, firstLine) {
			t.Errorf("rendered HTML missing content section: %s", firstLine)
		}
	}
	if strings.Contains(html, content.Personality) {
		t.Error("legacy personality section should not be duplicated")
	}
}

func TestSplitChapterSections(t *testing.T) {
	sections := splitChapterSections("模块一\n第一段\n第二段\n\n模块二\n第三段")
	if len(sections) != 2 || sections[0].Title != "模块一" || len(sections[0].Paragraphs) != 2 {
		t.Fatalf("unexpected sections: %#v", sections)
	}
}
