package prompts

import (
	"strings"
	"testing"

	"fatelumen/backend/internal/model"
)

func TestChapterRegistryContract(t *testing.T) {
	chapters := ChapterDefinitions()
	if len(chapters) != 10 {
		t.Fatalf("expected 10 chapters, got %d", len(chapters))
	}
	for i, chapter := range chapters {
		if chapter.No != i+1 {
			t.Errorf("chapter %d has no %d", i, chapter.No)
		}
		if chapter.Key != expectedReportChapterKeys[i] {
			t.Errorf("chapter %d key = %s", i, chapter.Key)
		}
		if chapter.Name == "" || len(chapter.RequiredFacts) == 0 || len(chapter.DefaultFacts) == 0 {
			t.Errorf("chapter %s is incomplete", chapter.Key)
		}
	}
	chapters[0].Name = "changed"
	if current, _ := ChapterByKey("destiny_depth"); current.Name == "changed" {
		t.Fatal("registry leaked mutable slice")
	}
}

func TestLocaleRegistry(t *testing.T) {
	for _, locale := range []string{"zh", "en", "ja", "ko"} {
		spec, err := NormalizeLocale(locale)
		if err != nil || spec.Code != locale || spec.Instruction == "" {
			t.Fatalf("locale %s unavailable: %v", locale, err)
		}
	}
	if _, err := NormalizeLocale("fr"); err == nil {
		t.Fatal("unsupported locale accepted")
	}
}

func TestBuildChapterPromptPreviewFiltersFacts(t *testing.T) {
	facts := model.InterpretationFacts{FactsHash: "facts-123", Warnings: []string{"private warning"}, Versions: model.InterpretationFactVersions{FactsSchemaVersion: "facts-v3"}}
	preview, err := BuildChapterPromptPreview("ja", "wealth_depth", facts)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Locale.Code != "ja" || !strings.Contains(preview.UserPrompt, "locale: ja") {
		t.Fatal("locale not applied")
	}
	if preview.FactsHash != "facts-123" || preview.InputFacts["facts_hash"] != "facts-123" {
		t.Fatal("facts hash missing")
	}
	if _, ok := preview.InputFacts["warnings"]; ok {
		t.Fatal("non-whitelisted facts leaked")
	}
	if preview.Chapter.Key != "wealth_depth" || preview.OutputSchema == nil {
		t.Fatal("chapter contract missing")
	}
}
