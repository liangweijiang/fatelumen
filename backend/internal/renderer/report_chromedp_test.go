//go:build chromedp

package renderer

import (
	"context"
	"os"
	"testing"

	"fatelumen/backend/internal/bazi"
	"fatelumen/backend/internal/model"
)

func TestRenderReportPDF_Chromedp(t *testing.T) {
	chromiumPath := os.Getenv("CHROMIUM_PATH")

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

	content := model.ReportContent{
		Locale:      "en",
		SummaryLine: "A bright Wood rising at dawn.",
		Summary:     "Your chart reveals a dynamic balance of Wood and Fire elements, creating a personality that is both ambitious and charismatic. The day master sits on a strong foundation.",
		Personality: "Your Yang Wood day master gives you a natural leadership quality — you are the tall tree in the forest. Combined with Fire in the month branch, your creative energy is formidable.",
		Career:      "The presence of Earth in your hour pillar suggests strong wealth potential through steady effort.",
		Relationship: "Your day branch reveals a harmonious Peach Blossom configuration, suggesting balanced romantic relationships.",
		Health:      "The strong Wood element relates to liver and detoxification functions in traditional theory. This is a general wellness observation.",
		YearlyFortune: []model.YearlyFortuneItem{
			{Year: 2026, Note: "Fire element supports your Wood day master, bringing career visibility."},
			{Year: 2027, Note: "Earth stabilization — focus on consolidating achievements."},
			{Year: 2028, Note: "Metal year activates your wealth star. Strategic decisions ahead."},
		},
		Suggestions: []string{
			"Channel Wood energy into daily creative practice.",
			"Cultivate patience during Earth-dominant months.",
			"Build a network of Metal-element mentors.",
			"Practice mindfulness to balance Fire.",
			"Incorporate Water activities for harmony.",
		},
	}

	data := BuildReportPDFData(chart, content, "2026-06-11")
	renderer := NewChromedpRenderer(chromiumPath)
	pdf, err := RenderReportPDF(context.Background(), renderer, data)
	if err != nil {
		t.Fatalf("RenderReportPDF: %v", err)
	}
	if len(pdf) == 0 {
		t.Error("PDF is empty")
	}
	if len(pdf) < 500 {
		t.Errorf("PDF too small: %d bytes", len(pdf))
	}

	os.WriteFile("/tmp/report_test.pdf", pdf, 0644)
	t.Logf("PDF saved to /tmp/report_test.pdf (%d bytes)", len(pdf))
}
