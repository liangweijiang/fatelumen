package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/notify"
	"fatelumen/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type notifierRecorder struct {
	messages []notify.Message
	err      error
}

func (n *notifierRecorder) Send(_ context.Context, msg notify.Message) error {
	n.messages = append(n.messages, msg)
	return n.err
}

func (n *notifierRecorder) Channel() string { return "test" }

func TestFullReportOutcomeNotifierUsesFrozenReportLocaleAndTerminalTemplate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.FullReport{}, &model.FullReportResult{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	report := model.FullReport{
		PublicID: "01JNOTIFICATION000000000000", UserID: 42, Locale: "ja",
		Status: model.FullReportStatusCompleted, CurrentStage: model.FullReportStatusCompleted,
		ChapterTotal: 10, ChapterSucceeded: 10, FactsHash: "facts", RetentionPolicy: "days:30",
		ExpiresAt: now.Add(30 * 24 * time.Hour), CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&report).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.FullReportResult{ReportID: report.ID, Locale: "ja", ContentHash: "content", PDFURL: "https://example.test/report.pdf", CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	recorder := &notifierRecorder{}
	outcomes := NewFullReportOutcomeNotifier(repository.NewFullReportRepo(db), recorder)
	outcomes.Completed(context.Background(), report.ID)
	outcomes.Failed(context.Background(), report.ID, "generation_failed")
	if len(recorder.messages) != 2 {
		t.Fatalf("message count = %d, want 2", len(recorder.messages))
	}
	if recorder.messages[0].Template != "report_ready" || recorder.messages[0].Locale != "ja" || recorder.messages[0].To != "42" {
		t.Fatalf("completed message = %+v", recorder.messages[0])
	}
	if recorder.messages[1].Template != "report_failed" || recorder.messages[1].Data["error_code"] != "generation_failed" {
		t.Fatalf("failed message = %+v", recorder.messages[1])
	}
}

func TestFullReportOutcomeNotifierChannelFailureDoesNotEscape(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.FullReport{}, &model.FullReportResult{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	report := model.FullReport{PublicID: "01JNOTIFYFAILURE0000000000", UserID: 43, Locale: "zh", Status: model.FullReportStatusFailed, CurrentStage: model.FullReportStatusFailed, ChapterTotal: 10, FactsHash: "facts", RetentionPolicy: "days:30", ExpiresAt: now.Add(30 * 24 * time.Hour), CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&report).Error; err != nil {
		t.Fatal(err)
	}
	recorder := &notifierRecorder{err: errors.New("channel unavailable")}
	outcomes := NewFullReportOutcomeNotifier(repository.NewFullReportRepo(db), recorder)
	outcomes.Failed(context.Background(), report.ID, "generation_failed")
	if len(recorder.messages) != 1 {
		t.Fatalf("message count = %d, want 1", len(recorder.messages))
	}
}
