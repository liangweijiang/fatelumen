package job

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ---------- MemoryQueue.ReclaimStale ----------

func TestMemoryQueue_ReclaimStale_Noop(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	// Enqueue, dequeue to processing, then try reclaim
	enqueued := &Job{Type: "report"}
	if err := q.Enqueue(ctx, enqueued); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	_, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}

	// Set updated_at to 10 minutes ago to simulate stale
	q.mu.Lock()
	job := q.jobs[enqueued.ID]
	job.UpdatedAt = time.Now().Add(-10 * time.Minute)
	q.mu.Unlock()

	reclaimed, failed, err := q.ReclaimStale(ctx, time.Minute)
	if err != nil {
		t.Fatalf("ReclaimStale: %v", err)
	}
	if reclaimed != 0 || failed != 0 {
		t.Fatalf("expected 0 reclaimed/0 failed for memory queue no-op, got %d/%d", reclaimed, failed)
	}

	// Job should still be processing (no-op)
	got, _ := q.Get(ctx, enqueued.ID)
	if got.Status != StatusProcessing {
		t.Fatalf("expected processing after no-op reclaim, got %s", got.Status)
	}
}

// ---------- DBQueue.ReclaimStale (in-memory SQLite) ----------

func newSQLiteDBQueue(t *testing.T) *DBQueue {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewDBQueue(db)
}

func createJobInDB(t *testing.T, q *DBQueue, status Status, attempts int, maxAttempts int, updatedAgo time.Duration) *Job {
	t.Helper()
	now := time.Now()
	job := &Job{
		ID:          NewJobID(),
		Type:        "report",
		Status:      status,
		Attempts:    attempts,
		MaxAttempts: maxAttempts,
		Payload:     json.RawMessage(`{"report_id":1}`),
		CreatedAt:   now,
		UpdatedAt:   now.Add(-updatedAgo),
	}
	if err := q.db.Create(job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}
	return job
}

func TestDBQueue_ReclaimStale_ProcessingStaleUnderMax_ResetToPending(t *testing.T) {
	q := newSQLiteDBQueue(t)
	ctx := context.Background()

	job := createJobInDB(t, q, StatusProcessing, 1, 3, 10*time.Minute)

	reclaimed, failed, err := q.ReclaimStale(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("ReclaimStale: %v", err)
	}
	if reclaimed != 1 {
		t.Fatalf("expected 1 reclaimed, got %d", reclaimed)
	}
	if failed != 0 {
		t.Fatalf("expected 0 failed, got %d", failed)
	}

	got, _ := q.Get(ctx, job.ID)
	if got.Status != StatusPending {
		t.Fatalf("expected pending after reclaim, got %s", got.Status)
	}
	if got.Attempts != 2 {
		t.Fatalf("expected attempts incremented to 2, got %d", got.Attempts)
	}
	if got.Result == "" {
		t.Fatal("expected result message set")
	}
}

func TestDBQueue_ReclaimStale_ProcessingStaleAtMax_TransitionToFailed(t *testing.T) {
	q := newSQLiteDBQueue(t)
	ctx := context.Background()

	job := createJobInDB(t, q, StatusProcessing, 3, 3, 10*time.Minute)

	reclaimed, failed, err := q.ReclaimStale(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("ReclaimStale: %v", err)
	}
	if reclaimed != 0 {
		t.Fatalf("expected 0 reclaimed, got %d", reclaimed)
	}
	if failed != 1 {
		t.Fatalf("expected 1 failed, got %d", failed)
	}

	got, _ := q.Get(ctx, job.ID)
	if got.Status != StatusFailed {
		t.Fatalf("expected failed after reclaim, got %s", got.Status)
	}
	if got.Result == "" {
		t.Fatal("expected result message set")
	}
}

func TestDBQueue_ReclaimStale_ProcessingNotStale_Untouched(t *testing.T) {
	q := newSQLiteDBQueue(t)
	ctx := context.Background()

	// Updated 2 minutes ago, threshold is 5 minutes — not stale
	job := createJobInDB(t, q, StatusProcessing, 1, 3, 2*time.Minute)

	reclaimed, failed, err := q.ReclaimStale(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("ReclaimStale: %v", err)
	}
	if reclaimed != 0 || failed != 0 {
		t.Fatalf("expected 0 reclaimed/0 failed, got %d/%d", reclaimed, failed)
	}

	got, _ := q.Get(ctx, job.ID)
	if got.Status != StatusProcessing {
		t.Fatalf("expected processing unchanged, got %s", got.Status)
	}
	if got.Attempts != 1 {
		t.Fatalf("expected attempts unchanged at 1, got %d", got.Attempts)
	}
}

func TestDBQueue_ReclaimStale_PendingAndDone_Untouched(t *testing.T) {
	q := newSQLiteDBQueue(t)
	ctx := context.Background()

	pendingJob := createJobInDB(t, q, StatusPending, 0, 3, 10*time.Minute)
	doneJob := createJobInDB(t, q, StatusDone, 1, 3, 10*time.Minute)

	reclaimed, failed, err := q.ReclaimStale(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("ReclaimStale: %v", err)
	}
	if reclaimed != 0 || failed != 0 {
		t.Fatalf("expected 0 reclaimed/0 failed, got %d/%d", reclaimed, failed)
	}

	pending, _ := q.Get(ctx, pendingJob.ID)
	if pending.Status != StatusPending {
		t.Fatalf("pending job should remain pending, got %s", pending.Status)
	}
	done, _ := q.Get(ctx, doneJob.ID)
	if done.Status != StatusDone {
		t.Fatalf("done job should remain done, got %s", done.Status)
	}
}

func TestDBQueue_ReclaimStale_MultipleJobs_MixedScenarios(t *testing.T) {
	q := newSQLiteDBQueue(t)
	ctx := context.Background()

	// Stale processing, attempts < max → should reclaim
	createJobInDB(t, q, StatusProcessing, 0, 3, 10*time.Minute)

	// Stale processing, attempts >= max → should fail
	createJobInDB(t, q, StatusProcessing, 3, 3, 10*time.Minute)

	// Not stale processing → untouched
	createJobInDB(t, q, StatusProcessing, 1, 3, 1*time.Minute)

	// Pending → untouched
	createJobInDB(t, q, StatusPending, 0, 3, 10*time.Minute)

	// Done → untouched
	createJobInDB(t, q, StatusDone, 1, 3, 10*time.Minute)

	reclaimed, failed, err := q.ReclaimStale(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("ReclaimStale: %v", err)
	}
	if reclaimed != 1 {
		t.Fatalf("expected 1 reclaimed, got %d", reclaimed)
	}
	if failed != 1 {
		t.Fatalf("expected 1 failed, got %d", failed)
	}
}
