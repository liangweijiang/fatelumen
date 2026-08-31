package prompts

import (
	"strings"
	"testing"

	"fatelumen/backend/internal/model"
)

func TestSemanticDigestRejectsUnknownRequiredStrength(t *testing.T) {
	_, err := buildSemanticDigest("destiny_depth", "zh", model.InterpretationFacts{DayMasterStrength: model.DayMasterStrengthFact{Level: "new_unmapped_level"}}, []string{"day_master_strength"})
	if err == nil {
		t.Fatal("unknown required strength level was accepted")
	}
}

func TestSemanticDigestSummarizesTenGodsWithoutMechanicalNoise(t *testing.T) {
	high, low := 35.0, 0.0
	facts := model.InterpretationFacts{TenGodStructure: []model.ScoredFact{
		{Code: "ten_god.peer.same", Level: "visible_and_rooted", Score: &high},
		{Code: "ten_god.officer.same", Level: "missing", Score: &low},
		{Code: "ten_god.category.peer", Level: "dominant", Score: &high},
	}}
	digest, err := buildSemanticDigest("ten_gods_full", "zh", facts, []string{"ten_god_structure"})
	if err != nil {
		t.Fatal(err)
	}
	text := digest.Text()
	for _, expected := range []string{"主要十神为同阴阳比肩", "原局不显同阴阳七杀"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("digest missing %q: %s", expected, text)
		}
	}
	for _, forbidden := range []string{"visible_and_rooted", "ten_god.category.peer", "已计算"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("digest leaked %q: %s", forbidden, text)
		}
	}
}

func TestSemanticDigestCropsLuckFactsByChapter(t *testing.T) {
	cycles := []model.LuckCycle{{GanZhi: "甲子", StartYear: 2020}, {GanZhi: "乙丑", StartYear: 2030}, {GanZhi: "丙寅", StartYear: 2040}}
	years := make([]model.AnnualFortune, 10)
	for i := range years {
		years[i] = model.AnnualFortune{Year: 2026 + i, GanZhi: "丙午"}
	}
	facts := model.InterpretationFacts{LuckCycles: cycles, AnnualFortunes: years}
	digest, err := buildSemanticDigest("destiny_depth", "zh", facts, []string{"luck_cycles", "annual_fortunes"})
	if err != nil {
		t.Fatal(err)
	}
	text := digest.Text()
	if strings.Contains(text, "丙寅") || strings.Contains(text, "2029年") {
		t.Fatalf("general chapter received excessive luck data: %s", text)
	}
	full, err := buildSemanticDigest("ten_year_years", "zh", facts, []string{"annual_fortunes"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full.Text(), "2035年") {
		t.Fatalf("ten-year chapter did not retain ten years: %s", full.Text())
	}
}

func TestSemanticDigestCoverageReportsMissingSelectedFacts(t *testing.T) {
	digest, err := buildSemanticDigest("health_depth", "zh", model.InterpretationFacts{}, []string{"chart", "climate", "disease"})
	if err != nil {
		t.Fatalf("buildSemanticDigest: %v", err)
	}
	if digest.Coverage.SelectedFacts != 3 || digest.Coverage.CoveredFacts != 1 || digest.Coverage.UnavailableFacts != 2 {
		t.Fatalf("unexpected coverage: %+v", digest.Coverage)
	}
	if digest.Coverage.Ready {
		t.Fatal("coverage with missing facts must not be ready")
	}
}
