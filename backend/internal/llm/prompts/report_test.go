package prompts

import (
	"encoding/json"
	"strings"
	"testing"

	"fatelumen/backend/internal/bazi"
	"fatelumen/backend/internal/model"
)

var expectedReportChapterKeys = []string{
	"destiny_depth",
	"ten_gods_full",
	"luck_cycle",
	"ten_year_years",
	"career_depth",
	"wealth_depth",
	"love_depth",
	"health_depth",
	"element_tuning",
	"life_plan",
}

func sampleChart(t *testing.T) *model.ChartData {
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

func TestReportSystemPrompt(t *testing.T) {
	if ReportSystemPrompt == "" {
		t.Fatal("ReportSystemPrompt is empty")
	}
	lower := strings.ToLower(ReportSystemPrompt)

	for _, kw := range []string{"json", "summary", "personality", "career", "relationship", "health", "yearly_fortune", "suggestions", "chapters"} {
		if !strings.Contains(lower, kw) {
			t.Errorf("ReportSystemPrompt missing keyword: %s", kw)
		}
	}

	for _, kw := range []string{"no medical diagnosis", "absolute language", "do not use doom", "fatalistic", "not acceptable"} {
		if !strings.Contains(lower, kw) {
			t.Errorf("ReportSystemPrompt missing constraint: %s", kw)
		}
	}

	if !strings.Contains(ReportSystemPrompt, "EXACTLY 10 entries") {
		t.Fatal("ReportSystemPrompt must require exactly 10 chapters")
	}
}

func TestReportNoAI(t *testing.T) {
	lower := strings.ToLower(ReportSystemPrompt)

	words := strings.Fields(lower)
	for _, w := range words {
		if w == "ai" {
			t.Error("ReportSystemPrompt contains standalone forbidden word")
		}
	}
	if strings.Contains(lower, "artificial") {
		t.Error("ReportSystemPrompt contains forbidden wording")
	}

	chart := sampleChart(t)
	userPrompt, err := BuildReportUserPrompt("en", chart)
	if err != nil {
		t.Fatalf("BuildReportUserPrompt: %v", err)
	}

	lowerUser := strings.ToLower(userPrompt)
	userWords := strings.Fields(lowerUser)
	for _, w := range userWords {
		if w == "ai" {
			t.Error("user prompt contains standalone forbidden word")
		}
	}
}

func TestBuildReportUserPrompt(t *testing.T) {
	chart := sampleChart(t)

	userPrompt, err := BuildReportUserPrompt("en", chart)
	if err != nil {
		t.Fatalf("BuildReportUserPrompt: %v", err)
	}

	if !strings.Contains(userPrompt, "locale: en") {
		t.Error("user prompt missing locale")
	}
	for _, kw := range []string{"pillars", "day_master", "five_elements_count", "strength", "luck_cycles", "annual_fortunes"} {
		if !strings.Contains(userPrompt, kw) {
			t.Errorf("user prompt missing chart data: %s", kw)
		}
	}
	for _, kw := range []string{"summary", "personality", "career", "relationship", "health", "yearly_fortune", "suggestions", "chapters"} {
		if !strings.Contains(userPrompt, kw) {
			t.Errorf("user prompt missing section: %s", kw)
		}
	}
	if !strings.Contains(userPrompt, "EXACTLY 10 entries") {
		t.Fatal("user prompt must require exactly 10 chapters")
	}
}

func TestBuildReportUserPrompt_Locales(t *testing.T) {
	chart := sampleChart(t)
	for _, loc := range []string{"en", "zh", "ja", "ko"} {
		prompt, err := BuildReportUserPrompt(loc, chart)
		if err != nil {
			t.Errorf("locale %s: %v", loc, err)
			continue
		}
		if !strings.Contains(prompt, "locale: "+loc) {
			t.Errorf("locale %s not in prompt", loc)
		}
	}
}

func TestReportGroups_ChapterContract(t *testing.T) {
	groups := ReportGroups()
	if len(groups) != 13 {
		t.Fatalf("expected 13 report groups, got %d", len(groups))
	}

	combined := ""
	for _, g := range groups {
		combined += "\n" + g.System
	}

	for i, key := range expectedReportChapterKeys {
		want := `"` + key + `"`
		if !strings.Contains(ReportSystemPrompt, want) {
			t.Errorf("system prompt missing chapter key %d %s", i+1, key)
		}
		if !strings.Contains(combined, key) {
			t.Errorf("group prompts missing chapter key %d %s", i+1, key)
		}
	}

	for _, removed := range []string{"chart_detail", "remedies", "fortune_guide"} {
		if strings.Contains(combined, removed) {
			t.Errorf("group prompts should not generate removed chapter key %s", removed)
		}
	}
}

func TestReportContentJSON_RoundTrip(t *testing.T) {
	content := model.ReportContent{
		Locale:       "en",
		SummaryLine:  "A commanding Fire rising at noon, blessed with both brilliance and warmth.",
		Summary:      "Born under the Bing day master, the chart reveals a strong Fire and Wood pattern. The day master sits on the Horse branch, amplifying visibility, leadership, and expressive drive.",
		Personality:  "The Bing Fire day master brings warmth, enthusiasm, and the desire to inspire others. The ten-god pattern adds intellectual curiosity and practical emotional intelligence.",
		Career:       "With Fire as the dominant element and Wood feeding it, career growth fits creative, leadership, educational, or advisory domains. Wealth grows best through sustained expertise.",
		Relationship: "The day branch shows how the spouse palace carries both support and a need for mature communication. Stable bonds form through consistency and shared rhythm.",
		Health:       "The five-element balance suggests attention to rest, hydration, and emotional regulation as general wellness habits. This is not medical advice.",
		YearlyFortune: []model.YearlyFortuneItem{
			{Year: 2026, Note: "The year brings visibility and career momentum."},
			{Year: 2027, Note: "The year favors consolidation and relationship building."},
			{Year: 2028, Note: "The year asks for more careful financial planning."},
		},
		Suggestions: []string{
			"Channel creative energy into a structured daily routine.",
			"Use rest and hydration to balance excess Fire.",
			"Prefer long-term skill building over speculation.",
			"Build a grounded support network.",
			"Schedule regular quiet reflection.",
		},
		Chapters: makeReportChapters(),
	}

	b, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Marshal ReportContent: %v", err)
	}
	if !json.Valid(b) {
		t.Fatal("marshaled ReportContent is not valid JSON")
	}

	var restored model.ReportContent
	if err := json.Unmarshal(b, &restored); err != nil {
		t.Fatalf("Unmarshal ReportContent: %v", err)
	}

	if restored.SummaryLine != content.SummaryLine {
		t.Error("SummaryLine mismatch after round-trip")
	}
	if restored.Personality != content.Personality {
		t.Error("Personality mismatch after round-trip")
	}
	if len(restored.YearlyFortune) != 3 {
		t.Fatalf("expected 3 yearly_fortune items, got %d", len(restored.YearlyFortune))
	}
	if len(restored.Suggestions) != 5 {
		t.Fatalf("expected 5 suggestions, got %d", len(restored.Suggestions))
	}
	assertTenChapters(t, restored.Chapters)
}

