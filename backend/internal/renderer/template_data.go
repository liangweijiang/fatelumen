package renderer

import "fatelumen/backend/internal/model"

var elementColors = map[string]string{
	"木": "#5c7060",
	"火": "#b8473e",
	"土": "#a8851a",
	"金": "#9a8f7a",
	"水": "#3f5a6b",
}

var localeLabels = map[string]map[string]string{
	"en": {"personality": "Personality", "element_note": "Element Note", "destiny": "Destiny"},
	"zh": {"personality": "性格", "element_note": "五行提示", "destiny": "命格"},
	"ja": {"personality": "性格", "element_note": "五行の気", "destiny": "運命"},
	"ko": {"personality": "성격", "element_note": "오행", "destiny": "운명"},
}

type QuickImageData struct {
	Brand          string
	DayMasterLabel string
	GenDate        string
	Locale         string
	Content        model.QuickContent
	Pillars        []PillarDisplay
	T              map[string]string
}

type PillarDisplay struct {
	PositionLabel string
	Stem          string
	Branch        string
	ElementColor  string
}

var pillarLabels = map[string]map[string]string{
	"en": {"year": "Year", "month": "Month", "day": "Day", "hour": "Hour"},
	"zh": {"year": "年柱", "month": "月柱", "day": "日柱", "hour": "时柱"},
	"ja": {"year": "年柱", "month": "月柱", "day": "日柱", "hour": "時柱"},
	"ko": {"year": "년주", "month": "월주", "day": "일주", "hour": "시주"},
}

func BuildQuickImageData(content model.QuickContent, chart *model.ChartData, locale, genDate string) *QuickImageData {
	t, ok := localeLabels[locale]
	if !ok {
		t = localeLabels["en"]
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

	return &QuickImageData{
		Brand:          "FateLumen",
		DayMasterLabel: chart.DayMaster.Stem + " · " + t["destiny"],
		GenDate:        genDate,
		Locale:         locale,
		Content:        content,
		Pillars:        displays,
		T:              t,
	}
}
