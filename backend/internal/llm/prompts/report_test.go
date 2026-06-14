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
	for _, kw := range []string{"json", "summary", "personality", "career", "relationship", "health", "yearly_fortune", "suggestions", "chapters"} {
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
	for _, kw := range []string{"summary", "personality", "career", "relationship", "health", "yearly_fortune", "suggestions", "chapters"} {
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
		Chapters: []model.Chapter{
			{No: 1, Key: "chart_detail", Title: "Refined Chart Reading", Body: "chapter 1 body..."},
			{No: 2, Key: "destiny_depth", Title: "In-Depth Destiny Reading", Body: "chapter 2 body..."},
			{No: 3, Key: "ten_gods_full", Title: "Full Ten-Gods Panorama", Body: "chapter 3 body..."},
			{No: 4, Key: "luck_cycle", Title: "Lifelong Luck-Cycle Trend", Body: "chapter 4 body..."},
			{No: 5, Key: "ten_year_years", Title: "Next Ten Years", Body: "chapter 5 body..."},
			{No: 6, Key: "career_depth", Title: "Career Deep Dive", Body: "chapter 6 body..."},
			{No: 7, Key: "wealth_depth", Title: "Wealth Deep Dive", Body: "chapter 7 body..."},
			{No: 8, Key: "love_depth", Title: "Relationship Deep Dive", Body: "chapter 8 body..."},
			{No: 9, Key: "health_depth", Title: "Health Deep Dive", Body: "chapter 9 body..."},
			{No: 10, Key: "remedies", Title: "Remedies", Body: "chapter 10 body..."},
			{No: 11, Key: "fortune_guide", Title: "Fortune Guide", Body: "chapter 11 body..."},
			{No: 12, Key: "life_plan", Title: "Lifelong Guidance", Body: "chapter 12 body..."},
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
	if len(restored.Chapters) != 12 {
		t.Fatalf("expected 12 chapters, got %d", len(restored.Chapters))
	}
	for i, ch := range restored.Chapters {
		if ch.No != i+1 {
			t.Errorf("chapter %d: expected No=%d, got %d", i, i+1, ch.No)
		}
		if ch.Key == "" {
			t.Errorf("chapter %d: Key is empty", i)
		}
		if ch.Body == "" {
			t.Errorf("chapter %d: Body is empty", i)
		}
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
			{No: 1, Key: "chart_detail", Title: "Refined Chart Reading", Body: "Chapter 1 body with at least 300 words of substantive analysis covering the four pillars, hidden stems, ten gods, nayin, and luck-cycle onset. This chapter must reference the actual stems, branches, and elements present in the given chart rather than generic descriptions. The analysis should trace how the heavenly stems interact with earthly branches to form the destiny pattern, noting any special combinations or clashes."},
			{No: 2, Key: "destiny_depth", Title: "In-Depth Destiny Reading", Body: "Chapter 2 body with detailed strength scoring rationale and favorable/unfavorable god analysis. Explain the structural-pattern mechanics — the WHY behind the chart configuration. Cover the interaction between the day master's season, the month branch, and the overall element distribution. Discuss which gods are useful and why, referencing the actual ten-god derivations from the chart."},
			{No: 3, Key: "ten_gods_full", Title: "Full Ten-Gods Panorama", Body: "Chapter 3 panoramic analysis of all ten gods including those from hidden stems if present. Examine their influence on personality traits, behavioral patterns, and family dynamics. Map each ten god to specific life domains and explain how their interactions shape the native's approach to relationships, career, and personal growth."},
			{No: 4, Key: "luck_cycle", Title: "Lifelong Luck-Cycle Trend", Body: "Chapter 4 traces the lifelong luck-cycle trend year by year, identifying auspicious levels and pivotal turning years based on the chart's luck cycles. Provide a textual narrative of how each major luck cycle phase affects the native's life trajectory, highlighting key transition years and their significance."},
			{No: 5, Key: "ten_year_years", Title: "Next Ten Years, Year by Year", Body: "Chapter 5 offers a per-year four-quadrant view for the next ten years, covering career, wealth, relationships, and health. For each year, identify key months that deserve attention and describe the dominant elemental influences shaping each quadrant."},
			{No: 6, Key: "career_depth", Title: "Career Deep Dive", Body: "Chapter 6 deep-dives into career fit by industry, identifying optimal promotion years, job-change timing, entrepreneurship windows, and characteristics of beneficial mentors and partners. Align career recommendations with the chart's favorable elements and ten-god configuration."},
			{No: 7, Key: "wealth_depth", Title: "Wealth Deep Dive", Body: "Chapter 7 analyzes wealth patterns including peak months for regular income and windfall wealth, saving cycles, caution years for financial decisions, and auspicious numbers and colors for wealth enhancement. Reference the chart's wealth star positions and interactions."},
			{No: 8, Key: "love_depth", Title: "Relationship Deep Dive", Body: "Chapter 8 examines relationships in depth, describing ideal partner traits derived from the spouse palace and day branch, identifying the best years to meet significant partners, and offering compatibility insights using zodiac and chart matching principles."},
			{No: 9, Key: "health_depth", Title: "Health Deep Dive", Body: "Chapter 9 provides health analysis through the five-element balance lens, noting predispositions and wellness prevention strategies without making any medical diagnosis. Identify high-risk months based on elemental clashes and recommend appropriate exercise and dietary styles."},
			{No: 10, Key: "remedies", Title: "Remedies for Near-Term Challenges", Body: "Chapter 10 presents actionable remedies for near-term challenges with step-by-step approaches, auspicious-date guidance for important decisions, and five-element adjustment strategies. Tailor each remedy to the chart's specific elemental weaknesses and upcoming luck-cycle phases."},
			{No: 11, Key: "fortune_guide", Title: "Five-Element Fortune Guide", Body: "Chapter 11 offers a five-element fortune guide covering auspicious colors, crystals, directions, home layout recommendations, optimal career directions, and benefactor zodiac signs. Ground each recommendation in the chart's actual element counts and favorable god analysis."},
			{No: 12, Key: "life_plan", Title: "Lifelong Guidance & Planning", Body: "Chapter 12 provides lifelong guidance with custom planning for youth, midlife, and elder years. Cover key habits to cultivate, pitfalls to avoid, and milestone planning aligned with major luck-cycle transitions."},
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

	if len(restored.Chapters) != 12 {
		t.Fatalf("expected 12 chapters, got %d", len(restored.Chapters))
	}
}

// System prompt must contain all required JSON keys
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
