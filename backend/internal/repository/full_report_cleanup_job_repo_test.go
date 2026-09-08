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
