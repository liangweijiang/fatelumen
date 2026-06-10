package prompts

import (
	"encoding/json"
	"strings"
	"testing"

	"fatelumen/backend/internal/bazi"
	"fatelumen/backend/internal/model"
)

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

	// 关键约束词
	for _, kw := range []string{"json", "summary", "personality", "career", "relationship", "health", "yearly_fortune", "suggestions"} {
		if !strings.Contains(lower, kw) {
			t.Errorf("ReportSystemPrompt missing keyword: %s", kw)
		}
	}

	// 禁止绝对化
	for _, kw := range []string{"no medical diagnosis", "absolute language", "do not use doom", "fatalistic", "not acceptable"} {
		if !strings.Contains(lower, kw) {
			t.Errorf("ReportSystemPrompt missing constraint: %s", kw)
		}
	}
}

func TestReportNoAI(t *testing.T) {
	lower := strings.ToLower(ReportSystemPrompt)

	// P2 红线：全文不含 "AI"
	words := strings.Fields(lower)
	for _, w := range words {
		if w == "ai" {
			t.Error("ReportSystemPrompt contains standalone 'AI' word — P2 violation")
		}
	}
	if strings.Contains(lower, "artificial") {
		t.Error("ReportSystemPrompt contains 'artificial' — P2 violation")
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
			t.Error("user prompt contains standalone 'AI' word — P2 violation")
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
	// 四柱等关键信息应出现在 prompt 中
	for _, kw := range []string{"pillars", "day_master", "five_elements_count", "strength", "luck_cycles"} {
		if !strings.Contains(userPrompt, kw) {
			t.Errorf("user prompt missing chart data: %s", kw)
		}
	}
	// 要求章节
	for _, kw := range []string{"summary", "personality", "career", "relationship", "health", "yearly_fortune", "suggestions"} {
		if !strings.Contains(userPrompt, kw) {
			t.Errorf("user prompt missing section: %s", kw)
		}
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

func TestReportContentJSON_RoundTrip(t *testing.T) {
	content := model.ReportContent{
		Locale:      "en",
		SummaryLine: "A commanding Fire rising at noon, blessed with both brilliance and warmth.",
		Summary:     "Born under the Bing (Yang Fire) day master, your chart reveals a striking elemental configuration dominated by Fire and Wood. The day master sits on the Horse branch, forming a Fire-Wood synergy that amplifies creativity and leadership. The strength assessment suggests a balanced constitution leaning toward yang — outward, expressive, and transformative. This report examines each pillar in context, tracing how the interplay of heavenly stems and earthly branches shapes your destiny.",
		Personality: "Your Bing Fire day master endows you with natural warmth, enthusiasm, and an innate ability to inspire others. The presence of Yin Wood in the month pillar adds depth and flexibility to your otherwise direct nature. Ten gods analysis reveals a dominance of Eating God and Direct Resource, indicating a personality that balances intellectual pursuit with emotional intelligence. You thrive in environments where you can express ideas freely, and you possess an almost magnetic quality that draws collaborators and opportunities alike.",
		Career:      "With Fire as your dominant element and Wood feeding it, your career path aligns naturally with creative, leadership, or educational domains. The favorable elements of Earth and Metal suggest that structured environments and systematic work provide the grounding your expansive Fire nature needs. Major luck cycles during your thirties and forties activate wealth stars in the earthly branches, pointing to significant financial growth during these decades. Cultivate patience — the strongest returns come from sustained effort, not quick wins.",
		Relationship: "Your day branch (Horse) forms a harmonious relationship with the month branch, indicating supportive early family dynamics. The spouse palace is occupied by a resource star, suggesting a partner who is nurturing, intelligent, and perhaps older or more mature. Peach blossom indicators in the year pillar suggest early romantic attention, but lasting bonds form after age 28 when the luck cycle enters a more stable Wood phase.",
		Health:      "The Fire dominance in your chart suggests attention to heart and circulatory wellness as a general wellness priority. The relative weakness of Water in your elemental balance points to the importance of hydration, rest, and kidney care. These are general wellness observations based on traditional elemental theory and should not be taken as medical advice. Regular exercise that grounds your energetic nature — such as walking in nature or yoga — provides excellent balance.",
		YearlyFortune: []model.YearlyFortuneItem{
			{Year: 2026, Note: "The Bing-Wu year resonates strongly with your Fire day master, bringing heightened visibility and career momentum."},
			{Year: 2027, Note: "Ding-Wei year introduces Earth element stability — an excellent period for consolidating gains and relationship building."},
			{Year: 2028, Note: "Wu-Shen year activates Metal, your wealth element, through the Monkey branch — anticipate financial shifts and strategic decisions."},
		},
		Suggestions: []string{
			"Channel creative Fire energy into a structured daily routine to maximize productivity.",
			"Cultivate Water-element activities (swimming, meditation near water) to balance elemental excess.",
			"During Metal luck cycles, invest in long-term assets rather than speculative ventures.",
			"Build a support network of Earth-element personalities who provide grounded perspective.",
			"Schedule regular periods of quiet reflection to prevent Fire burnout.",
		},
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
}

func TestReportContent_RequiredFields(t *testing.T) {
	// Minimum valid JSON
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
		SummaryLine: "火旺之命，光明磊落",
		Summary:     "详细总论...",
		Personality: "性格分析...",
		Chapters: []model.Chapter{
			{No: 1, Key: "overview", Title: "总览", Body: "命盘概览..."},
		},
	}

	b, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Marshal with chapters: %v", err)
	}

	var restored model.ReportContent
	if err := json.Unmarshal(b, &restored); err != nil {
		t.Fatalf("Unmarshal with chapters: %v", err)
	}

	if len(restored.Chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(restored.Chapters))
	}
}

// System prompt must contain all required JSON keys
func TestReportSystemPrompt_JSONSchema(t *testing.T) {
	requiredKeys := []string{
		`"summary"`, `"summary_line"`, `"personality"`,
		`"career"`, `"relationship"`, `"health"`,
		`"yearly_fortune"`, `"suggestions"`,
	}
	for _, key := range requiredKeys {
		if !strings.Contains(ReportSystemPrompt, key) {
			t.Errorf("ReportSystemPrompt missing JSON key: %s", key)
		}
	}
}
