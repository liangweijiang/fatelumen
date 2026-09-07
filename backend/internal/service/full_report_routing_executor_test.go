package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/llm/prompts"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type routingTestProvider struct {
	name  string
	raw   string
	err   error
	usage llm.GenerationUsage
}

type trackingFailureProvider struct {
	delay       time.Duration
	active      int32
	maxActive   int32
	callCount   int32
	respectDone bool
}

func (p *trackingFailureProvider) Name() string { return "tracking-failure" }
func (p *trackingFailureProvider) GenerateJSON(ctx context.Context, system, user string, opts ...llm.Option) (string, error) {
	result, err := p.GenerateJSONDetailed(ctx, system, user, opts...)
	return result.Content, err
}
func (p *trackingFailureProvider) GenerateJSONDetailed(ctx context.Context, _ string, _ string, _ ...llm.Option) (llm.GenerationResult, error) {
	atomic.AddInt32(&p.callCount, 1)
	active := atomic.AddInt32(&p.active, 1)
	defer atomic.AddInt32(&p.active, -1)
	for {
		maximum := atomic.LoadInt32(&p.maxActive)
		if active <= maximum || atomic.CompareAndSwapInt32(&p.maxActive, maximum, active) {
			break
		}
	}
	if p.respectDone {
		<-ctx.Done()
		return llm.GenerationResult{}, ctx.Err()
	}
	select {
	case <-time.After(p.delay):
		return llm.GenerationResult{}, errors.New("simulated upstream failure")
	case <-ctx.Done():
		return llm.GenerationResult{}, ctx.Err()
	}
}

func (p *routingTestProvider) GenerateJSONDetailed(context.Context, string, string, ...llm.Option) (llm.GenerationResult, error) {
	return llm.GenerationResult{Content: p.raw, Usage: p.usage}, p.err
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
		{Frozen: FullReportModelRoute{RouteNo: 2, ProviderCode: "custom", Model: "auto", MaxRetries: 0, MaxAttempts: 1, TimeoutSeconds: 5, Temperature: 0.5}, Provider: &routingTestProvider{name: "custom", raw: validRaw, usage: testUsage(100, 200, 300)}},
	}
	executor := &fullReportExecutor{reports: repository.NewFullReportRepo(db)}
	row := repository.FullReportChapterWithPayload{Chapter: chapter, Payload: payload}
	ctx := logger.WithTraceID(context.Background(), "trace-real-123")
	if err := executor.runChapter(ctx, report.ID, "zh", model.InterpretationFacts{}, row, routes); err != nil {
		t.Fatal(err)
	}
	var attempts []model.FullReportAttempt
	if err := db.Order("attempt_no ASC").Find(&attempts).Error; err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 2 || attempts[0].RouteNo != 1 || attempts[0].Status != model.FullReportAttemptStatusFailed || attempts[1].RouteNo != 2 || attempts[1].Model != "auto" || attempts[1].Status != model.FullReportAttemptStatusSucceeded {
		t.Fatalf("unexpected route attempts: %+v", attempts)
	}
	if attempts[0].PromptTokens != nil || attempts[0].TotalTokens != nil {
		t.Fatalf("missing provider usage must remain null: %+v", attempts[0])
	}
	if attempts[1].PromptTokens == nil || *attempts[1].PromptTokens != 100 || attempts[1].TotalTokens == nil || *attempts[1].TotalTokens != 300 {
		t.Fatalf("provider usage was not persisted: %+v", attempts[1])
	}
	if attempts[0].TraceID != "trace-real-123" || attempts[1].TraceID != "trace-real-123" {
		t.Fatalf("request trace was not propagated: %+v", attempts)
	}
	var finished model.FullReportChapter
	if err := db.First(&finished, chapter.ID).Error; err != nil {
		t.Fatal(err)
	}
	if finished.Status != model.FullReportChapterStatusSucceeded || finished.SelectedAttemptID == nil || *finished.SelectedAttemptID != attempts[1].ID {
		t.Fatalf("backup attempt was not selected: %+v", finished)
	}
}

