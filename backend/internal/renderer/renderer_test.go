package renderer

import (
	"strings"
	"testing"

	"fatelumen/backend/internal/bazi"
	"fatelumen/backend/internal/model"
)

func TestRenderTemplate(t *testing.T) {
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

	content := model.QuickContent{
		SummaryLine: "A resilient Water personality flowing with quiet strength",
		Personality: "Born under the Ren (Yang Water) day master, you possess a deep, adaptable nature that navigates life with grace.",
		Strengths:   []string{"Strategic thinking", "Calm under pressure", "Lifelong learner"},
		Weaknesses:  []string{"Overthinking", "Difficulty saying no", "Emotional reservation"},
		ElementNote: "Strong Water and Metal support each other; Wood is your favorable element to channel creative expression.",
	}

	data := BuildQuickImageData(content, chart, "en", "2025-06-10")
	html, err := RenderTemplate(data)
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}

	checks := []string{
		"FateLumen",
		"Ren",
		content.SummaryLine,
		content.Personality,
		content.ElementNote,
		"fatelumen.com",
		"庚",
		"午",
		"甲",
		"申",
		"壬",
		"子",
		"丁",
		"未",
		"1080px",
		"1350px",
		"oklch(86% 0.039 80)",
		"oklch(56% 0.120 80)",
		"Personality",
		"Element Note",
	}

	for _, c := range checks {
		if !strings.Contains(html, c) {
			t.Errorf("HTML missing expected content: %q", c)
		}
	}
}

func TestRenderTemplateNoAI(t *testing.T) {
	chart, _ := bazi.Calculate(bazi.BirthInput{
		Gender: 1, CalendarType: 0, Year: 1990, Month: 8, Day: 15, Hour: 14, Minute: 30,
	})
	content := model.QuickContent{SummaryLine: "Test", Personality: "Test", ElementNote: "Test"}
	data := BuildQuickImageData(content, chart, "en", "2025-01-01")

	html, err := RenderTemplate(data)
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}
	if strings.Contains(strings.ToLower(html), " ai ") {
		t.Error("HTML contains 'AI' — P2 violation")
	}
}

func TestRenderTemplateAllLocales(t *testing.T) {
	chart, _ := bazi.Calculate(bazi.BirthInput{
		Gender: 0, CalendarType: 0, Year: 2000, Month: 1, Day: 1, Hour: 8,
	})
	content := model.QuickContent{
		SummaryLine: "Test",
		Personality: "Test",
		ElementNote: "Test",
	}
	for _, loc := range []string{"en", "zh", "ja", "ko"} {
		data := BuildQuickImageData(content, chart, loc, "2025-01-01")
		html, err := RenderTemplate(data)
		if err != nil {
			t.Errorf("locale %s: RenderTemplate: %v", loc, err)
			continue
		}
		if !strings.Contains(html, "FateLumen") {
			t.Errorf("locale %s: missing brand", loc)
		}
	}
}

func TestRenderTemplatePillarColors(t *testing.T) {
	chart, _ := bazi.Calculate(bazi.BirthInput{
		Gender: 1, CalendarType: 0, Year: 1990, Month: 8, Day: 15, Hour: 14, Minute: 30,
	})
	content := model.QuickContent{SummaryLine: "Test", Personality: "Test", ElementNote: "Test"}
	data := BuildQuickImageData(content, chart, "en", "2025-01-01")

	for _, p := range data.Pillars {
		if p.ElementColor == "" {
			t.Errorf("pillar %s has empty element color", p.PositionLabel)
		}
	}
}
