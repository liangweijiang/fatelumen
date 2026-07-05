package service

import (
	"testing"

	"fatelumen/backend/internal/model"
)

func TestMergeReportChapters_PrefersDetailedTenYearNotes(t *testing.T) {
	chapters := []model.Chapter{
		{
			No:    4,
			Key:   "ten_year_years",
			Title: "Ten Years",
			Body:  "short body",
			Years: []model.YearNote{
				{Year: 2026, GanZhi: "丙午", Note: "short"},
				{Year: 2027, GanZhi: "丁未", Note: "short"},
			},
		},
		{
			No:    4,
			Key:   "ten_year_years",
			Title: "",
			Body:  "this is a much longer ten-year overview body",
			Years: []model.YearNote{
				{Year: 2026, GanZhi: "丙午", Note: "this is the longer and more useful yearly note"},
			},
		},
	}

	merged := mergeReportChapters(chapters)
	if len(merged) != 1 {
		t.Fatalf("expected one merged chapter, got %d", len(merged))
	}
	if merged[0].Body != "this is a much longer ten-year overview body" {
		t.Fatalf("expected longer body, got %q", merged[0].Body)
	}
	if len(merged[0].Years) != 2 {
		t.Fatalf("expected two unique years, got %d", len(merged[0].Years))
	}
	if merged[0].Years[0].Year != 2026 || merged[0].Years[0].Note != "this is the longer and more useful yearly note" {
		t.Fatalf("expected detailed 2026 note, got %+v", merged[0].Years[0])
	}
}