func testUsage(prompt, completion, total int) llm.GenerationUsage {
	return llm.GenerationUsage{PromptTokens: &prompt, CompletionTokens: &completion, TotalTokens: &total}
}

func TestRunChaptersHonorsFrozenConcurrency(t *testing.T) {
	repo, db, report := setupRoutingExecutorStore(t)
	definitions := prompts.ChapterDefinitions()[:6]
	rows := make([]repository.FullReportChapterWithPayload, 0, len(definitions))
	now := time.Now().UTC()
	for _, definition := range definitions {
		chapter := model.FullReportChapter{ReportID: report.ID, ChapterNo: uint8(definition.No), ChapterKey: definition.Key, Title: definition.Name, Status: model.FullReportChapterStatusPending, PromptHash: strings.Repeat("b", 64), ValidationStatus: model.FullReportValidationStatusPending, CreatedAt: now, UpdatedAt: now}
		if err := db.Create(&chapter).Error; err != nil {
			t.Fatal(err)
		}
		payload := model.FullReportChapterPayload{ChapterID: chapter.ID, SemanticDigest: "digest", LanguageInstruction: "zh", TerminologySnapshot: model.JSONRaw(`[]`), FinalPrompt: "frozen prompt", OutputSchema: model.JSONRaw(`{}`), CreatedAt: now}
		if err := db.Create(&payload).Error; err != nil {
			t.Fatal(err)
		}
		rows = append(rows, repository.FullReportChapterWithPayload{Chapter: chapter, Payload: payload})
	}
	provider := &trackingFailureProvider{delay: 80 * time.Millisecond}
	routes := []ResolvedFullReportRoute{{Frozen: FullReportModelRoute{RouteNo: 1, ProviderCode: "failure", Model: "failure-v1", MaxAttempts: 1, TimeoutSeconds: 5}, Provider: provider}}
	executor := &fullReportExecutor{reports: repo}
	err := executor.runChapters(context.Background(), report.ID, "zh", model.InterpretationFacts{}, rows, FullReportRuntimeConfig{ChapterConcurrency: 2}, routes)
	if err == nil {
		t.Fatal("expected exhausted chapter errors")
	}
	if got := atomic.LoadInt32(&provider.callCount); got != int32(len(rows)) {
		t.Fatalf("expected %d calls, got %d", len(rows), got)
	}
	if got := atomic.LoadInt32(&provider.maxActive); got != 2 {
		t.Fatalf("expected frozen concurrency 2, got %d", got)
	}
}

