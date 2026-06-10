package job

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"fatelumen/backend/internal/pkg/logger"
)

var _ Queue = (*MemoryQueue)(nil)
var _ Queue = (*DBQueue)(nil)

func TestMemoryQueue_EnqueueDequeue(t *testing.T) {
	q := NewMemoryQueue()
	ctx := logger.WithTraceID(context.Background(), "trace-abc123")

	job := &Job{
		Type:    "report",
		Payload: json.RawMessage(`{"report_id":1}`),
	}
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if job.ID == "" {
		t.Fatal("expected job ID to be set")
	}
	if job.TraceID != "trace-abc123" {
		t.Fatalf("expected trace_id 'trace-abc123', got '%s'", job.TraceID)
	}
	if job.Status != StatusPending {
		t.Fatalf("expected pending, got %s", job.Status)
	}

	dequeued, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	if dequeued == nil {
		t.Fatal("expected a job")
	}
	if dequeued.ID != job.ID {
		t.Fatalf("expected job ID %s, got %s", job.ID, dequeued.ID)
	}
	if dequeued.Status != StatusProcessing {
		t.Fatalf("expected processing, got %s", dequeued.Status)
	}
}

func TestMemoryQueue_DequeueEmpty(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()
	job, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	if job != nil {
		t.Fatal("expected nil for empty queue")
	}
}

func TestMemoryQueue_UpdateStatus(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	q.Enqueue(ctx, &Job{Type: "report"})

	j, _ := q.Dequeue(ctx)
	if err := q.UpdateStatus(ctx, j.ID, StatusDone, "https://cdn.example.com/report.pdf"); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, _ := q.Get(ctx, j.ID)
	if got.Status != StatusDone {
		t.Fatalf("expected done, got %s", got.Status)
	}
	if got.Result != "https://cdn.example.com/report.pdf" {
		t.Fatalf("expected result URL, got '%s'", got.Result)
	}
}

func TestMemoryQueue_UpdateStatusIllegalTransition(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	q.Enqueue(ctx, &Job{Type: "report"})
	j, _ := q.Dequeue(ctx)

	// processing → pending is illegal, should be silently ignored
	if err := q.UpdateStatus(ctx, j.ID, StatusPending, ""); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	got, _ := q.Get(ctx, j.ID)
	if got.Status != StatusProcessing {
		t.Fatalf("status should remain processing, got %s", got.Status)
	}
}

func TestMemoryQueue_UpdateStatusToFailed(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	q.Enqueue(ctx, &Job{Type: "report"})
	j, _ := q.Dequeue(ctx)

	if err := q.UpdateStatus(ctx, j.ID, StatusFailed, "llm timeout"); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, _ := q.Get(ctx, j.ID)
	if got.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", got.Status)
	}
	if got.Attempts != 1 {
		t.Fatalf("expected attempts 1, got %d", got.Attempts)
	}
	if got.Result != "llm timeout" {
		t.Fatalf("expected result 'llm timeout', got '%s'", got.Result)
	}
}

func TestMemoryQueue_TraceIDInherited(t *testing.T) {
	q := NewMemoryQueue()
	ctx := logger.WithTraceID(context.Background(), "trace-xyz-789")

	job := &Job{Type: "report"}
	q.Enqueue(ctx, job)

	if job.TraceID != "trace-xyz-789" {
		t.Fatalf("expected trace_id 'trace-xyz-789', got '%s'", job.TraceID)
	}

	got, _ := q.Get(ctx, job.ID)
	if got.TraceID != "trace-xyz-789" {
		t.Fatalf("persisted trace_id mismatch: got '%s'", got.TraceID)
	}
}

func TestMemoryQueue_ConcurrentDequeue(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	n := 50
	for i := 0; i < n; i++ {
		q.Enqueue(ctx, &Job{Type: "report"})
	}

	var mu sync.Mutex
	dequeued := make(map[string]bool)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				job, _ := q.Dequeue(ctx)
				if job == nil {
					return
				}
				mu.Lock()
				if dequeued[job.ID] {
					t.Errorf("duplicate dequeue: %s", job.ID)
				}
				dequeued[job.ID] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(dequeued) != n {
		t.Fatalf("expected %d dequeued jobs, got %d", n, len(dequeued))
	}
}

func TestMemoryQueue_GetNonExistent(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()
	got, err := q.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for nonexistent job")
	}
}

func TestMemoryQueue_EnqueueDefaults(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	job := &Job{Type: "report"}
	q.Enqueue(ctx, job)

	if job.MaxAttempts != 3 {
		t.Fatalf("expected MaxAttempts 3, got %d", job.MaxAttempts)
	}
	if job.Status != StatusPending {
		t.Fatalf("expected StatusPending, got %s", job.Status)
	}
	if job.Payload == nil {
		t.Fatal("expected non-nil payload")
	}
}

func TestMemoryQueue_Get(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	job := &Job{Type: "test", Payload: json.RawMessage(`{"k":"v"}`)}
	q.Enqueue(ctx, job)

	got, err := q.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != job.ID {
		t.Fatal("ID mismatch")
	}
	if got.Type != "test" {
		t.Fatal("Type mismatch")
	}
	if string(got.Payload) != `{"k":"v"}` {
		t.Fatalf("Payload mismatch: %s", string(got.Payload))
	}
}
