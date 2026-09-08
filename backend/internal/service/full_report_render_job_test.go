package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type renderRunnerStub struct {
	mu      sync.Mutex
	active  int
	maxSeen int
	delay   time.Duration
	err     error
}

func (r *renderRunnerStub) Run(ctx context.Context, task *model.FullReportRenderJob) error {
	r.mu.Lock()
	r.active++
	if r.active > r.maxSeen {
		r.maxSeen = r.active
	}
	r.mu.Unlock()
	defer func() { r.mu.Lock(); r.active--; r.mu.Unlock() }()
	select {
	case <-time.After(r.delay):
		return r.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func renderServiceTestRepo(t *testing.T) *repository.FullReportRenderJobRepo {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.FullReportRenderJob{}); err != nil {
		t.Fatal(err)
	}
	return repository.NewFullReportRenderJobRepo(db)
}

func renderQueueJob(t *testing.T, renderJobID uint64) *job.Job {
	t.Helper()
	payload, err := json.Marshal(fullReportPDFJobPayload{RenderJobID: renderJobID})
	if err != nil {
		t.Fatal(err)
	}
	return &job.Job{ID: job.NewJobID(), Type: FullReportPDFJobType, Payload: payload}
}

func TestFullReportPDFJobHandlerSuccessAndDuplicateDelivery(t *testing.T) {
	repo := renderServiceTestRepo(t)
	now := time.Now().UTC()
	task, _, _ := repo.Ensure(context.Background(), 20, "pdf-v1", 3, now)
	runner := &renderRunnerStub{}
	handler := NewFullReportPDFJobHandler(repo, runner, 1)
	queued := renderQueueJob(t, task.ID)
	if _, err := handler.Handle(context.Background(), queued); err != nil {
		t.Fatal(err)
	}
	if result, err := handler.Handle(context.Background(), queued); err != nil || result != model.FullReportRenderJobStatusSucceeded {
		t.Fatalf("duplicate delivery: result=%s err=%v", result, err)
	}
	stored, _ := repo.Get(context.Background(), task.ID)
	if stored.AttemptCount != 1 || stored.Status != model.FullReportRenderJobStatusSucceeded {
		t.Fatalf("unexpected stored task: %+v", stored)
	}
}

func TestFullReportPDFJobHandlerFailureRequeues(t *testing.T) {
	repo := renderServiceTestRepo(t)
	task, _, _ := repo.Ensure(context.Background(), 21, "pdf-v1", 2, time.Now().UTC())
	handler := NewFullReportPDFJobHandler(repo, &renderRunnerStub{err: errors.New("renderer unavailable")}, 1)
	if _, err := handler.Handle(context.Background(), renderQueueJob(t, task.ID)); err == nil {
		t.Fatal("expected render error")
	}
	stored, _ := repo.Get(context.Background(), task.ID)
	if stored.Status != model.FullReportRenderJobStatusQueued || stored.AttemptCount != 1 || stored.ErrorCode != "render_failed" {
		t.Fatalf("unexpected retry state: %+v", stored)
	}
}

func TestFullReportPDFJobHandlerConcurrencyOne(t *testing.T) {
	repo := renderServiceTestRepo(t)
	runner := &renderRunnerStub{delay: 25 * time.Millisecond}
	handler := NewFullReportPDFJobHandler(repo, runner, 1)
	now := time.Now().UTC()
	first, _, _ := repo.Ensure(context.Background(), 22, "pdf-v1", 3, now)
	second, _, _ := repo.Ensure(context.Background(), 23, "pdf-v1", 3, now)

	var wg sync.WaitGroup
	for _, task := range []*model.FullReportRenderJob{first, second} {
		wg.Add(1)
		go func(task *model.FullReportRenderJob) {
			defer wg.Done()
			if _, err := handler.Handle(context.Background(), renderQueueJob(t, task.ID)); err != nil {
				t.Errorf("Handle: %v", err)
			}
		}(task)
	}
	wg.Wait()
	if runner.maxSeen != 1 {
		t.Fatalf("expected max concurrency 1, got %d", runner.maxSeen)
	}
}

func TestRecoverFullReportPDFJobsEnqueuesDurableQueuedTasks(t *testing.T) {
	repo := renderServiceTestRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()
	task, _, _ := repo.Ensure(ctx, 24, "pdf-v1", 3, now.Add(-time.Hour))
	_, _ = repo.Claim(ctx, task.ID, now.Add(-time.Hour))
	queue := job.NewMemoryQueue()
	enqueued, failed, err := RecoverFullReportPDFJobs(ctx, repo, queue, 10*time.Minute)
	if err != nil || enqueued != 1 || failed != 0 {
		t.Fatalf("Recover: enqueued=%d failed=%d err=%v", enqueued, failed, err)
	}
	delivery, err := queue.Dequeue(ctx)
	if err != nil || delivery == nil || delivery.Type != FullReportPDFJobType {
		t.Fatalf("unexpected recovery delivery: job=%+v err=%v", delivery, err)
	}
}
