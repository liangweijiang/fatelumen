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

func setupFullReportCleanupRepo(t *testing.T) (*FullReportCleanupJobRepo, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.FullReport{}, &model.FullReportCleanupJob{}); err != nil {
		t.Fatal(err)
	}
	return NewFullReportCleanupJobRepo(db), db
}

func cleanupTestReport(index int, status string, expiresAt, now time.Time) *model.FullReport {
	return &model.FullReport{
		PublicID: fmt.Sprintf("01K000000000000000000000%02d", index), UserID: uint64(index + 1), Locale: "zh",
		Status: status, CurrentStage: status, ChapterTotal: 10, ChapterConcurrency: 3,
		ProviderChainKey: "frozen", FactsHash: fmt.Sprintf("%064d", index+1), RetentionPolicy: "30d",
		ExpiresAt: expiresAt, CreatedAt: now.Add(time.Duration(index) * time.Second), UpdatedAt: now,
	}
}

func TestScheduleExpiredClaimsOnlyBoundedTerminalReports(t *testing.T) {
	repo, db := setupFullReportCleanupRepo(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	reports := []*model.FullReport{
		cleanupTestReport(1, model.FullReportStatusCompleted, now.Add(-3*time.Hour), now),
		cleanupTestReport(2, model.FullReportStatusFailed, now.Add(-2*time.Hour), now),
		cleanupTestReport(3, model.FullReportStatusCompleted, now.Add(-time.Hour), now),
		cleanupTestReport(4, model.FullReportStatusGenerating, now.Add(-4*time.Hour), now),
		cleanupTestReport(5, model.FullReportStatusCompleted, now.Add(time.Hour), now),
	}
	for _, report := range reports {
		if err := db.Create(report).Error; err != nil {
			t.Fatal(err)
		}
	}
	tasks, err := repo.ScheduleExpired(context.Background(), now, 2, 3, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 || tasks[0].ReportID != reports[0].ID || tasks[1].ReportID != reports[1].ID {
		t.Fatalf("unexpected scheduled tasks: %+v", tasks)
	}
	var deleting int64
	if err := db.Model(&model.FullReport{}).Where("status = ?", model.FullReportStatusDeleting).Count(&deleting).Error; err != nil {
		t.Fatal(err)
	}
	if deleting != 2 {
		t.Fatalf("deleting reports = %d, want 2", deleting)
	}
	fullReports := NewFullReportRepo(db)
	if _, err := fullReports.GetByID(context.Background(), reports[0].ID, reports[0].UserID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleting report remained visible to user: %v", err)
	}
	if _, err := fullReports.AdminGetByID(context.Background(), reports[0].ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleting report detail remained visible to admin: %v", err)
	}
	var generating model.FullReport
	if err := db.First(&generating, reports[3].ID).Error; err != nil || generating.Status != model.FullReportStatusGenerating {
		t.Fatalf("active report was changed: %+v err=%v", generating, err)
	}
	var future model.FullReport
	if err := db.First(&future, reports[4].ID).Error; err != nil || future.Status != model.FullReportStatusCompleted {
		t.Fatalf("unexpired report was changed: %+v err=%v", future, err)
	}
}

func TestCleanupJobClaimIsExclusive(t *testing.T) {
	repo, db := setupFullReportCleanupRepo(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	report := cleanupTestReport(1, model.FullReportStatusCompleted, now.Add(-time.Hour), now)
	if err := db.Create(report).Error; err != nil {
		t.Fatal(err)
	}
	tasks, err := repo.ScheduleExpired(context.Background(), now, 10, 3, now)
	if err != nil || len(tasks) != 1 {
		t.Fatalf("schedule failed: tasks=%+v err=%v", tasks, err)
	}
	claimed, err := repo.Claim(context.Background(), tasks[0].ID, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Status != model.FullReportCleanupJobStatusRunning || claimed.AttemptCount != 1 {
		t.Fatalf("unexpected claimed task: %+v", claimed)
	}
	if _, err := repo.Claim(context.Background(), tasks[0].ID, now.Add(2*time.Second)); !errors.Is(err, ErrFullReportCleanupJobNotClaimed) {
		t.Fatalf("second claim err = %v", err)
	}
}

func TestCleanupJobDeletesEveryReportLayerAndKeepsAuditTask(t *testing.T) {
	reports, db := setupFullReportRepo(t)
	if err := db.AutoMigrate(&model.FullReportCleanupJob{}, &model.Order{}, &model.PaymentEvent{}, &model.CreditLedger{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)
	graph := fullReportGraph(now)
	graph.Report.Status = model.FullReportStatusCompleted
	graph.Report.CurrentStage = model.FullReportStatusCompleted
	graph.Report.ExpiresAt = now.Add(-time.Hour)
	if err := reports.CreateGraph(ctx, graph); err != nil {
		t.Fatal(err)
	}

	attempt := model.FullReportAttempt{
		ReportID: graph.Report.ID, ChapterID: graph.Chapters[0].ID, AttemptNo: 1, RouteNo: 1,
		Provider: "test", Model: "test", Status: model.FullReportAttemptStatusSucceeded,
		ValidationStatus: model.FullReportValidationStatusPassed, PromptHash: graph.Chapters[0].PromptHash,
		TraceID: "cleanup-test", StartedAt: now, CreatedAt: now,
	}
	if err := db.Create(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.FullReportAttemptPayload{AttemptID: attempt.ID, RequestParameters: model.JSONRaw(`{}`), RequestPrompt: "prompt", CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReportChapter{}).Where("id = ?", graph.Chapters[0].ID).Update("selected_attempt_id", attempt.ID).Error; err != nil {
		t.Fatal(err)
	}
	validation := model.FullReportValidationRun{ReportID: graph.Report.ID, RoundNo: 1, ValidatorVersion: "v1", Status: model.FullReportValidationStatusPassed, StartedAt: now, CreatedAt: now}
	if err := db.Create(&validation).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.FullReportValidationPayload{ValidationRunID: validation.ID, ValidationResult: model.JSONRaw(`{"passed":true}`), CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.FullReportResult{ReportID: graph.Report.ID, Locale: "zh", Content: model.ReportContent{}, ContentHash: fmt.Sprintf("%064d", 99), RenderVersion: "pdf-v2", PDFStorageKey: "reports/test.pdf", CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.FullReportRenderJob{ReportID: graph.Report.ID, RenderVersion: "pdf-v2", Status: model.FullReportRenderJobStatusSucceeded, MaxAttempts: 3, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	order := model.Order{
		UserID: graph.Report.UserID, ReportID: graph.Report.ID, Type: "report", SKU: "full-report",
		AmountCents: 1999, Currency: "usd", Provider: "test", Status: model.OrderStatusPaid,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.PaymentEvent{Provider: "test", EventID: "cleanup-payment-event", EventType: "paid", OrderID: &order.ID, ProcessedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.CreditLedger{UserID: graph.Report.UserID, Delta: -1, BalanceAfter: 9, Reason: "full_report", RefID: &graph.Report.ID, CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}

	cleanup := NewFullReportCleanupJobRepo(db)
	tasks, err := cleanup.ScheduleExpired(ctx, now, 10, 3, now)
	if err != nil || len(tasks) != 1 {
		t.Fatalf("schedule failed: tasks=%+v err=%v", tasks, err)
	}
	task, err := cleanup.Claim(ctx, tasks[0].ID, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := cleanup.AdvanceObjectDeleted(ctx, task.ID, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	steps := [][2]string{
		{model.FullReportCleanupStageObjectDeleted, model.FullReportCleanupStageAttemptsGone},
		{model.FullReportCleanupStageAttemptsGone, model.FullReportCleanupStageValidationsGone},
		{model.FullReportCleanupStageValidationsGone, model.FullReportCleanupStageChaptersGone},
		{model.FullReportCleanupStageChaptersGone, model.FullReportCleanupStageExecutionGone},
		{model.FullReportCleanupStageExecutionGone, model.FullReportCleanupStageResultGone},
		{model.FullReportCleanupStageResultGone, model.FullReportCleanupStageRenderJobsGone},
		{model.FullReportCleanupStageRenderJobsGone, model.FullReportCleanupStageReportGone},
	}
	for i, step := range steps {
		if err := cleanup.DeleteDatabaseStage(ctx, task.ID, step[0], step[1], now.Add(time.Duration(i+3)*time.Second)); err != nil {
			t.Fatalf("stage %s failed: %v", step[1], err)
		}
	}
	if err := cleanup.Succeed(ctx, task.ID, now.Add(20*time.Second)); err != nil {
		t.Fatal(err)
	}

	for _, table := range []any{
		&model.FullReport{}, &model.FullReportExecutionSnapshot{}, &model.FullReportExecutionPayload{},
		&model.FullReportChapter{}, &model.FullReportChapterPayload{}, &model.FullReportAttempt{},
		&model.FullReportAttemptPayload{}, &model.FullReportValidationRun{}, &model.FullReportValidationPayload{},
		&model.FullReportResult{}, &model.FullReportRenderJob{},
	} {
		var count int64
		if err := db.Model(table).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%T rows remain: %d", table, count)
		}
	}
	stored, err := cleanup.Get(ctx, task.ID)
	if err != nil || stored.Status != model.FullReportCleanupJobStatusSucceeded || stored.Stage != model.FullReportCleanupStageReportGone {
		t.Fatalf("cleanup audit task not retained: %+v err=%v", stored, err)
	}
	if stored.FinishedAt == nil {
		t.Fatal("cleanup audit task is missing deletion time")
	}
	var keptOrder model.Order
	if err := db.First(&keptOrder, order.ID).Error; err != nil || keptOrder.ReportID != graph.Report.ID || keptOrder.Status != model.OrderStatusPaid {
		t.Fatalf("financial order was changed: %+v err=%v", keptOrder, err)
	}
	var paymentCount, ledgerCount int64
	if err := db.Model(&model.PaymentEvent{}).Where("order_id = ?", order.ID).Count(&paymentCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.CreditLedger{}).Where("ref_id = ?", graph.Report.ID).Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if paymentCount != 1 || ledgerCount != 1 {
		t.Fatalf("financial records were deleted: payment_events=%d credit_ledgers=%d", paymentCount, ledgerCount)
	}
}

func TestCleanupJobFailureAndStaleRecoveryPreserveStage(t *testing.T) {
	repo, db := setupFullReportCleanupRepo(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	report := cleanupTestReport(1, model.FullReportStatusCompleted, now.Add(-time.Hour), now)
	if err := db.Create(report).Error; err != nil {
		t.Fatal(err)
	}
	tasks, err := repo.ScheduleExpired(context.Background(), now, 10, 3, now)
	if err != nil || len(tasks) != 1 {
		t.Fatalf("schedule failed: tasks=%+v err=%v", tasks, err)
	}
	claimed, err := repo.Claim(context.Background(), tasks[0].ID, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AdvanceObjectDeleted(context.Background(), claimed.ID, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	terminal, err := repo.RecordFailure(context.Background(), claimed.ID, "delete_failed", "temporary", now.Add(3*time.Second))
	if err != nil || terminal {
		t.Fatalf("first failure should requeue: terminal=%v err=%v", terminal, err)
	}
	reclaimed, err := repo.Claim(context.Background(), claimed.ID, now.Add(4*time.Second))
	if err != nil || reclaimed.Stage != model.FullReportCleanupStageObjectDeleted || reclaimed.AttemptCount != 2 {
		t.Fatalf("stage was not preserved: %+v err=%v", reclaimed, err)
	}
	requeued, failed, err := repo.RecoverStale(context.Background(), now.Add(5*time.Second), now.Add(6*time.Second))
	if err != nil || requeued != 1 || failed != 0 {
		t.Fatalf("unexpected recovery: requeued=%d failed=%d err=%v", requeued, failed, err)
	}
	stored, err := repo.Get(context.Background(), claimed.ID)
	if err != nil || stored.Status != model.FullReportCleanupJobStatusQueued || stored.Stage != model.FullReportCleanupStageObjectDeleted {
		t.Fatalf("unexpected recovered task: %+v err=%v", stored, err)
	}
}
