package prompts

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"fatelumen/backend/internal/bazi"
	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/model"
)

type mockLLM struct {
	fixedJSON string
}

func (m *mockLLM) Name() string { return "mock" }
func (m *mockLLM) GenerateJSON(ctx context.Context, system, user string, opts ...llm.Option) (string, error) {
	return m.fixedJSON, nil
}

func TestBuildQuickUserPrompt(t *testing.T) {
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

	userPrompt, err := BuildQuickUserPrompt("en", chart)
	if err != nil {
		t.Fatalf("BuildQuickUserPrompt: %v", err)
	}

	if !strings.Contains(userPrompt, "locale: en") {
		t.Error("user prompt missing locale")
	}
	if !strings.Contains(userPrompt, "summary_line") {
		t.Error("user prompt missing summary_line")
	}
	if !strings.Contains(userPrompt, "personality") {
		t.Error("user prompt missing personality")
	}
	if !strings.Contains(userPrompt, "day_master") || !strings.Contains(userPrompt, "pillars") {
		t.Error("user prompt missing chart data")
	}
	if strings.Contains(userPrompt, "AI") {
		t.Error("user prompt contains 'AI' — P2 violation")
	}
	if strings.Contains(QuickSystemPrompt, "AI") {
		t.Error("system prompt contains 'AI' — P2 violation")
	}
}

func TestQuickReadingParsing(t *testing.T) {
	fixedJSON := `{
  "summary_line": "A resilient Water personality flowing with quiet strength",
  "personality": "Born under the Ren (Yang Water) day master, you possess a deep, adaptable nature. Your mind works like an ocean — vast, reflective, and capable of navigating any current with grace.",
  "strengths": ["Strategic thinking", "Calm under pressure", "Lifelong learner"],
  "weaknesses": ["Overthinking tendencies", "Difficulty saying no", "Emotional reservation"],
  "element_note": "Your chart shows strong Water and Metal elements supporting each other, with Wood as your favorable element to channel creative expression."
}`

	var content model.QuickContent
	if err := json.Unmarshal([]byte(fixedJSON), &content); err != nil {
		t.Fatalf("unmarshal QuickContent: %v", err)
	}

	if content.SummaryLine == "" {
		t.Error("summary_line empty")
	}
	if content.Personality == "" {
		t.Error("personality empty")
	}
	if len(content.Strengths) == 0 {
		t.Error("strengths empty")
	}
	if len(content.Weaknesses) == 0 {
		t.Error("weaknesses empty")
	}
	if content.ElementNote == "" {
		t.Error("element_note empty")
	}
}

func TestQuickPromptRoundTrip(t *testing.T) {
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

	userPrompt, err := BuildQuickUserPrompt("ja", chart)
	if err != nil {
		t.Fatalf("BuildQuickUserPrompt: %v", err)
	}

	fixedResp := `{"summary_line":"水のごとき柔軟な宿命","personality":"壬の日主は大海の如し","strengths":["適応力","知性"],"weaknesses":["優柔不断"],"element_note":"水金相生、木が喜神"}`

	mock := &mockLLM{fixedJSON: fixedResp}
	ctx := context.Background()

	raw, err := mock.GenerateJSON(ctx, QuickSystemPrompt, userPrompt)
	if err != nil {
		t.Fatalf("GenerateJSON: %v", err)
	}

	var content model.QuickContent
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if content.SummaryLine != "水のごとき柔軟な宿命" {
		t.Errorf("unexpected summary_line: %s", content.SummaryLine)
	}
	if len(content.Strengths) != 2 {
		t.Errorf("expected 2 strengths, got %d", len(content.Strengths))
	}

	reJSON, _ := json.Marshal(&content)
	if !json.Valid(reJSON) {
		t.Error("re-serialized QuickContent not valid JSON")
	}
}

func TestAllLocales(t *testing.T) {
	chart, err := bazi.Calculate(bazi.BirthInput{
		Gender:       0,
		CalendarType: 0,
		Year:         2000,
		Month:        1,
		Day:          1,
		Hour:         8,
	})
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	for _, loc := range []string{"en", "zh", "ja", "ko"} {
		prompt, err := BuildQuickUserPrompt(loc, chart)
		if err != nil {
			t.Errorf("locale %s: %v", loc, err)
			continue
		}
		if !strings.Contains(prompt, "locale: "+loc) {
			t.Errorf("locale %s not found in prompt", loc)
		}
	}
}

func TestPromptNoAI(t *testing.T) {
	lower := strings.ToLower(QuickSystemPrompt)
	if strings.Contains(lower, " ai ") || strings.Contains(lower, "ai,") || strings.Contains(lower, " ai.") {
		t.Error("system prompt may contain 'AI' — P2 violation")
	}
	if strings.Contains(lower, "artificial intelligence") {
		t.Error("system prompt contains 'artificial intelligence' — P2 violation")
	}

	chart, _ := bazi.Calculate(bazi.BirthInput{
		Gender: 1, CalendarType: 0, Year: 1990, Month: 8, Day: 15, Hour: 14, Minute: 30,
	})

	user, _ := BuildQuickUserPrompt("en", chart)
	if strings.Contains(strings.ToLower(user), " ai ") {
		t.Error("user prompt contains 'AI' — P2 violation")
	}
}
