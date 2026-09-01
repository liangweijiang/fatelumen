package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"fatelumen/backend/internal/llm/prompts"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/repository"
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
		modules[i] = map[string]any{"序号": i + 1, "名称": section.Name, "正文": fmt.Sprintf("第%d个模块依据命盘资料进行完整说明，内容清楚且与其他模块不同。", i+1)}
	}
	raw, _ := json.Marshal(map[string]any{"章节": definition.Name, "模块": modules})
	result := validateChapterOutput(definition, "zh", string(raw), nil)
	if !result.Passed || !result.SchemaValid {
		t.Fatalf("valid output rejected: %+v", result)
	}
}

func TestValidateChapterOutputRejectsWrongLanguageAndSafetyBoundary(t *testing.T) {
	definition := prompts.ChapterDefinitions()[0]
	modules := make([]map[string]any, len(definition.Sections))
	for i, section := range definition.Sections {
		content := fmt.Sprintf("第%d个模块使用中文解释冻结资料，并给出完整而审慎的说明。", i+1)
		if i == 0 {
			content += "保证收益。"
		}
		modules[i] = map[string]any{"序号": i + 1, "名称": section.Name, "正文": content}
	}
	raw, _ := json.Marshal(map[string]any{"章节": definition.Name, "模块": modules})
	result := validateChapterOutput(definition, "en", string(raw), nil)
	if result.Passed {
		t.Fatal("wrong language and guaranteed return must be rejected")
	}
	codes := map[string]bool{}
	for _, rule := range result.Rules {
		if !rule.Passed {
			codes[rule.Code] = true
		}
	}
	if !codes["LANG_TARGET"] || !codes["SAFE_PROFIT"] {
		t.Fatalf("missing expected validation rules: %+v", result.Rules)
	}
}

func TestValidateChapterOutputRejectsProtectedFactConflict(t *testing.T) {
	definition := prompts.ChapterDefinitions()[0]
	modules := make([]map[string]any, len(definition.Sections))
	for i, section := range definition.Sections {
		content := fmt.Sprintf("第%d个模块根据己巳命盘事实进行说明，内容完整且保持审慎表达。", i+1)
		if i == 0 {
			content += "日主是甲，命局为身弱，用神为水，重点年份是2099年。"
		}
		modules[i] = map[string]any{"序号": i + 1, "名称": section.Name, "正文": content}
	}
	raw, _ := json.Marshal(map[string]any{"章节": definition.Name, "模块": modules})
	facts := model.InterpretationFacts{
		Input:             model.ReportInputSnapshot{Year: 1990},
		Chart:             model.ChartSnapshot{Data: model.ChartData{Pillars: model.Pillars{Year: model.Pillar{Stem: "己", Branch: "巳"}}, DayMaster: model.DayMaster{Stem: "丙"}}},
		DayMasterStrength: model.DayMasterStrengthFact{Level: "slightly_strong"},
		UsefulGod:         &model.UsefulGodAnalysis{Primary: "火"},
		AnnualFortunes:    []model.AnnualFortune{{Year: 2026, GanZhi: "丙午"}},
	}
	result := validateChapterOutput(definition, "zh", string(raw), nil, chapterValidationFrozen{Facts: &facts})
	if result.Passed {
		t.Fatal("conflicting deterministic facts must be rejected")
	}
	codes := map[string]bool{}
	for _, rule := range result.Rules {
		if !rule.Passed {
			codes[rule.Code] = true
		}
	}
	for _, code := range []string{"FACT_DAY_MASTER", "FACT_STRENGTH", "FACT_USEFUL_GOD", "FACT_ANNUAL_YEAR"} {
		if !codes[code] {
			t.Fatalf("missing %s in rules: %+v", code, result.Rules)
		}
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

func TestAggregateRetryRowsUsesAffectedChaptersAndRemainingBudget(t *testing.T) {
	rows := []repository.FullReportChapterWithPayload{
		{Chapter: model.FullReportChapter{ChapterNo: 1, AttemptCount: 1}},
		{Chapter: model.FullReportChapter{ChapterNo: 4, AttemptCount: 2}},
		{Chapter: model.FullReportChapter{ChapterNo: 7, AttemptCount: 1}},
	}
	retry := aggregateRetryRows(rows, []uint8{1, 4}, 2)
	if len(retry) != 1 || retry[0].Chapter.ChapterNo != 1 {
		t.Fatalf("unexpected aggregate retry selection: %+v", retry)
	}
}
