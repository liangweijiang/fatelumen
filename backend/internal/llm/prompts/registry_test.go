package prompts

import (
	"fmt"
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
		if chapter.Name == "" || len(chapter.RequiredFacts) == 0 || len(chapter.DefaultFacts) == 0 || len(chapter.Sections) == 0 || chapter.SourcePrompt == "" {
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

func TestChapterPromptsDoNotInterfereWithOutputLength(t *testing.T) {
	forbidden := []string{"max_tokens", "字数", "字符数", "篇幅", "Chinese characters", "1200-1800", "1000-1500"}
	facts := model.InterpretationFacts{FactsHash: "length-policy-check"}
	for _, locale := range []string{"zh", "en", "ja", "ko"} {
		for _, chapter := range ChapterDefinitions() {
			preview, err := BuildChapterPromptPreview(locale, chapter.Key, facts)
			if err != nil {
				t.Fatalf("build %s/%s prompt: %v", locale, chapter.Key, err)
			}
			prompt := preview.SystemPrompt + "\n" + preview.CompleteInstruction
			for _, token := range forbidden {
				if strings.Contains(prompt, token) {
					t.Fatalf("prompt %s/%s contains output-length control %q", locale, chapter.Key, token)
				}
			}
		}
	}
}

func TestBuildChapterPromptPreviewFiltersFacts(t *testing.T) {
	facts := model.InterpretationFacts{FactsHash: "facts-123", Warnings: []string{"private warning"}, Versions: model.InterpretationFactVersions{FactsSchemaVersion: "facts-v3"}}
	preview, err := BuildChapterPromptPreview("ja", "wealth_depth", facts)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Locale.Code != "ja" || !strings.Contains(preview.AdditiveInstruction, "日文") {
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

func TestBuildChapterPromptPreviewWithFactsEnforcesRequiredFacts(t *testing.T) {
	facts := model.InterpretationFacts{FactsHash: "facts-required"}
	preview, err := BuildChapterPromptPreviewWithFacts("zh", "destiny_depth", []string{"element_strength"}, facts)
	if err != nil {
		t.Fatalf("BuildChapterPromptPreviewWithFacts: %v", err)
	}
	for _, key := range []string{"chart", "day_master_strength", "element_strength"} {
		if _, ok := preview.InputFacts[key]; !ok {
			t.Fatalf("required or selected fact %q missing", key)
		}
	}
	if _, err := BuildChapterPromptPreviewWithFacts("zh", "destiny_depth", []string{"private.secret"}, facts); err == nil {
		t.Fatal("expected unsupported fact key to be rejected")
	}
}

func TestRuntimeChapterPromptExcludesExamplesAndTODOs(t *testing.T) {
	for _, chapter := range ChapterDefinitions() {
		prompt := runtimeChapterPrompt(chapter.SourcePrompt)
		if prompt == "" {
			t.Fatalf("chapter %s runtime prompt is empty", chapter.Key)
		}
		if strings.Contains(prompt, "示例输入：") || strings.Contains(prompt, "TODO：") {
			t.Fatalf("chapter %s leaked sample or TODO into runtime prompt", chapter.Key)
		}
	}
}

func TestOutputSchemaUsesFixedLightweightSections(t *testing.T) {
	preview, err := BuildChapterPromptPreview("zh", "destiny_depth", model.InterpretationFacts{FactsHash: "schema-test"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.UserPrompt, "模块数量、序号、名称和顺序必须") {
		t.Fatal("fixed module order was not stated")
	}
	if strings.Contains(preview.UserPrompt, "示例输入：") {
		t.Fatal("example input leaked into composed prompt")
	}
	properties := preview.OutputSchema["properties"].(map[string]any)
	sections := properties["模块"].(map[string]any)
	if sections["minItems"] != len(preview.Chapter.Sections) || sections["maxItems"] != len(preview.Chapter.Sections) {
		t.Fatal("schema does not lock the fixed section count")
	}
	items := sections["items"].(map[string]any)
	itemProperties := items["properties"].(map[string]any)
	if _, ok := itemProperties["事实引用"]; ok || strings.Contains(preview.UserPrompt, "事实引用") {
		t.Fatal("schema and prompt must not require model-declared fact references")
	}
}

func TestChinesePromptDoesNotLeakKnownInternalCodes(t *testing.T) {
	score := 13.08
	facts := model.InterpretationFacts{
		FactsHash:       "dictionary-test",
		ElementStrength: []model.ScoredFact{{Code: "element.power.wood", Level: "relatively_weak", Score: &score, Values: []string{"wood"}}},
		StemRelations:   []model.RelationFact{{Type: "combination", Symbols: []string{"甲", "己"}}},
	}
	preview, err := BuildChapterPromptPreviewWithFacts("zh", "destiny_depth", []string{"element_strength", "stem_relations"}, facts)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"element.power.wood", "relatively_weak", "combination", "【本次编排信息】", "【轻量输出约定】"} {
		if strings.Contains(preview.UserPrompt, forbidden) {
			t.Fatalf("internal code %q leaked into Chinese prompt", forbidden)
		}
	}
	for _, expected := range []string{"木力量偏弱", "甲、己天干五合", "\"章节\": \"命格深析\"", "\"名称\": \"日主强弱深层拆解\""} {
		if !strings.Contains(preview.UserPrompt, expected) {
			t.Fatalf("translated prompt missing %q", expected)
		}
	}
}

func TestAllChaptersEmbedFixedChineseJSONModules(t *testing.T) {
	for _, chapter := range ChapterDefinitions() {
		preview, err := BuildChapterPromptPreview("zh", chapter.Key, model.InterpretationFacts{FactsHash: "all-chapters"})
		if err != nil {
			t.Fatalf("chapter %s: %v", chapter.Key, err)
		}
		if strings.Contains(preview.UserPrompt, "【本次编排信息】") || strings.Contains(preview.UserPrompt, "【轻量输出约定】") {
			t.Fatalf("chapter %s contains obsolete metadata block", chapter.Key)
		}
		for index, section := range chapter.Sections {
			expected := fmt.Sprintf("{\"序号\": %d, \"名称\": \"%s\"", index+1, section.Name)
			if !strings.Contains(preview.UserPrompt, expected) {
				t.Fatalf("chapter %s missing fixed module %q", chapter.Key, expected)
			}
		}
	}
}

func TestPromptInjectsCompactFactsIntoAuthoredInputBlock(t *testing.T) {
	facts := model.InterpretationFacts{
		FactsHash: "compact-test",
		Input:     model.ReportInputSnapshot{Gender: 1},
		Chart: model.ChartSnapshot{Data: model.ChartData{
			Pillars: model.Pillars{
				Year: model.Pillar{Stem: "己", Branch: "巳"}, Month: model.Pillar{Stem: "丙", Branch: "子"},
				Day: model.Pillar{Stem: "丙", Branch: "寅"}, Hour: model.Pillar{Stem: "甲", Branch: "午"},
			},
			DayMaster: model.DayMaster{Stem: "丙", Element: "火", YinYang: "阳"},
		}},
		LuckCycles: []model.LuckCycle{{GanZhi: "乙亥", StartAge: 3, StartYear: 1993}},
	}
	preview, err := BuildChapterPromptPreviewWithFacts("zh", "destiny_depth", []string{"chart", "luck_cycles"}, facts)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"输入资料（系统已计算，直接引用）", "性别：乾·男", "八字：己巳 丙子 丙寅 甲午", "3岁起（1993年）乙亥"} {
		if !strings.Contains(preview.UserPrompt, expected) {
			t.Fatalf("composed prompt missing %q", expected)
		}
	}
	if strings.Contains(preview.UserPrompt, "后端计算好传入") || strings.Contains(preview.UserPrompt, "【系统已计算的前置事实】") {
		t.Fatal("placeholder or duplicated fact package leaked into composed prompt")
	}
}
