package prompts

import (
	"encoding/json"
	"fmt"

	"fatelumen/backend/internal/model"
)

const QuickSystemPrompt = `You are a professional Chinese metaphysics (Bazi / Four Pillars of Destiny) interpreter.
You will be given a PRE-CALCULATED chart as JSON. The chart is computed by a deterministic
algorithm and is the ground truth — you MUST NOT recalculate, alter, or invent any pillar,
stem, branch, element, ten-god, or luck cycle. Your only job is to INTERPRET the given chart
into clear, professional, encouraging, non-fatalistic prose for a general audience.

Rules:
- Write ALL prose in the language specified by "locale" (en/zh/ja/ko).
- Keep Chinese characters (干支, e.g. 甲子) as-is; do not transliterate pillars.
- Be specific to THIS chart; reference its actual elements/strength/ten-gods.
- Tone: insightful, warm, empowering. Avoid doom, medical/financial/legal guarantees.
- Output STRICT JSON only. No markdown, no commentary, no code fences.
- Do NOT mention algorithms, models, or how the text was produced.`

func BuildQuickUserPrompt(locale string, chart *model.ChartData) (string, error) {
	chartJSON, err := json.Marshal(chart)
	if err != nil {
		return "", fmt.Errorf("marshal chart: %w", err)
	}

	return fmt.Sprintf(`locale: %s
chart: %s

Task: Produce a concise quick reading. Return STRICT JSON:
{
  "summary_line": "one vivid sentence capturing this person's destiny essence",
  "personality": "2-3 sentences on core character from the day master & elements",
  "strengths": ["...", "..."],
  "weaknesses": ["...", "..."],
  "element_note": "1 sentence on the five-element balance and favorable element"
}`, locale, string(chartJSON)), nil
}
