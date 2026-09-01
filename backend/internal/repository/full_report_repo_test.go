package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"fatelumen/backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupFullReportRepo(t *testing.T) (*FullReportRepo, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.FullReport{},
		&model.FullReportExecutionSnapshot{},
		&model.FullReportExecutionPayload{},
		&model.FullReportChapter{},
		&model.FullReportChapterPayload{},
		&model.FullReportAttempt{},
		&model.FullReportAttemptPayload{},
		&model.FullReportResult{},
	); err != nil {
		t.Fatal(err)
	}
	return NewFullReportRepo(db), db
}

func fullReportGraph(now time.Time) FullReportCreateGraph {
	report := &model.FullReport{
		PublicID:           "01J00000000000000000000000",
		UserID:             7,
		Locale:             "zh",
		Status:             model.FullReportStatusPending,
		CurrentStage:       model.FullReportStatusPending,
		ChapterTotal:       10,
		ProviderChainKey:   "default",
		ChapterConcurrency: 3,
		FactsHash:          fmt.Sprintf("%064d", 1),
		RetentionPolicy:    "30d",
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	snapshot := &model.FullReportExecutionSnapshot{
		ChartHash: fmt.Sprintf("%064d", 2), FactsHash: report.FactsHash, ExecutionHash: fmt.Sprintf("%064d", 3),
		InputSchemaVersion: "input-v1", ChartSchemaVersion: "chart-v1", FactsSchemaVersion: "facts-v1",
		RuleSetVersion: "rules-v1", PromptVersion: "prompt-v1", DictionaryVersion: "dictionary-v1",
		RuntimePolicyVersion: "runtime-v1", LocationDatabaseVersion: "location-v1", TimezoneDatabaseVersion: "timezone-v1",
		SolarAlgorithmVersion: "solar-v1", LunarGoVersion: "lunar-v1", FrozenAt: now, CreatedAt: now,
	}
	payload := &model.FullReportExecutionPayload{
		InputSnapshot: model.JSONRaw(`{}`), TimeCalculationSnapshot: model.JSONRaw(`{}`), ChartSnapshot: model.JSONRaw(`{}`),
		FactsSnapshot: model.JSONRaw(`{}`), PreflightResult: model.JSONRaw(`{"passed":true}`),
		ChapterPlanSnapshot: model.JSONRaw(`[]`), RuntimeConfigSnapshot: model.JSONRaw(`{}`), CreatedAt: now,
	}
	chapters := make([]model.FullReportChapter, 10)
	chapterPayloads := make([]model.FullReportChapterPayload, 10)
	for i := 0; i < 10; i++ {
		chapters[i] = model.FullReportChapter{
			ChapterNo: uint8(i + 1), ChapterKey: fmt.Sprintf("chapter_%02d", i+1), Title: fmt.Sprintf("第%d章", i+1),
			Status: model.FullReportChapterStatusPending, PromptHash: fmt.Sprintf("%064d", i+10),
			ValidationStatus: model.FullReportValidationStatusPending, CreatedAt: now, UpdatedAt: now,
		}
		chapterPayloads[i] = model.FullReportChapterPayload{
			SemanticDigest: "摘要", LanguageInstruction: "语言", TerminologySnapshot: model.JSONRaw(`[]`),
			FinalPrompt: "prompt", OutputSchema: model.JSONRaw(`{}`), CreatedAt: now,
		}
	}
	return FullReportCreateGraph{Report: report, Snapshot: snapshot, ExecutionPayload: payload, Chapters: chapters, ChapterPayloads: chapterPayloads}
}

func TestFullReportCreateGraphAndBatchLoadPayloads(t *testing.T) {
	repo, _ := setupFullReportRepo(t)
	graph := fullReportGraph(time.Now().UTC())
	if err := repo.CreateGraph(context.Background(), graph); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.ListChaptersWithPayload(context.Background(), graph.Report.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 10 {
		t.Fatalf("chapters = %d, want 10", len(rows))
	}
	for i, row := range rows {
		if int(row.Chapter.ChapterNo) != i+1 || row.Payload.ChapterID != row.Chapter.ID {
			t.Fatalf("chapter payload association broken at %d", i)
		}
	}
}

func TestFullReportBeginPreflightClaimsExactlyOnce(t *testing.T) {
	repo, _ := setupFullReportRepo(t)
	report := fullReportGraph(time.Now().UTC()).Report
	if err := repo.Create(context.Background(), report); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := repo.BeginPreflight(context.Background(), report.ID, now); err != nil {
		t.Fatalf("first claim failed: %v", err)
	}
	if err := repo.BeginPreflight(context.Background(), report.ID, now); !errors.Is(err, ErrFullReportImmutableWrite) {
		t.Fatalf("second claim should be rejected, got %v", err)
	}
	stored, err := repo.GetByID(context.Background(), report.ID, report.UserID)
	if err != nil || stored.Status != model.FullReportStatusPreflighting {
		t.Fatalf("unexpected claimed report: %+v err=%v", stored, err)
	}
}

func TestFullReportCreateGraphRequiresTenChapters(t *testing.T) {
	repo, _ := setupFullReportRepo(t)
	graph := fullReportGraph(time.Now().UTC())
	graph.Chapters = graph.Chapters[:9]
	graph.ChapterPayloads = graph.ChapterPayloads[:9]
	if err := repo.CreateGraph(context.Background(), graph); !errors.Is(err, ErrFullReportChapterCount) {
		t.Fatalf("err = %v, want chapter count error", err)
	}
}

func TestFullReportTerminalBlocksAttempts(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	graph := fullReportGraph(now)
	if err := repo.CreateGraph(context.Background(), graph); err != nil {
		t.Fatal(err)
	}
	chapterID := graph.Chapters[0].ID
	attempt := &model.FullReportAttempt{
		ReportID: graph.Report.ID, ChapterID: chapterID, AttemptNo: 1, RouteNo: 1,
		Provider: "mock", Model: "mock-v1", Status: model.FullReportAttemptStatusRunning,
		ValidationStatus: model.FullReportValidationStatusPending, PromptHash: fmt.Sprintf("%064d", 99),
		TraceID: "trace-1", StartedAt: now, CreatedAt: now,
	}
	payload := &model.FullReportAttemptPayload{RequestParameters: model.JSONRaw(`{}`), RequestPrompt: "prompt", CreatedAt: now}
	if err := repo.AppendAttempt(context.Background(), attempt, payload); err != nil {
		t.Fatal(err)
	}
	if err := repo.Fail(context.Background(), graph.Report.ID, "validation_failed", "failed", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	blocked := &model.FullReportAttempt{
		ReportID: graph.Report.ID, ChapterID: chapterID, AttemptNo: 2, RouteNo: 1,
		Provider: "mock", Model: "mock-v1", Status: model.FullReportAttemptStatusRunning,
		ValidationStatus: model.FullReportValidationStatusPending, PromptHash: fmt.Sprintf("%064d", 100),
		TraceID: "trace-2", StartedAt: now, CreatedAt: now,
	}
	if err := repo.AppendAttempt(context.Background(), blocked, payload); !errors.Is(err, ErrFullReportTerminal) {
		t.Fatalf("err = %v, want terminal error", err)
	}
	var attempts int64
	if err := db.Model(&model.FullReportAttempt{}).Where("report_id = ?", graph.Report.ID).Count(&attempts).Error; err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestFullReportCompleteRequiresTenValidatedChapters(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	graph := fullReportGraph(now)
	if err := repo.CreateGraph(context.Background(), graph); err != nil {
		t.Fatal(err)
	}
	result := &model.FullReportResult{
		Locale: "zh", Content: model.ReportContent{Locale: "zh"}, ContentHash: fmt.Sprintf("%064d", 200),
		RenderVersion: "render-v1", CreatedAt: now,
	}
	if err := repo.Complete(context.Background(), graph.Report.ID, result, now); !errors.Is(err, ErrFullReportNotReady) {
		t.Fatalf("err = %v, want not ready", err)
	}
	if err := db.Model(&model.FullReportChapter{}).Where("report_id = ?", graph.Report.ID).Updates(map[string]any{
		"status": model.FullReportChapterStatusSucceeded, "schema_valid": true, "validation_status": model.FullReportValidationStatusPassed,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.Complete(context.Background(), graph.Report.ID, result, now); err != nil {
		t.Fatal(err)
	}
	if err := repo.Complete(context.Background(), graph.Report.ID, result, now); !errors.Is(err, ErrFullReportTerminal) {
		t.Fatalf("second completion err = %v, want terminal", err)
	}
}
