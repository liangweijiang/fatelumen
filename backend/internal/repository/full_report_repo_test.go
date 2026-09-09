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
		&model.User{},
		&model.CreditLedger{},
		&model.BirthProfile{},
		&model.FullReport{},
		&model.FullReportExecutionSnapshot{},
		&model.FullReportExecutionPayload{},
		&model.FullReportChapter{},
		&model.FullReportChapterPayload{},
		&model.FullReportAttempt{},
		&model.FullReportAttemptPayload{},
		&model.FullReportValidationRun{},
		&model.FullReportValidationPayload{},
		&model.FullReportResult{},
		&model.FullReportRenderJob{},
	); err != nil {
		t.Fatal(err)
	}
	return NewFullReportRepo(db), db
}

func TestCreateWithCreditChargeIsAtomic(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	user := model.User{ID: 101, Email: "credit@test.local", Credits: 20, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	report := fullReportGraph(now).Report
	report.PublicID = "01JCREDITCHARGE00000000000"
	report.UserID = user.ID
	if err := repo.CreateWithCreditCharge(context.Background(), report, 10); err != nil {
		t.Fatal(err)
	}
	if !report.Paid || report.PayMethod != "credit" || report.ID == 0 {
		t.Fatalf("report charge metadata = paid:%v method:%q id:%d", report.Paid, report.PayMethod, report.ID)
	}
	var storedUser model.User
	if err := db.First(&storedUser, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedUser.Credits != 10 {
		t.Fatalf("credits = %d, want 10", storedUser.Credits)
	}
	var ledger model.CreditLedger
	if err := db.Where("user_id = ? AND reason = ? AND ref_id = ?", user.ID, "consume_report", report.ID).First(&ledger).Error; err != nil {
		t.Fatal(err)
	}
	if ledger.Delta != -10 || ledger.BalanceAfter != 10 {
		t.Fatalf("ledger = %+v", ledger)
	}
}

func TestCreateWithCreditChargeRejectsInsufficientBalanceWithoutReport(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	user := model.User{ID: 102, Email: "poor@test.local", Credits: 9, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	report := fullReportGraph(now).Report
	report.PublicID = "01JINSUFFICIENT000000000000"
	report.UserID = user.ID
	if err := repo.CreateWithCreditCharge(context.Background(), report, 10); !errors.Is(err, ErrInsufficientCredits) {
		t.Fatalf("error = %v, want ErrInsufficientCredits", err)
	}
	var reportCount, ledgerCount int64
	if err := db.Model(&model.FullReport{}).Where("user_id = ?", user.ID).Count(&reportCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.CreditLedger{}).Where("user_id = ?", user.ID).Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if reportCount != 0 || ledgerCount != 0 {
		t.Fatalf("rolled back counts = reports:%d ledgers:%d", reportCount, ledgerCount)
	}
}

func TestCreateWithCreditChargeKeepsUnlimitedExemption(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	user := model.User{ID: 103, Email: "unlimited@test.local", Credits: 0, Unlimited: true, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	report := fullReportGraph(now).Report
	report.PublicID = "01JUNLIMITED0000000000000"
	report.UserID = user.ID
	if err := repo.CreateWithCreditCharge(context.Background(), report, 10); err != nil {
		t.Fatal(err)
	}
	if !report.Paid || report.PayMethod != "unlimited" {
		t.Fatalf("unlimited report = paid:%v method:%q", report.Paid, report.PayMethod)
	}
	var ledgerCount int64
	if err := db.Model(&model.CreditLedger{}).Where("user_id = ?", user.ID).Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if ledgerCount != 0 {
		t.Fatalf("unlimited user ledger count = %d, want 0", ledgerCount)
	}
}

func TestCancelPendingCreditChargeRefundsExactlyOnce(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	user := model.User{ID: 104, Email: "enqueue@test.local", Credits: 20, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	report := fullReportGraph(now).Report
	report.PublicID = "01JENQUEUEFAIL000000000000"
	report.UserID = user.ID
	if err := repo.CreateWithCreditCharge(context.Background(), report, 10); err != nil {
		t.Fatal(err)
	}
	failedAt := now.Add(time.Second)
	if err := repo.CancelPendingCreditCharge(context.Background(), report.ID, "enqueue_failed", "queue unavailable", failedAt); err != nil {
		t.Fatal(err)
	}
	if err := repo.CancelPendingCreditCharge(context.Background(), report.ID, "enqueue_failed", "queue unavailable", failedAt); err != nil {
		t.Fatalf("idempotent compensation failed: %v", err)
	}
	var storedUser model.User
	if err := db.First(&storedUser, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedUser.Credits != 20 {
		t.Fatalf("credits = %d, want restored 20", storedUser.Credits)
	}
	var refundCount int64
	if err := db.Model(&model.CreditLedger{}).Where("user_id = ? AND reason = ? AND ref_id = ?", user.ID, "refund_report_creation", report.ID).Count(&refundCount).Error; err != nil {
		t.Fatal(err)
	}
	if refundCount != 1 {
		t.Fatalf("refund ledger count = %d, want 1", refundCount)
	}
	var stored model.FullReport
	if err := db.First(&stored, report.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.FullReportStatusFailed || stored.Paid {
		t.Fatalf("report after compensation = status:%s paid:%v", stored.Status, stored.Paid)
	}
}

func TestFailRefundsConsumedReportCreditsExactlyOnce(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	user := model.User{ID: 105, Email: "terminal-failure@test.local", Credits: 20, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	report := fullReportGraph(now).Report
	report.PublicID = "01JTERMINALFAIL00000000000"
	report.UserID = user.ID
	if err := repo.CreateWithCreditCharge(context.Background(), report, 10); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReport{}).Where("id = ?", report.ID).Updates(map[string]any{
		"status": model.FullReportStatusGenerating, "current_stage": model.FullReportStatusGenerating,
	}).Error; err != nil {
		t.Fatal(err)
	}
	failedAt := now.Add(time.Minute)
	if err := repo.Fail(context.Background(), report.ID, "generation_failed", "routes exhausted", failedAt); err != nil {
		t.Fatal(err)
	}
	if err := repo.Fail(context.Background(), report.ID, "generation_failed", "routes exhausted", failedAt); !errors.Is(err, ErrFullReportTerminal) {
		t.Fatalf("second terminal failure = %v, want ErrFullReportTerminal", err)
	}
	var storedUser model.User
	if err := db.First(&storedUser, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedUser.Credits != 20 {
		t.Fatalf("credits = %d, want restored 20", storedUser.Credits)
	}
	var consumeCount, refundCount int64
	if err := db.Model(&model.CreditLedger{}).Where("user_id = ? AND reason = ? AND ref_id = ?", user.ID, "consume_report", report.ID).Count(&consumeCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.CreditLedger{}).Where("user_id = ? AND reason = ? AND ref_id = ?", user.ID, "refund_report", report.ID).Count(&refundCount).Error; err != nil {
		t.Fatal(err)
	}
	if consumeCount != 1 || refundCount != 1 {
		t.Fatalf("ledger counts = consume:%d refund:%d", consumeCount, refundCount)
	}
	var stored model.FullReport
	if err := db.First(&stored, report.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.FullReportStatusFailed || stored.Paid {
		t.Fatalf("failed report = status:%s paid:%v", stored.Status, stored.Paid)
	}
	settlement, err := repo.AdminCreditSettlement(context.Background(), &stored)
	if err != nil {
		t.Fatal(err)
	}
	if settlement.Status != "refunded" || settlement.Charged != 10 || settlement.Refunded != 10 || settlement.Net != 0 || len(settlement.Entries) != 2 {
		t.Fatalf("credit settlement = %+v", settlement)
	}
}

func TestFailDoesNotCreateRefundForUnlimitedReport(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	user := model.User{ID: 106, Email: "unlimited-failure@test.local", Credits: 0, Unlimited: true, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	report := fullReportGraph(now).Report
	report.PublicID = "01JUNLIMITEDFAIL00000000000"
	report.UserID = user.ID
	if err := repo.CreateWithCreditCharge(context.Background(), report, 10); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReport{}).Where("id = ?", report.ID).Updates(map[string]any{
		"status": model.FullReportStatusGenerating, "current_stage": model.FullReportStatusGenerating,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.Fail(context.Background(), report.ID, "generation_failed", "routes exhausted", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	var storedUser model.User
	if err := db.First(&storedUser, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedUser.Credits != 0 {
		t.Fatalf("credits = %d, want unchanged 0", storedUser.Credits)
	}
	var ledgerCount int64
	if err := db.Model(&model.CreditLedger{}).Where("user_id = ?", user.ID).Count(&ledgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if ledgerCount != 0 {
		t.Fatalf("unlimited user ledger count = %d, want 0", ledgerCount)
	}
	var stored model.FullReport
	if err := db.First(&stored, report.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.FullReportStatusFailed || !stored.Paid || stored.PayMethod != "unlimited" {
		t.Fatalf("unlimited failed report = status:%s paid:%v method:%q", stored.Status, stored.Paid, stored.PayMethod)
	}
	settlement, err := repo.AdminCreditSettlement(context.Background(), &stored)
	if err != nil {
		t.Fatal(err)
	}
	if settlement.Status != "exempt" || settlement.Charged != 0 || settlement.Refunded != 0 || len(settlement.Entries) != 0 {
		t.Fatalf("unlimited credit settlement = %+v", settlement)
	}
}

func TestPrepareAndCompleteRenderingRequiresStoredPDF(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	report := model.FullReport{PublicID: "01JPDFPIPELINE000000000000", UserID: 1, Locale: "zh", Status: model.FullReportStatusAssembling, CurrentStage: model.FullReportStatusAssembling, ChapterTotal: 10, ChapterSucceeded: 10, FactsHash: fmt.Sprintf("%064d", 1), RetentionPolicy: "days:30", ExpiresAt: now.Add(30 * 24 * time.Hour), CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&report).Error; err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 10; i++ {
		chapter := model.FullReportChapter{ReportID: report.ID, ChapterNo: uint8(i), ChapterKey: fmt.Sprintf("chapter_%d", i), Title: "章节", Status: model.FullReportChapterStatusSucceeded, PromptHash: fmt.Sprintf("%064d", i), SchemaValid: true, ValidationStatus: model.FullReportValidationStatusPassed, CreatedAt: now, UpdatedAt: now}
		if err := db.Create(&chapter).Error; err != nil {
			t.Fatal(err)
		}
	}
	finished := now
	validation := model.FullReportValidationRun{ReportID: report.ID, RoundNo: 1, ValidatorVersion: "v1", Status: model.FullReportValidationStatusPassed, StartedAt: now, FinishedAt: &finished, CreatedAt: now}
	if err := db.Create(&validation).Error; err != nil {
		t.Fatal(err)
	}
	result := &model.FullReportResult{Locale: "zh", ContentHash: fmt.Sprintf("%064d", 2)}
	task, err := repo.PrepareRendering(context.Background(), report.ID, result, "pdf-v2", 3, now)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != model.FullReportRenderJobStatusQueued {
		t.Fatalf("unexpected task: %+v", task)
	}
	stored, _ := repo.GetByInternalID(context.Background(), report.ID)
	if stored.Status != model.FullReportStatusRendering {
		t.Fatalf("expected rendering, got %s", stored.Status)
	}
	if err := repo.CompleteRendering(context.Background(), report.ID, "", "", "", now); !errors.Is(err, ErrFullReportNotReady) {
		t.Fatalf("expected missing PDF rejection, got %v", err)
	}
	if err := repo.CompleteRendering(context.Background(), report.ID, "reports/a.pdf", "https://example.test/a.pdf", fmt.Sprintf("%064d", 3), now); err != nil {
		t.Fatal(err)
	}
	stored, _ = repo.GetByInternalID(context.Background(), report.ID)
	if stored.Status != model.FullReportStatusCompleted {
		t.Fatalf("expected completed, got %s", stored.Status)
	}
}

func TestAdminListPageFiltersAndUsesStableCursor(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	base := time.Date(2026, time.September, 7, 5, 0, 0, 0, time.UTC)
	profile := model.BirthProfile{UserID: 7, DisplayName: "测试档案"}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		report := model.FullReport{
			PublicID: fmt.Sprintf("01J000000000000000000000%02d", i), UserID: 7, ProfileID: &profile.ID,
			Locale: "zh", Status: model.FullReportStatusCompleted, CurrentStage: model.FullReportStatusCompleted,
			ChapterTotal: 10, ChapterSucceeded: 10, FactsHash: fmt.Sprintf("%064d", i+1),
			RetentionPolicy: "30d", ExpiresAt: base.Add(30 * 24 * time.Hour), CreatedAt: base.Add(time.Duration(i) * time.Minute), UpdatedAt: base,
		}
		if err := db.Create(&report).Error; err != nil {
			t.Fatal(err)
		}
	}

	first, hasMore, err := repo.AdminListPage(context.Background(), AdminFullReportListFilter{UserID: 7, Locale: "zh", Status: model.FullReportStatusCompleted}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || !hasMore || first[0].CreatedAt.Before(first[1].CreatedAt) || first[0].ProfileName != profile.DisplayName {
		t.Fatalf("unexpected first page: rows=%+v has_more=%v", first, hasMore)
	}
	second, hasMore, err := repo.AdminListPage(context.Background(), AdminFullReportListFilter{
		UserID: 7, Locale: "zh", Status: model.FullReportStatusCompleted,
		Cursor: &FullReportCursor{CreatedAt: first[1].CreatedAt, ID: first[1].ID},
	}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || hasMore || second[0].ID == first[0].ID || second[0].ID == first[1].ID {
		t.Fatalf("unexpected second page: rows=%+v has_more=%v", second, hasMore)
	}
}

func TestAdminListAttemptsFiltersWithinSelectedChapter(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	graph := fullReportGraph(now)
	if err := repo.CreateGraph(context.Background(), graph); err != nil {
		t.Fatal(err)
	}
	for index, chapter := range graph.Chapters[:2] {
		for attemptNo := 1; attemptNo <= index+1; attemptNo++ {
			attempt := model.FullReportAttempt{
				ReportID: graph.Report.ID, ChapterID: chapter.ID, AttemptNo: uint16(attemptNo), RouteNo: 1,
				Provider: "mock", Model: "mock-v1", Status: model.FullReportAttemptStatusSucceeded,
				ValidationStatus: model.FullReportValidationStatusPassed, PromptHash: fmt.Sprintf("%064d", attemptNo),
				TraceID: fmt.Sprintf("trace-%d-%d", index, attemptNo), StartedAt: now.Add(time.Duration(attemptNo) * time.Second), CreatedAt: now,
			}
			if err := db.Create(&attempt).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	rows, total, err := repo.AdminListAttempts(context.Background(), graph.Report.ID, graph.Chapters[1].ID, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(rows) != 1 || rows[0].ChapterID != graph.Chapters[1].ID {
		t.Fatalf("unexpected chapter attempts: total=%d rows=%+v", total, rows)
	}
}

func TestAdminTraceProjectsExecutionSectionAndSelectedAttemptArtifacts(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	graph := fullReportGraph(now)
	graph.ExecutionPayload.InputSnapshot = model.JSONRaw(`{"name":"input-only"}`)
	graph.ChapterPayloads[0].FinalParsedOutput = model.JSONRaw(`{"legacy":true}`)
	graph.ChapterPayloads[0].ValidationResult = model.JSONRaw(`{"summary":"legacy"}`)
	if err := repo.CreateGraph(context.Background(), graph); err != nil {
		t.Fatal(err)
	}
	raw, err := repo.AdminGetExecutionSection(context.Background(), graph.Report.ID, "input_snapshot")
	if err != nil || string(raw) != `{"name":"input-only"}` {
		t.Fatalf("unexpected projected section: %s err=%v", raw, err)
	}
	if _, err := repo.AdminGetExecutionSection(context.Background(), graph.Report.ID, "unknown"); err == nil {
		t.Fatal("unsupported execution section should fail")
	}
	attempt := model.FullReportAttempt{ReportID: graph.Report.ID, ChapterID: graph.Chapters[0].ID, AttemptNo: 1, RouteNo: 1, Provider: "mock", Model: "mock-v1", Status: model.FullReportAttemptStatusSucceeded, SchemaValid: true, ValidationStatus: model.FullReportValidationStatusPassed, PromptHash: graph.Chapters[0].PromptHash, TraceID: "trace-artifact", StartedAt: now, CreatedAt: now}
	if err := db.Create(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	payload := model.FullReportAttemptPayload{AttemptID: attempt.ID, RequestParameters: model.JSONRaw(`{}`), RequestPrompt: "prompt", RawOutput: `{"raw":true}`, ParsedOutput: model.JSONRaw(`{"selected":true}`), ValidationResult: model.JSONRaw(`{"passed":true,"rules":[]}`), CreatedAt: now}
	if err := db.Create(&payload).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReportChapter{}).Where("id = ?", graph.Chapters[0].ID).Update("selected_attempt_id", attempt.ID).Error; err != nil {
		t.Fatal(err)
	}
	artifact, err := repo.AdminGetChapterArtifact(context.Background(), graph.Report.ID, graph.Chapters[0].ID, "content")
	if err != nil || string(artifact.Value.(model.JSONRaw)) != `{"selected":true}` {
		t.Fatalf("selected attempt content not preferred: %+v err=%v", artifact, err)
	}
	validation, err := repo.AdminGetAttemptValidation(context.Background(), graph.Report.ID, attempt.ID)
	if err != nil || string(validation) != `{"passed":true,"rules":[]}` {
		t.Fatalf("unexpected attempt validation: %s err=%v", validation, err)
	}
	if _, err := repo.AdminGetAttemptValidation(context.Background(), graph.Report.ID+1, attempt.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-report attempt should be hidden, got %v", err)
	}
	stats, err := repo.AdminAttemptStats(context.Background(), graph.Report.ID)
	if err != nil || len(stats) != 1 || stats[0].AttemptCount != 1 || stats[0].SucceededCount != 1 {
		t.Fatalf("unexpected attempt stats: %+v err=%v", stats, err)
	}
}

func TestAdminTraceEndpointsHideDeletingReport(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()
	graph := fullReportGraph(now)
	if err := repo.CreateGraph(ctx, graph); err != nil {
		t.Fatal(err)
	}
	attempt := model.FullReportAttempt{
		ReportID: graph.Report.ID, ChapterID: graph.Chapters[0].ID, AttemptNo: 1, RouteNo: 1,
		Provider: "mock", Model: "mock-v1", Status: model.FullReportAttemptStatusSucceeded,
		ValidationStatus: model.FullReportValidationStatusPassed, PromptHash: graph.Chapters[0].PromptHash,
		TraceID: "trace-deleting", StartedAt: now, CreatedAt: now,
	}
	if err := db.Create(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.FullReportAttemptPayload{AttemptID: attempt.ID, RequestParameters: model.JSONRaw(`{}`), RequestPrompt: "prompt", ParsedOutput: model.JSONRaw(`{}`), ValidationResult: model.JSONRaw(`{"passed":true}`), CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	validation := model.FullReportValidationRun{ReportID: graph.Report.ID, RoundNo: 1, ValidatorVersion: "v1", Status: model.FullReportValidationStatusPassed, StartedAt: now, CreatedAt: now}
	if err := db.Create(&validation).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.FullReportValidationPayload{ValidationRunID: validation.ID, ValidationResult: model.JSONRaw(`{"passed":true}`), CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.FullReportResult{ReportID: graph.Report.ID, Locale: "zh", Content: model.ReportContent{}, ContentHash: fmt.Sprintf("%064d", 88), RenderVersion: "pdf-v2", CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReport{}).Where("id = ?", graph.Report.ID).Updates(map[string]any{"status": model.FullReportStatusDeleting, "current_stage": model.FullReportStatusDeleting}).Error; err != nil {
		t.Fatal(err)
	}

	checks := []struct {
		name string
		call func() error
	}{
		{"result", func() error { _, err := repo.AdminGetResult(ctx, graph.Report.ID); return err }},
		{"execution", func() error { _, err := repo.AdminGetExecutionTrace(ctx, graph.Report.ID); return err }},
		{"facts", func() error {
			_, err := repo.AdminGetExecutionSection(ctx, graph.Report.ID, "facts_snapshot")
			return err
		}},
		{"attempt list", func() error {
			_, _, err := repo.AdminListAttempts(ctx, graph.Report.ID, graph.Chapters[0].ID, 20, 0)
			return err
		}},
		{"attempt", func() error { _, err := repo.AdminGetAttemptTrace(ctx, graph.Report.ID, attempt.ID); return err }},
		{"attempt validation", func() error { _, err := repo.AdminGetAttemptValidation(ctx, graph.Report.ID, attempt.ID); return err }},
		{"validation list", func() error { _, err := repo.AdminListValidationRuns(ctx, graph.Report.ID); return err }},
		{"validation", func() error { _, err := repo.AdminGetValidationTrace(ctx, graph.Report.ID, validation.ID); return err }},
		{"chapter list", func() error { _, err := repo.AdminListChapters(ctx, graph.Report.ID); return err }},
		{"chapter", func() error {
			_, err := repo.AdminGetChapterTrace(ctx, graph.Report.ID, graph.Chapters[0].ID)
			return err
		}},
		{"chapter artifact", func() error {
			_, err := repo.AdminGetChapterArtifact(ctx, graph.Report.ID, graph.Chapters[0].ID, "prompt")
			return err
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); !errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("deleting report remained readable: %v", err)
			}
		})
	}
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
	repo, db := setupFullReportRepo(t)
	graph := fullReportGraph(time.Now().UTC())
	if err := repo.CreateGraph(context.Background(), graph); err != nil {
		t.Fatal(err)
	}
	queryCount := 0
	if err := db.Callback().Query().Before("gorm:query").Register("test:count_chapter_payload_queries", func(*gorm.DB) {
		queryCount++
	}); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.ListChaptersWithPayload(context.Background(), graph.Report.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 10 {
		t.Fatalf("chapters = %d, want 10", len(rows))
	}
	if queryCount != 2 {
		t.Fatalf("chapter payload loader executed %d queries, want exactly 2", queryCount)
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

func TestFullReportChartLinkCanOnlyBeWrittenDuringPreflight(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	report := fullReportGraph(now).Report
	if err := repo.Create(context.Background(), report); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateChartID(context.Background(), report.ID, 10); !errors.Is(err, ErrFullReportImmutableWrite) {
		t.Fatalf("pending report chart update error = %v, want immutable write", err)
	}
	if err := repo.BeginPreflight(context.Background(), report.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateChartID(context.Background(), report.ID, 10); err != nil {
		t.Fatalf("preflight chart update failed: %v", err)
	}
	if err := db.Model(&model.FullReport{}).Where("id = ?", report.ID).Updates(map[string]any{"status": model.FullReportStatusGenerating, "current_stage": model.FullReportStatusGenerating}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateChartID(context.Background(), report.ID, 11); !errors.Is(err, ErrFullReportImmutableWrite) {
		t.Fatalf("generating report chart update error = %v, want immutable write", err)
	}
	var stored model.FullReport
	if err := db.First(&stored, report.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ChartID == nil || *stored.ChartID != 10 {
		t.Fatalf("chart link changed outside preflight: %v", stored.ChartID)
	}
}

func TestFullReportStageTransitionsRecordTimesAndRejectSkipping(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	report := fullReportGraph(now).Report
	report.Status = model.FullReportStatusGenerating
	report.CurrentStage = model.FullReportStatusGenerating
	report.GeneratingAt = &now
	if err := repo.Create(context.Background(), report); err != nil {
		t.Fatal(err)
	}
	assemblingAt := now.Add(2 * time.Second)
	if err := repo.BeginAssembling(context.Background(), report.ID, assemblingAt); err != nil {
		t.Fatal(err)
	}
	var stored model.FullReport
	if err := db.First(&stored, report.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.FullReportStatusAssembling || stored.AssemblingAt == nil || !stored.AssemblingAt.Equal(assemblingAt) {
		t.Fatalf("unexpected stage metadata: %+v", stored)
	}
	failedAt := now.Add(4 * time.Second)
	if err := repo.Fail(context.Background(), report.ID, "render_failed", "failed", failedAt); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&stored, report.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.FullReportStatusFailed || stored.FailedAt == nil || !stored.FailedAt.Equal(failedAt) {
		t.Fatalf("failed stage metadata missing: %+v", stored)
	}
	if err := repo.BeginAssembling(context.Background(), report.ID, failedAt.Add(time.Second)); !errors.Is(err, ErrFullReportImmutableWrite) {
		t.Fatalf("terminal transition err = %v", err)
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
	if err := db.Model(&model.FullReport{}).Where("id = ?", graph.Report.ID).Updates(map[string]any{"status": model.FullReportStatusGenerating, "current_stage": model.FullReportStatusGenerating}).Error; err != nil {
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

func TestFullReportFailConvergesUnfinishedChaptersAndCounts(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	graph := fullReportGraph(now)
	if err := repo.CreateGraph(context.Background(), graph); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReport{}).Where("id = ?", graph.Report.ID).Updates(map[string]any{
		"status": model.FullReportStatusGenerating, "current_stage": model.FullReportStatusGenerating,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReportChapter{}).Where("id IN ?", []uint64{graph.Chapters[0].ID, graph.Chapters[1].ID}).Updates(map[string]any{
		"status": model.FullReportChapterStatusSucceeded, "schema_valid": true,
		"validation_status": model.FullReportValidationStatusPassed, "completed_at": now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	failedAt := now.Add(time.Minute)
	if err := repo.Fail(context.Background(), graph.Report.ID, "generation_failed", "model routes exhausted", failedAt); err != nil {
		t.Fatal(err)
	}

	var report model.FullReport
	if err := db.First(&report, graph.Report.ID).Error; err != nil {
		t.Fatal(err)
	}
	if report.Status != model.FullReportStatusFailed || report.ChapterSucceeded != 2 || report.ChapterFailed != 8 {
		t.Fatalf("unexpected terminal counts: status=%s succeeded=%d failed=%d", report.Status, report.ChapterSucceeded, report.ChapterFailed)
	}
	var succeeded, failed []model.FullReportChapter
	if err := db.Where("report_id = ? AND status = ?", graph.Report.ID, model.FullReportChapterStatusSucceeded).Find(&succeeded).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("report_id = ? AND status = ?", graph.Report.ID, model.FullReportChapterStatusFailed).Find(&failed).Error; err != nil {
		t.Fatal(err)
	}
	if len(succeeded) != 2 || len(failed) != 8 {
		t.Fatalf("unexpected chapter states: succeeded=%d failed=%d", len(succeeded), len(failed))
	}
	for _, chapter := range failed {
		if chapter.ValidationStatus != model.FullReportValidationStatusFailed || chapter.ErrorCode != "generation_failed" || chapter.CompletedAt == nil || !chapter.CompletedAt.Equal(failedAt) {
			t.Fatalf("failed chapter did not converge: %+v", chapter)
		}
	}
}

func TestRecoverInterruptedExecutionClosesRunningAttempts(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	graph := fullReportGraph(now)
	if err := repo.CreateGraph(context.Background(), graph); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReport{}).Where("id = ?", graph.Report.ID).Updates(map[string]any{
		"status": model.FullReportStatusGenerating, "current_stage": model.FullReportStatusGenerating,
	}).Error; err != nil {
		t.Fatal(err)
	}
	attempt := &model.FullReportAttempt{
		ReportID: graph.Report.ID, ChapterID: graph.Chapters[0].ID, AttemptNo: 1, RouteNo: 1,
		Provider: "mock", Model: "mock-v1", Status: model.FullReportAttemptStatusRunning,
		ValidationStatus: model.FullReportValidationStatusPending, PromptHash: graph.Chapters[0].PromptHash,
		TraceID: "recovery-trace", StartedAt: now, CreatedAt: now,
	}
	if err := repo.AppendAttempt(context.Background(), attempt, &model.FullReportAttemptPayload{RequestParameters: model.JSONRaw(`{}`), RequestPrompt: "prompt", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	recoveredAt := now.Add(time.Minute)
	if err := repo.RecoverInterruptedExecution(context.Background(), graph.Report.ID, recoveredAt); err != nil {
		t.Fatal(err)
	}
	var stored model.FullReportAttempt
	if err := db.First(&stored, attempt.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.FullReportAttemptStatusFailed || stored.ValidationStatus != model.FullReportValidationStatusFailed || stored.ErrorCode != "worker_interrupted" || stored.FinishedAt == nil || !stored.FinishedAt.Equal(recoveredAt) {
		t.Fatalf("interrupted attempt was not closed: %+v", stored)
	}
	var chapter model.FullReportChapter
	if err := db.First(&chapter, graph.Chapters[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if chapter.Status != model.FullReportChapterStatusPending || chapter.AttemptCount != 1 {
		t.Fatalf("chapter recovery cursor changed unexpectedly: %+v", chapter)
	}
}

func TestFullReportRenderFailureKeepsSuccessfulChapterCounts(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	graph := fullReportGraph(now)
	if err := repo.CreateGraph(context.Background(), graph); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReportChapter{}).Where("report_id = ?", graph.Report.ID).Updates(map[string]any{
		"status": model.FullReportChapterStatusSucceeded, "schema_valid": true,
		"validation_status": model.FullReportValidationStatusPassed, "completed_at": now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReport{}).Where("id = ?", graph.Report.ID).Updates(map[string]any{
		"status": model.FullReportStatusRendering, "current_stage": model.FullReportStatusRendering,
		"chapter_succeeded": 10,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.Fail(context.Background(), graph.Report.ID, "render_failed", "pdf failed", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	var report model.FullReport
	if err := db.First(&report, graph.Report.ID).Error; err != nil {
		t.Fatal(err)
	}
	if report.ChapterSucceeded != 10 || report.ChapterFailed != 0 {
		t.Fatalf("render failure changed chapter outcome: succeeded=%d failed=%d", report.ChapterSucceeded, report.ChapterFailed)
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
	if err := db.Model(&model.FullReport{}).Where("id = ?", graph.Report.ID).Updates(map[string]any{"status": model.FullReportStatusRendering, "current_stage": model.FullReportStatusRendering}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.CompleteRendering(context.Background(), graph.Report.ID, "reports/test.pdf", "https://example.test/report.pdf", fmt.Sprintf("%064d", 300), now); !errors.Is(err, ErrFullReportNotReady) {
		t.Fatalf("err = %v, want not ready", err)
	}
	if err := db.Model(&model.FullReportChapter{}).Where("report_id = ?", graph.Report.ID).Updates(map[string]any{
		"status": model.FullReportChapterStatusSucceeded, "schema_valid": true, "validation_status": model.FullReportValidationStatusPassed,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReport{}).Where("id = ?", graph.Report.ID).Updates(map[string]any{"status": model.FullReportStatusAssembling, "current_stage": model.FullReportStatusAssembling}).Error; err != nil {
		t.Fatal(err)
	}
	finished := now.Add(time.Second)
	run := &model.FullReportValidationRun{ReportID: graph.Report.ID, ValidatorVersion: "report-validator-v1", Status: model.FullReportValidationStatusPassed, StartedAt: now, FinishedAt: &finished, CreatedAt: now}
	if err := repo.SaveAggregateValidation(context.Background(), run, &model.FullReportValidationPayload{ValidationResult: model.JSONRaw(`{"passed":true}`), CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.PrepareRendering(context.Background(), graph.Report.ID, result, "render-v1", 3, finished); err != nil {
		t.Fatal(err)
	}
	if err := repo.CompleteRendering(context.Background(), graph.Report.ID, "reports/test.pdf", "https://example.test/report.pdf", fmt.Sprintf("%064d", 300), now); err != nil {
		t.Fatal(err)
	}
	if err := repo.CompleteRendering(context.Background(), graph.Report.ID, "reports/test.pdf", "https://example.test/report.pdf", fmt.Sprintf("%064d", 300), now); err != nil {
		t.Fatalf("idempotent second completion failed: %v", err)
	}
}

func TestFullReportCompleteRequiresPassedAggregateValidation(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	graph := fullReportGraph(now)
	if err := repo.CreateGraph(context.Background(), graph); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReportChapter{}).Where("report_id = ?", graph.Report.ID).Updates(map[string]any{
		"status": model.FullReportChapterStatusSucceeded, "schema_valid": true, "validation_status": model.FullReportValidationStatusPassed,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReport{}).Where("id = ?", graph.Report.ID).Updates(map[string]any{"status": model.FullReportStatusAssembling, "current_stage": model.FullReportStatusAssembling}).Error; err != nil {
		t.Fatal(err)
	}
	finished := now.Add(time.Second)
	run := &model.FullReportValidationRun{ReportID: graph.Report.ID, ValidatorVersion: "report-validator-v1", Status: model.FullReportValidationStatusFailed, Retryable: true, AffectedChapters: 1, StartedAt: now, FinishedAt: &finished, CreatedAt: now}
	payload := &model.FullReportValidationPayload{ValidationResult: model.JSONRaw(`{"passed":false,"affected_chapters":[4]}`), CreatedAt: now}
	if err := repo.SaveAggregateValidation(context.Background(), run, payload); err != nil {
		t.Fatal(err)
	}
	result := &model.FullReportResult{Locale: "zh", Content: model.ReportContent{Locale: "zh"}, ContentHash: fmt.Sprintf("%064d", 200), RenderVersion: "render-v1", CreatedAt: now}
	if _, err := repo.PrepareRendering(context.Background(), graph.Report.ID, result, "render-v1", 3, finished); !errors.Is(err, ErrFullReportNotReady) {
		t.Fatalf("err = %v, want not ready", err)
	}
	var storedPayload model.FullReportValidationPayload
	if err := db.Where("validation_run_id = ?", run.ID).First(&storedPayload).Error; err != nil {
		t.Fatal(err)
	}
	if string(storedPayload.ValidationResult) == "" {
		t.Fatal("aggregate validation payload was not stored")
	}
}

func TestPrepareAggregateRetryResetsOnlyAffectedChapter(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	graph := fullReportGraph(now)
	if err := repo.CreateGraph(context.Background(), graph); err != nil {
		t.Fatal(err)
	}
	selected := uint64(44)
	if err := db.Model(&model.FullReport{}).Where("id = ?", graph.Report.ID).Updates(map[string]any{
		"status": model.FullReportStatusAssembling, "current_stage": model.FullReportStatusAssembling, "chapter_succeeded": 10,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReportChapter{}).Where("report_id = ?", graph.Report.ID).Updates(map[string]any{
		"status": model.FullReportChapterStatusSucceeded, "selected_attempt_id": selected,
		"schema_valid": true, "validation_status": model.FullReportValidationStatusPassed, "attempt_count": 1,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.PrepareAggregateRetry(context.Background(), graph.Report.ID, []uint8{4}, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var report model.FullReport
	if err := db.First(&report, graph.Report.ID).Error; err != nil {
		t.Fatal(err)
	}
	if report.Status != model.FullReportStatusGenerating || report.ChapterSucceeded != 9 {
		t.Fatalf("unexpected report after retry preparation: %+v", report)
	}
	var affected, untouched model.FullReportChapter
	if err := db.Where("report_id = ? AND chapter_no = 4", graph.Report.ID).First(&affected).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("report_id = ? AND chapter_no = 3", graph.Report.ID).First(&untouched).Error; err != nil {
		t.Fatal(err)
	}
	if affected.Status != model.FullReportChapterStatusPending || affected.SelectedAttemptID != nil || affected.AttemptCount != 1 {
		t.Fatalf("affected chapter was not reset correctly: %+v", affected)
	}
	if untouched.Status != model.FullReportChapterStatusSucceeded || untouched.SelectedAttemptID == nil {
		t.Fatalf("unaffected chapter changed: %+v", untouched)
	}
}

func TestAdminValidationAndChapterTraceAreReportScoped(t *testing.T) {
	repo, db := setupFullReportRepo(t)
	now := time.Now().UTC()
	graph := fullReportGraph(now)
	if err := repo.CreateGraph(context.Background(), graph); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.FullReport{}).Where("id = ?", graph.Report.ID).Updates(map[string]any{"status": model.FullReportStatusAssembling, "current_stage": model.FullReportStatusAssembling}).Error; err != nil {
		t.Fatal(err)
	}
	finished := now.Add(time.Second)
	run := &model.FullReportValidationRun{ReportID: graph.Report.ID, ValidatorVersion: "report-validator-v1", Status: model.FullReportValidationStatusPassed, StartedAt: now, FinishedAt: &finished, CreatedAt: now}
	if err := repo.SaveAggregateValidation(context.Background(), run, &model.FullReportValidationPayload{ValidationResult: model.JSONRaw(`{"passed":true}`), CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	runs, err := repo.AdminListValidationRuns(context.Background(), graph.Report.ID)
	if err != nil || len(runs) != 1 || runs[0].ID != run.ID {
		t.Fatalf("unexpected validation runs: %+v err=%v", runs, err)
	}
	trace, err := repo.AdminGetValidationTrace(context.Background(), graph.Report.ID, run.ID)
	if err != nil || trace.Payload.ValidationRunID != run.ID {
		t.Fatalf("unexpected validation trace: %+v err=%v", trace, err)
	}
	chapters, err := repo.AdminListChapters(context.Background(), graph.Report.ID)
	if err != nil || len(chapters) != 10 {
		t.Fatalf("unexpected chapters: %d err=%v", len(chapters), err)
	}
	chapterTrace, err := repo.AdminGetChapterTrace(context.Background(), graph.Report.ID, graph.Chapters[0].ID)
	if err != nil || chapterTrace.Payload.ChapterID != graph.Chapters[0].ID {
		t.Fatalf("unexpected chapter trace: %+v err=%v", chapterTrace, err)
	}
	if err := db.Model(&model.FullReportValidationRun{}).Where("id = ?", run.ID).Count(new(int64)).Error; err != nil {
		t.Fatal(err)
	}
}
