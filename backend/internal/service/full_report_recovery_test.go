package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRecoverInterruptedFullReportsRequeuesOnlyActiveGenerationStages(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.FullReport{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	profileID := uint64(21)
	statuses := []string{
		model.FullReportStatusPending,
		model.FullReportStatusPreflighting,
		model.FullReportStatusGenerating,
		model.FullReportStatusAssembling,
		model.FullReportStatusRendering,
		model.FullReportStatusCompleted,
		model.FullReportStatusFailed,
	}
	for i, status := range statuses {
		report := &model.FullReport{
			PublicID: fmt.Sprintf("01J000000000000000000000%02d", i), UserID: uint64(i + 1), ProfileID: &profileID,
			Locale: "zh", Status: status, CurrentStage: status, ChapterTotal: 10, ChapterConcurrency: 3,
			ProviderChainKey: "default", FactsHash: fmt.Sprintf("%064d", i+1), RetentionPolicy: "30d",
			ExpiresAt: now.Add(30 * 24 * time.Hour), CreatedAt: now.Add(time.Duration(i) * time.Second), UpdatedAt: now,
		}
		if err := db.Create(report).Error; err != nil {
			t.Fatal(err)
		}
	}

	queue := job.NewMemoryQueue()
	recovered, err := RecoverInterruptedFullReports(context.Background(), repository.NewFullReportRepo(db), queue)
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 4 {
		t.Fatalf("recovered = %d, want 4", recovered)
	}
	for i := 0; i < recovered; i++ {
		queued, err := queue.Dequeue(context.Background(), job.LaneReportGeneration)
		if err != nil || queued == nil {
			t.Fatalf("missing recovered delivery %d: job=%+v err=%v", i, queued, err)
		}
		if queued.Type != FullReportJobType || queued.Attempts != 1 || queued.MaxAttempts != 1 {
			t.Fatalf("unexpected recovered job: %+v", queued)
		}
		var payload fullReportPayload
		if err := json.Unmarshal(queued.Payload, &payload); err != nil || payload.ReportID == 0 || payload.ProfileID != profileID {
			t.Fatalf("invalid recovered payload: %+v err=%v", payload, err)
		}
	}
	queued, err := queue.Dequeue(context.Background(), job.LaneReportGeneration)
	if err != nil || queued != nil {
		t.Fatalf("terminal/rendering report was unexpectedly queued: job=%+v err=%v", queued, err)
	}
}
