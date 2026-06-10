package job

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"fatelumen/backend/internal/pkg/logger"
)

// ---------- test helpers ----------

type controllableHandler struct {
	mu         sync.Mutex
	failCount  int
	failUntil  int // fail this many times before succeeding
	panicOn    int // panic on this attempt number (0-indexed)
	attempts   int
	blockCh    chan struct{} // block handler until closed
	traceIDCh  chan string   // capture trace_id received
	resultVal  string
}

func (h *controllableHandler) Handle(ctx context.Context, job *Job) (string, error) {
	h.mu.Lock()
	h.attempts++
	attempt := h.attempts
	h.mu.Unlock()

	if h.traceIDCh != nil {
		select {
		case h.traceIDCh <- logger.TraceIDFromCtx(ctx):
		default:
		}
	}

	if h.blockCh != nil {
		<-h.blockCh
	}

	if h.panicOn != 0 && (h.panicOn < 0 || attempt == h.panicOn) {
		panic(fmt.Sprintf("intentional panic on attempt %d", attempt))
	}

	if attempt <= h.failUntil {
		return "", fmt.Errorf("fake error on attempt %d", attempt)
	}

	return h.resultVal, nil
}

func (h *controllableHandler) getAttempts() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.attempts
}

func enqueueJob(q Queue, ctx context.Context, jobType string, payload json.RawMessage) (*Job, error) {
	job := &Job{Type: jobType, Payload: payload}
	if err := q.Enqueue(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

func waitJobStatus(q Queue, id string, want Status, timeout time.Duration) (*Job, error) {
	deadline := time.After(timeout)
	for {
		job, err := q.Get(context.Background(), id)
		if err != nil || job == nil {
			select {
			case <-deadline:
				return nil, fmt.Errorf("timeout waiting for status %s on job %s", want, id)
			case <-time.After(10 * time.Millisecond):
				continue
			}
		}
		if job.Status == want {
			return job, nil
		}
		select {
		case <-deadline:
			return nil, fmt.Errorf("timeout waiting for status %s on job %s, current=%s", want, id, job.Status)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// ---------- tests ----------

func TestWorker_Success(t *testing.T) {
	q := NewMemoryQueue()
	registry := NewHandlerRegistry()

	handler := &controllableHandler{resultVal: "https://cdn.example.com/ok.pdf"}
	registry.Register("report", handler)

	ctx := logger.WithTraceID(context.Background(), "trace-success")
	job, _ := enqueueJob(q, ctx, "report", json.RawMessage(`{"id":1}`))

	w := NewWorker(q, registry, 10*time.Millisecond, 1)
	w.Start(context.Background())
	defer w.Stop()

	j, err := waitJobStatus(q, job.ID, StatusDone, 3*time.Second)
	if err != nil {
		t.Fatalf("waitJobStatus: %v", err)
	}
	if j.Result != "https://cdn.example.com/ok.pdf" {
		t.Fatalf("expected result, got '%s'", j.Result)
	}
	if j.Status != StatusDone {
		t.Fatalf("expected done, got %s", j.Status)
	}
}

func TestWorker_RetryThenSuccess(t *testing.T) {
	q := NewMemoryQueue()
	registry := NewHandlerRegistry()

	// Fail first 2 attempts, succeed on 3rd
	handler := &controllableHandler{failUntil: 2, resultVal: "ok-after-retry"}
	registry.Register("report", handler)

	ctx := context.Background()
	job, _ := enqueueJob(q, ctx, "report", json.RawMessage(`{"id":2}`))
	job.MaxAttempts = 4

	w := NewWorker(q, registry, 10*time.Millisecond, 1)
	w.Start(context.Background())
	defer w.Stop()

	j, err := waitJobStatus(q, job.ID, StatusDone, 3*time.Second)
	if err != nil {
		t.Fatalf("waitJobStatus: %v", err)
	}
	if j.Result != "ok-after-retry" {
		t.Fatalf("expected result 'ok-after-retry', got '%s'", j.Result)
	}
	if handler.getAttempts() != 3 {
		t.Fatalf("expected 3 attempts, got %d", handler.getAttempts())
	}
}

func TestWorker_PermanentFail(t *testing.T) {
	q := NewMemoryQueue()
	registry := NewHandlerRegistry()

	// Always fail
	handler := &controllableHandler{failUntil: 99, resultVal: "nope"}
	registry.Register("report", handler)

	ctx := context.Background()
	job := &Job{Type: "report", Payload: json.RawMessage(`{"id":3}`), MaxAttempts: 2}
	q.Enqueue(ctx, job)

	w := NewWorker(q, registry, 10*time.Millisecond, 1)
	w.Start(context.Background())
	defer w.Stop()

	j, err := waitJobStatus(q, job.ID, StatusFailed, 3*time.Second)
	if err != nil {
		t.Fatalf("waitJobStatus: %v", err)
	}
	if j.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", j.Status)
	}
	attempts := handler.getAttempts()
	if attempts != 2 {
		t.Fatalf("expected 2 attempts (MaxAttempts=2), got %d", attempts)
	}
}

func TestWorker_PanicRecovery(t *testing.T) {
	q := NewMemoryQueue()
	registry := NewHandlerRegistry()

	// Panic on first attempt, succeed afterward
	handler := &controllableHandler{panicOn: 1, resultVal: "recovered"}
	registry.Register("report", handler)

	ctx := context.Background()
	job, _ := enqueueJob(q, ctx, "report", json.RawMessage(`{"id":4}`))
	job.MaxAttempts = 3

	w := NewWorker(q, registry, 10*time.Millisecond, 1)
	w.Start(context.Background())
	defer w.Stop()

	j, err := waitJobStatus(q, job.ID, StatusDone, 3*time.Second)
	if err != nil {
		t.Fatalf("waitJobStatus: %v", err)
	}
	if j.Status != StatusDone {
		t.Fatalf("expected done after panic recovery, got %s", j.Status)
	}
	attempts := handler.getAttempts()
	if attempts < 2 {
		t.Fatalf("expected at least 2 attempts (1 panic + 1 success), got %d", attempts)
	}
}

func TestWorker_PanicAllRetries(t *testing.T) {
	q := NewMemoryQueue()
	registry := NewHandlerRegistry()

	// Panic every time (panicOn = -1)
	handler := &controllableHandler{panicOn: -1, resultVal: "never"}
	registry.Register("report", handler)

	ctx := context.Background()
	job, _ := enqueueJob(q, ctx, "report", json.RawMessage(`{"id":5}`))
	job.MaxAttempts = 2

	w := NewWorker(q, registry, 10*time.Millisecond, 1)
	w.Start(context.Background())
	defer w.Stop()

	j, err := waitJobStatus(q, job.ID, StatusFailed, 3*time.Second)
	if err != nil {
		t.Fatalf("waitJobStatus: %v", err)
	}
	if j.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", j.Status)
	}
}

func TestWorker_TraceIDContinuation(t *testing.T) {
	q := NewMemoryQueue()
	registry := NewHandlerRegistry()

	traceCh := make(chan string, 1)
	handler := &controllableHandler{resultVal: "ok", traceIDCh: traceCh}
	registry.Register("report", handler)

	ctx := logger.WithTraceID(context.Background(), "trace-chain-999")
	job, _ := enqueueJob(q, ctx, "report", json.RawMessage(`{"id":6}`))

	w := NewWorker(q, registry, 10*time.Millisecond, 1)
	w.Start(context.Background())
	defer w.Stop()

	j, err := waitJobStatus(q, job.ID, StatusDone, 3*time.Second)
	if err != nil {
		t.Fatalf("waitJobStatus: %v", err)
	}
	if j.Status != StatusDone {
		t.Fatalf("expected done, got %s", j.Status)
	}

	select {
	case receivedTraceID := <-traceCh:
		if receivedTraceID != "trace-chain-999" {
			t.Fatalf("expected trace_id 'trace-chain-999', got '%s'", receivedTraceID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for trace_id")
	}
}

func TestWorker_GracefulShutdown(t *testing.T) {
	q := NewMemoryQueue()
	registry := NewHandlerRegistry()

	// Handler blocks until we unblock it
	blockCh := make(chan struct{})
	handler := &controllableHandler{resultVal: "shutdown-ok", blockCh: blockCh}
	registry.Register("report", handler)

	ctx := context.Background()
	job, _ := enqueueJob(q, ctx, "report", json.RawMessage(`{"id":7}`))

	poll := 10 * time.Millisecond
	w := NewWorker(q, registry, poll, 1)
	w.Start(context.Background())

	// Wait for the job to be picked up (processing)
	j, err := waitJobStatus(q, job.ID, StatusProcessing, 3*time.Second)
	if err != nil {
		t.Fatalf("job should be processing: %v", err)
	}
	_ = j

	// Call Stop in a goroutine — it should block until handler finishes
	stopped := make(chan struct{})
	go func() {
		w.Stop()
		close(stopped)
	}()

	// Handler should still be blocked, Stop should NOT have returned yet
	select {
	case <-stopped:
		t.Fatal("Stop returned before handler finished")
	case <-time.After(200 * time.Millisecond):
		// expected
	}

	// Unblock handler
	close(blockCh)

	// Now Stop should return
	select {
	case <-stopped:
		// OK
	case <-time.After(3 * time.Second):
		t.Fatal("Stop did not return after handler finished")
	}

	// Job should be done
	j, err = q.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if j.Status != StatusDone {
		t.Fatalf("expected done after graceful shutdown, got %s", j.Status)
	}
}

func TestWorker_HandlerNotFound(t *testing.T) {
	q := NewMemoryQueue()
	registry := NewHandlerRegistry()
	// No handler registered

	ctx := context.Background()
	job, _ := enqueueJob(q, ctx, "unknown_type", json.RawMessage(`{"id":8}`))

	w := NewWorker(q, registry, 10*time.Millisecond, 1)
	w.Start(context.Background())
	defer w.Stop()

	j, err := waitJobStatus(q, job.ID, StatusFailed, 3*time.Second)
	if err != nil {
		t.Fatalf("waitJobStatus: %v", err)
	}
	if j.Status != StatusFailed {
		t.Fatalf("expected failed for unknown type, got %s", j.Status)
	}
	if j.Result == "" {
		t.Fatal("expected error result message")
	}
}

func TestHandlerRegistry_RegisterAndGet(t *testing.T) {
	r := NewHandlerRegistry()

	h := &controllableHandler{resultVal: "test"}
	r.Register("test_type", h)

	got, err := r.Get("test_type")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != h {
		t.Fatal("handler mismatch")
	}

	_, err = r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent type")
	}
}

func TestWorker_MultipleWorkers(t *testing.T) {
	q := NewMemoryQueue()
	registry := NewHandlerRegistry()

	n := 20
	var mu sync.Mutex
	completed := make(map[string]bool)
	handler := &controllableHandler{resultVal: "multi-ok"}
	registry.Register("report", handler)

	ctx := context.Background()
	var ids []string
	for i := 0; i < n; i++ {
		job, _ := enqueueJob(q, ctx, "report", json.RawMessage(fmt.Sprintf(`{"i":%d}`, i)))
		ids = append(ids, job.ID)
	}

	w := NewWorker(q, registry, 10*time.Millisecond, 4)
	w.Start(context.Background())
	defer w.Stop()

	deadline := time.After(5 * time.Second)
	for len(completed) < n {
		for _, id := range ids {
			job, _ := q.Get(ctx, id)
			if job != nil && (job.Status == StatusDone || job.Status == StatusFailed) {
				mu.Lock()
				completed[id] = true
				mu.Unlock()
			}
		}
		select {
		case <-deadline:
			t.Fatalf("timeout: only %d/%d jobs completed", len(completed), n)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func TestHandlerRegistry_Overwrite(t *testing.T) {
	r := NewHandlerRegistry()
	h1 := &controllableHandler{resultVal: "first"}
	h2 := &controllableHandler{resultVal: "second"}
	r.Register("type", h1)
	r.Register("type", h2)
	got, _ := r.Get("type")
	if got != h2 {
		t.Fatal("expected second handler after overwrite")
	}
}

func TestNewJobID_NotEmpty(t *testing.T) {
	if NewJobID() == "" {
		t.Fatal("NewJobID returned empty")
	}
}
