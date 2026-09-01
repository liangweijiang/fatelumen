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

func validAggregateRows(t *testing.T, facts model.InterpretationFacts) []repository.FullReportChapterWithPayload {
	t.Helper()
	definitions := prompts.ChapterDefinitions()
	rows := make([]repository.FullReportChapterWithPayload, 0, len(definitions))
	for _, definition := range definitions {
		modules := make([]chapterModuleOutput, 0, len(definition.Sections))
		for i, section := range definition.Sections {
			body := fmt.Sprintf("这是第%d章第%d个模块的完整分析内容，用于验证章节之间内容独立且没有遗漏。", definition.No, i+1)
			if definition.Key == "ten_year_years" && i == 0 {
				items := make([]string, 0, len(facts.AnnualFortunes))
				for _, annual := range facts.AnnualFortunes {
					items = append(items, fmt.Sprintf("%d年%s运势分析", annual.Year, annual.GanZhi))
				}
				body += strings.Join(items, "；")
			}
			modules = append(modules, chapterModuleOutput{No: i + 1, Name: section.Name, Content: body})
		}
		output := chapterOutput{Chapter: definition.Name, Modules: modules}
		raw, err := json.Marshal(output)
		if err != nil {
			t.Fatal(err)
		}
		selected := uint64(definition.No)
		rows = append(rows, repository.FullReportChapterWithPayload{
			Chapter: model.FullReportChapter{ChapterNo: uint8(definition.No), ChapterKey: definition.Key, Title: definition.Name, Status: model.FullReportChapterStatusSucceeded, SelectedAttemptID: &selected, SchemaValid: true, ValidationStatus: model.FullReportValidationStatusPassed},
			Payload: model.FullReportChapterPayload{FinalRawOutput: string(raw), FinalParsedOutput: raw, OutputSchema: model.JSONRaw(`{"type":"object"}`), TerminologySnapshot: model.JSONRaw(`[]`)},
		})
	}
	return rows
}

func aggregateFacts() model.InterpretationFacts {
	years := make([]model.AnnualFortune, 10)
	for i := range years {
		years[i] = model.AnnualFortune{Year: 2026 + i, GanZhi: []string{"丙午", "丁未", "戊申", "己酉", "庚戌", "辛亥", "壬子", "癸丑", "甲寅", "乙卯"}[i]}
	}
	return model.InterpretationFacts{Input: model.ReportInputSnapshot{Year: 1990}, AnnualFortunes: years}
}

func TestValidateFullReportPassesCompleteTenChapterSet(t *testing.T) {
	facts := aggregateFacts()
	result := validateFullReport("zh", facts, validAggregateRows(t, facts), time.Now().UTC())
	if !result.Passed || len(result.AffectedChapters) != 0 {
		t.Fatalf("unexpected validation: %+v", result)
	}
}

func TestValidateFullReportLocatesMissingAnnualFact(t *testing.T) {
	facts := aggregateFacts()
	rows := validAggregateRows(t, facts)
	var parsed chapterOutput
	if err := json.Unmarshal(rows[3].Payload.FinalParsedOutput, &parsed); err != nil {
		t.Fatal(err)
	}
	parsed.Modules[0].Content = strings.ReplaceAll(parsed.Modules[0].Content, "2035年乙卯运势分析", "")
	raw, _ := json.Marshal(parsed)
	rows[3].Payload.FinalRawOutput = string(raw)
	rows[3].Payload.FinalParsedOutput = raw
	result := validateFullReport("zh", facts, rows, time.Now().UTC())
	if result.Passed || len(result.AffectedChapters) != 1 || result.AffectedChapters[0] != 4 {
		t.Fatalf("expected chapter 4 failure, got %+v", result)
	}
}

func TestValidateFullReportRejectsDuplicateChapterIdentity(t *testing.T) {
	facts := aggregateFacts()
	rows := validAggregateRows(t, facts)
	rows[1].Chapter.ChapterNo = rows[0].Chapter.ChapterNo
	rows[1].Chapter.ChapterKey = rows[0].Chapter.ChapterKey
	result := validateFullReport("zh", facts, rows, time.Now().UTC())
	if result.Passed {
		t.Fatal("duplicate chapter identity should fail aggregate validation")
	}
}