func TestRunChapterTimesOutAndPersistsFailure(t *testing.T) {
	repo, db, report := setupRoutingExecutorStore(t)
	row := createRoutingChapter(t, db, report.ID, prompts.ChapterDefinitions()[0])
	provider := &trackingFailureProvider{respectDone: true}
	routes := []ResolvedFullReportRoute{{Frozen: FullReportModelRoute{RouteNo: 1, ProviderCode: "slow", Model: "slow-v1", MaxAttempts: 1, TimeoutSeconds: 1}, Provider: provider}}
	executor := &fullReportExecutor{reports: repo}
	started := time.Now()
	err := executor.runChapter(context.Background(), report.ID, "zh", model.InterpretationFacts{}, row, routes)
	if err == nil || !strings.Contains(err.Error(), "all frozen model routes exhausted") {
		t.Fatalf("expected route exhaustion after timeout, got %v", err)
	}
	if elapsed := time.Since(started); elapsed < 900*time.Millisecond || elapsed > 3*time.Second {
		t.Fatalf("unexpected timeout duration: %v", elapsed)
	}
	var attempt model.FullReportAttempt
	if err := db.First(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	if attempt.Status != model.FullReportAttemptStatusFailed || attempt.ErrorCode != "LLM_TIMEOUT" || attempt.FinishedAt == nil {
		t.Fatalf("timeout failure was not persisted correctly: %+v", attempt)
	}
}

func TestRunChapterExhaustsEveryFrozenRoute(t *testing.T) {
	repo, db, report := setupRoutingExecutorStore(t)
	row := createRoutingChapter(t, db, report.ID, prompts.ChapterDefinitions()[0])
	primary := &trackingFailureProvider{}
	backup := &trackingFailureProvider{}
	routes := []ResolvedFullReportRoute{
		{Frozen: FullReportModelRoute{RouteNo: 1, ProviderCode: "primary", Model: "primary-v1", MaxAttempts: 2, TimeoutSeconds: 5}, Provider: primary},
		{Frozen: FullReportModelRoute{RouteNo: 2, ProviderCode: "backup", Model: "backup-v1", MaxAttempts: 1, TimeoutSeconds: 5}, Provider: backup},
	}
	executor := &fullReportExecutor{reports: repo}
	err := executor.runChapter(context.Background(), report.ID, "zh", model.InterpretationFacts{}, row, routes)
	if err == nil || !strings.Contains(err.Error(), "after 3 attempts") {
		t.Fatalf("expected three-attempt exhaustion, got %v", err)
	}
	if atomic.LoadInt32(&primary.callCount) != 2 || atomic.LoadInt32(&backup.callCount) != 1 {
		t.Fatalf("unexpected route calls: primary=%d backup=%d", primary.callCount, backup.callCount)
	}
	var attempts []model.FullReportAttempt
	if err := db.Order("attempt_no ASC").Find(&attempts).Error; err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 3 || attempts[0].RouteNo != 1 || attempts[1].RouteNo != 1 || attempts[2].RouteNo != 2 {
		t.Fatalf("unexpected persisted attempt chain: %+v", attempts)
	}
}

func setupRoutingExecutorStore(t *testing.T) (*repository.FullReportRepo, *gorm.DB, model.FullReport) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.FullReport{}, &model.FullReportChapter{}, &model.FullReportChapterPayload{}, &model.FullReportAttempt{}, &model.FullReportAttemptPayload{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	report := model.FullReport{PublicID: "01JROUTETEST00000000000000", UserID: 1, Locale: "zh", PayMethod: "credits", Status: model.FullReportStatusGenerating, CurrentStage: model.FullReportStatusGenerating, ChapterTotal: 10, ProviderChainKey: "frozen", ChapterConcurrency: 2, FactsHash: strings.Repeat("a", 64), RetentionPolicy: "30d", ExpiresAt: now.Add(30 * 24 * time.Hour), CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&report).Error; err != nil {
		t.Fatal(err)
	}
	return repository.NewFullReportRepo(db), db, report
}

func createRoutingChapter(t *testing.T, db *gorm.DB, reportID uint64, definition prompts.ChapterDefinition) repository.FullReportChapterWithPayload {
	t.Helper()
	now := time.Now().UTC()
	chapter := model.FullReportChapter{ReportID: reportID, ChapterNo: uint8(definition.No), ChapterKey: definition.Key, Title: definition.Name, Status: model.FullReportChapterStatusPending, PromptHash: strings.Repeat("b", 64), ValidationStatus: model.FullReportValidationStatusPending, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&chapter).Error; err != nil {
		t.Fatal(err)
	}
	payload := model.FullReportChapterPayload{ChapterID: chapter.ID, SemanticDigest: "digest", LanguageInstruction: "zh", TerminologySnapshot: model.JSONRaw(`[]`), FinalPrompt: "frozen prompt", OutputSchema: model.JSONRaw(`{}`), CreatedAt: now}
	if err := db.Create(&payload).Error; err != nil {
		t.Fatal(err)
	}
	return repository.FullReportChapterWithPayload{Chapter: chapter, Payload: payload}
}
