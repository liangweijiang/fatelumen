package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"fatelumen/backend/internal/llm/prompts"
)

func validPreflightPlans() []frozenChapterPlan {
	definitions := prompts.ChapterDefinitions()
	plans := make([]frozenChapterPlan, len(definitions))
	for i, definition := range definitions {
		facts := make(map[string]any, len(definition.RequiredFacts))
		for _, key := range definition.RequiredFacts {
			facts[key] = map[string]any{"present": true}
		}
		plans[i] = frozenChapterPlan{
			Definition: definition,
			Preview: &prompts.ChapterPromptPreview{
				ChapterInstruction: "chapter instruction", AdditiveInstruction: "locale instruction",
				CompleteInstruction: "complete instruction", InputFacts: facts, OutputSchema: map[string]any{"type": "object"},
			},
			PromptHash: strings.Repeat("a", 64),
		}
	}
	return plans
}

func TestPreflightFullReportPassesCompleteTenChapterPlan(t *testing.T) {
	runtime := FullReportRuntimeConfig{ChapterConcurrency: 3, MaxAttempts: 2, ChapterTimeout: time.Minute, Provider: "deepseek", Model: "deepseek-chat"}
	result := PreflightFullReport("zh", runtime, validPreflightPlans())
	if !result.Passed || len(result.Errors) != 0 || len(result.Chapters) != 10 {
		t.Fatalf("unexpected preflight result: %+v", result)
	}
}

func TestPreflightFullReportBlocksMissingRequiredFactAndRuntime(t *testing.T) {
	plans := validPreflightPlans()
	delete(plans[0].Preview.InputFacts, plans[0].Definition.RequiredFacts[0])
	result := PreflightFullReport("zh", FullReportRuntimeConfig{}, plans)
	if result.Passed || len(result.Errors) < 2 || result.Chapters[0].Passed {
		t.Fatalf("preflight should be blocked: %+v", result)
	}
}

func TestValidateChapterOutputStrictContract(t *testing.T) {
	definition := prompts.ChapterDefinitions()[0]
	modules := make([]map[string]any, len(definition.Sections))
	for i, section := range definition.Sections {
		modules[i] = map[string]any{"序号": i + 1, "名称": section.Name, "正文": strings.Repeat("内容完整且可读。", 5)}
	}
	raw, _ := json.Marshal(map[string]any{"章节": definition.Name, "模块": modules})
	result := validateChapterOutput(definition, "zh", string(raw), nil)
	if !result.Passed || !result.SchemaValid {
		t.Fatalf("valid output rejected: %+v", result)
	}
}

func TestValidateChapterOutputRejectsUnknownFieldAndMarkdown(t *testing.T) {
	definition := prompts.ChapterDefinitions()[0]
	unknown := `{"章节":"命格深析","模块":[],"额外":"不允许"}`
	if result := validateChapterOutput(definition, "zh", unknown, nil); result.Passed || result.SchemaValid {
		t.Fatalf("unknown field must be rejected: %+v", result)
	}
	if result := validateChapterOutput(definition, "zh", "```json\n{}\n```", nil); result.Passed {
		t.Fatalf("markdown envelope must be rejected: %+v", result)
	}
}