func TestReportContent_RequiredFields(t *testing.T) {
	raw := `{
		"summary": "Test summary",
		"summary_line": "Test summary line",
		"personality": "Test personality",
		"career": "Test career",
		"relationship": "Test relationship",
		"health": "Test health",
		"yearly_fortune": [{"year": 2026, "note": "Test note"}],
		"suggestions": ["Suggestion 1"]
	}`

	var content model.ReportContent
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		t.Fatalf("Unmarshal minimal ReportContent: %v", err)
	}

	if content.Summary == "" {
		t.Error("summary required but empty")
	}
	if content.SummaryLine == "" {
		t.Error("summary_line required but empty")
	}
}

func TestReportContent_WithChapters(t *testing.T) {
	content := model.ReportContent{
		Locale:      "zh",
		SummaryLine: "火明之命，光照四方",
		Summary:     "详细总论...",
		Personality: "性格分析...",
		Chapters:    makeReportChapters(),
	}

	b, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Marshal with chapters: %v", err)
	}

	var restored model.ReportContent
	if err := json.Unmarshal(b, &restored); err != nil {
		t.Fatalf("Unmarshal with chapters: %v", err)
	}

	assertTenChapters(t, restored.Chapters)
}

func TestReportSystemPrompt_JSONSchema(t *testing.T) {
	requiredKeys := []string{
		`"summary"`, `"summary_line"`, `"personality"`,
		`"career"`, `"relationship"`, `"health"`,
		`"yearly_fortune"`, `"suggestions"`, `"chapters"`,
	}
	for _, key := range requiredKeys {
		if !strings.Contains(ReportSystemPrompt, key) {
			t.Errorf("ReportSystemPrompt missing JSON key: %s", key)
		}
	}
}

func makeReportChapters() []model.Chapter {
	titles := []string{
		"In-Depth Destiny Reading",
		"Full Ten-Gods Panorama",
		"Lifelong Luck-Cycle Trend",
		"Next Ten Years",
		"Career Deep Dive",
		"Wealth Deep Dive",
		"Relationship Deep Dive",
		"Health Deep Dive",
		"Five-Element Climate Balance",
		"Lifelong Guidance",
	}
	chapters := make([]model.Chapter, 0, len(expectedReportChapterKeys))
	for i, key := range expectedReportChapterKeys {
		chapters = append(chapters, model.Chapter{
			No:    i + 1,
			Key:   key,
			Title: titles[i],
			Body:  "Chapter body with substantive chart-specific analysis.",
		})
	}
	return chapters
}

func assertTenChapters(t *testing.T, chapters []model.Chapter) {
	t.Helper()
	if len(chapters) != len(expectedReportChapterKeys) {
		t.Fatalf("expected %d chapters, got %d", len(expectedReportChapterKeys), len(chapters))
	}
	for i, ch := range chapters {
		if ch.No != i+1 {
			t.Errorf("chapter %d: expected No=%d, got %d", i, i+1, ch.No)
		}
		if ch.Key != expectedReportChapterKeys[i] {
			t.Errorf("chapter %d: expected Key=%s, got %s", i, expectedReportChapterKeys[i], ch.Key)
		}
		if ch.Title == "" {
			t.Errorf("chapter %d: Title is empty", i)
		}
		if ch.Body == "" {
			t.Errorf("chapter %d: Body is empty", i)
		}
	}
}
