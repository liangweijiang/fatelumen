//go:build chromedp

package renderer

import (
	"context"
	"os"
	"testing"

	"fatelumen/backend/internal/bazi"
	"fatelumen/backend/internal/model"
)

func TestChromedpScreenshot(t *testing.T) {
	chromiumPath := os.Getenv("CHROMIUM_PATH")
	renderer := NewChromedpRenderer(chromiumPath)

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
		Personality: "Born under the Ren (Yang Water) day master, you possess a deep, adaptable nature.",
		Strengths:   []string{"Strategic thinking", "Calm under pressure"},
		Weaknesses:  []string{"Overthinking", "Difficulty saying no"},
		ElementNote: "Strong Water and Metal support each other; Wood is your favorable element.",
	}

	data := BuildQuickImageData(content, chart, "en", "2025-06-10")
	html, err := RenderTemplate(data)
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}

	ctx := context.Background()
	png, err := renderer.Render(ctx, html, FormatPNG)
	if err != nil {
		t.Fatalf("Render PNG: %v (hint: run with chromium installed and fonts-noto-cjk)", err)
	}
	if len(png) == 0 {
		t.Error("screenshot is empty")
	}
	if len(png) < 1000 {
		t.Errorf("screenshot too small: %d bytes", len(png))
	}

	// 保存截图用于人工检查
	os.WriteFile("/tmp/quick_test.png", png, 0644)
	t.Logf("screenshot saved to /tmp/quick_test.png (%d bytes)", len(png))
}
