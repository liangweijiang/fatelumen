package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/llm/prompts"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type routingTestProvider struct {
	name string
	raw  string
	err  error
}

func (p *routingTestProvider) Name() string { return p.name }
func (p *routingTestProvider) GenerateJSON(context.Context, string, string, ...llm.Option) (string, error) {
	return p.raw, p.err
}

func TestRunChapterExhaustsPrimaryThenUsesFrozenBackupRoute(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.FullReport{}, &model.FullReportChapter{}, &model.FullReportChapterPayload{}, &model.FullReportAttempt{}, &model.FullReportAttemptPayload{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	report := model.FullReport{PublicID: "01JROUTETEST00000000000000", UserID: 1, Locale: "zh", PayMethod: "credits", Status: model.FullReportStatusGenerating, CurrentStage: model.FullReportStatusGenerating, ChapterTotal: 10, ProviderChainKey: "frozen", ChapterConcurrency: 1, FactsHash: strings.Repeat("a", 64), RetentionPolicy: "30d", ExpiresAt: now.Add(30 * 24 * time.Hour), CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&report).Error; err != nil {
		t.Fatal(err)
	}
	definition := prompts.ChapterDefinitions()[0]
	chapter := model.FullReportChapter{ReportID: report.ID, ChapterNo: uint8(definition.No), ChapterKey: definition.Key, Title: definition.Name, Status: model.FullReportChapterStatusPending, PromptHash: strings.Repeat("b", 64), ValidationStatus: model.FullReportValidationStatusPending, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&chapter).Error; err != nil {
		t.Fatal(err)
	}
	payload := model.FullReportChapterPayload{ChapterID: chapter.ID, SemanticDigest: "digest", LanguageInstruction: "zh", TerminologySnapshot: model.JSONRaw(`[]`), FinalPrompt: "frozen prompt", OutputSchema: model.JSONRaw(`{}`), CreatedAt: now}
	if err := db.Create(&payload).Error; err != nil {
		t.Fatal(err)
	}
	modules := make([]string, len(definition.Sections))
	for i, section := range definition.Sections {
		modules[i] = fmt.Sprintf(`{"序号":%d,"名称":%q,"正文":%q}`, i+1, section.Name, fmt.Sprintf("第%d部分依据已提供资料作出完整、审慎且清楚的分析说明。", i+1))
	}
	validRaw := fmt.Sprintf(`{"章节":%q,"模块":[%s]}`, definition.Name, strings.Join(modules, ","))
	routes := []ResolvedFullReportRoute{
		{Frozen: FullReportModelRoute{RouteNo: 1, ProviderCode: "primary", Model: "primary-model", MaxRetries: 0, MaxAttempts: 1, TimeoutSeconds: 5, Temperature: 0.5}, Provider: &routingTestProvider{name: "primary", err: errors.New("temporary upstream failure")}},
		{Frozen: FullReportModelRoute{RouteNo: 2, ProviderCode: "custom", Model: "auto", MaxRetries: 0, MaxAttempts: 1, TimeoutSeconds: 5, Temperature: 0.5}, Provider: &routingTestProvider{name: "custom", raw: validRaw}},
	}
	executor := &fullReportExecutor{reports: repository.NewFullReportRepo(db)}
	row := repository.FullReportChapterWithPayload{Chapter: chapter, Payload: payload}
	if err := executor.runChapter(context.Background(), report.ID, "zh", model.InterpretationFacts{}, row, routes); err != nil {
		t.Fatal(err)
	}
	var attempts []model.FullReportAttempt
	if err := db.Order("attempt_no ASC").Find(&attempts).Error; err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 2 || attempts[0].RouteNo != 1 || attempts[0].Status != model.FullReportAttemptStatusFailed || attempts[1].RouteNo != 2 || attempts[1].Model != "auto" || attempts[1].Status != model.FullReportAttemptStatusSucceeded {
		t.Fatalf("unexpected route attempts: %+v", attempts)
	}
	var finished model.FullReportChapter
	if err := db.First(&finished, chapter.ID).Error; err != nil {
		t.Fatal(err)
	}
	if finished.Status != model.FullReportChapterStatusSucceeded || finished.SelectedAttemptID == nil || *finished.SelectedAttemptID != attempts[1].ID {
		t.Fatalf("backup attempt was not selected: %+v", finished)
	}
}
