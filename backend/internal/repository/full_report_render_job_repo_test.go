package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"fatelumen/backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func renderJobTestRepo(t *testing.T) (*FullReportRenderJobRepo, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.FullReportRenderJob{}); err != nil {
		t.Fatal(err)
	}
	return NewFullReportRenderJobRepo(db), db
}

func TestFullReportRenderJobEnsureIsIdempotent(t *testing.T) {
	repo, _ := renderJobTestRepo(t)
	now := time.Now().UTC()
	first, created, err := repo.Ensure(context.Background(), 9, "pdf-v1", 3, now)
	if err != nil || !created {
		t.Fatalf("first Ensure: created=%v err=%v", created, err)
	}
	second, created, err := repo.Ensure(context.Background(), 9, "pdf-v1", 3, now.Add(time.Second))
	if err != nil || created || first.ID != second.ID {
		t.Fatalf("duplicate Ensure: first=%d second=%d created=%v err=%v", first.ID, second.ID, created, err)
	}
}

func TestFullReportRenderJobClaimAndRetryBudget(t *testing.T) {
	repo, _ := renderJobTestRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()
	task, _, _ := repo.Ensure(ctx, 10, "pdf-v1", 2, now)
	claimed, err := repo.Claim(ctx, task.ID, now)
	if err != nil || claimed.AttemptCount != 1 {
		t.Fatalf("first claim: task=%+v err=%v", claimed, err)
	}
	if _, err := repo.Claim(ctx, task.ID, now); !errors.Is(err, ErrFullReportRenderJobNotClaimed) {
		t.Fatalf("duplicate claim should fail, got %v", err)
	}
	terminal, err := repo.RecordFailure(ctx, task.ID, "render_failed", "first", now.Add(time.Second))
	if err != nil || terminal {
		t.Fatalf("first failure should requeue: terminal=%v err=%v", terminal, err)
	}
	claimed, err = repo.Claim(ctx, task.ID, now.Add(2*time.Second))
	if err != nil || claimed.AttemptCount != 2 {
		t.Fatalf("second claim: task=%+v err=%v", claimed, err)
	}
	terminal, err = repo.RecordFailure(ctx, task.ID, "render_failed", "second", now.Add(3*time.Second))
	if err != nil || !terminal {
		t.Fatalf("second failure should exhaust budget: terminal=%v err=%v", terminal, err)
	}
	stored, _ := repo.Get(ctx, task.ID)
	if stored.Status != model.FullReportRenderJobStatusFailed || stored.FinishedAt == nil {
		t.Fatalf("unexpected terminal task: %+v", stored)
	}
}

func TestFullReportRenderJobRecoverStale(t *testing.T) {
	repo, db := renderJobTestRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()
	retryable, _, _ := repo.Ensure(ctx, 11, "pdf-v1", 3, now.Add(-time.Hour))
	exhausted, _, _ := repo.Ensure(ctx, 12, "pdf-v1", 1, now.Add(-time.Hour))
	_, _ = repo.Claim(ctx, retryable.ID, now.Add(-time.Hour))
	_, _ = repo.Claim(ctx, exhausted.ID, now.Add(-time.Hour))
	db.Model(&model.FullReportRenderJob{}).Where("id IN ?", []uint64{retryable.ID, exhausted.ID}).Update("updated_at", now.Add(-time.Hour))

	recovered, failed, err := repo.RecoverStale(ctx, now.Add(-10*time.Minute), now)
	if err != nil || len(recovered) != 1 || recovered[0].ID != retryable.ID || failed != 1 {
		t.Fatalf("RecoverStale: recovered=%+v failed=%d err=%v", recovered, failed, err)
	}
	failedTask, _ := repo.Get(ctx, exhausted.ID)
	if failedTask.Status != model.FullReportRenderJobStatusFailed || failedTask.ErrorCode != "render_interrupted" {
		t.Fatalf("unexpected exhausted task: %+v", failedTask)
	}
}
