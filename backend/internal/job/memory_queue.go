package job

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"fatelumen/backend/internal/pkg/logger"
)

// MemoryQueue 进程内内存队列实现（sync.Mutex + map + slice）。
// 用于本地开发与单测。
type MemoryQueue struct {
	mu       sync.Mutex
	jobs     map[string]*Job
	order    []string // job IDs in insertion order for FIFO dequeue
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{jobs: make(map[string]*Job)}
}

func (q *MemoryQueue) Enqueue(ctx context.Context, job *Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if job.ID == "" {
		job.ID = NewJobID()
	}
	job.TraceID = logger.TraceIDFromCtx(ctx)
	if job.Status == "" {
		job.Status = StatusPending
	}
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = 3
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = time.Now()
	}
	if job.Payload == nil {
		job.Payload = json.RawMessage("{}")
	}

	copyJob := *job
	q.jobs[job.ID] = &copyJob
	q.order = append(q.order, job.ID)
	return nil
}

func (q *MemoryQueue) Dequeue(ctx context.Context) (*Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, id := range q.order {
		job, ok := q.jobs[id]
		if !ok {
			continue
		}
		if job.Status == StatusPending {
			job.Status = StatusProcessing
			job.UpdatedAt = time.Now()
			// 从顺序中移除
			q.order = append(q.order[:i], q.order[i+1:]...)
			copyJob := *job
			return &copyJob, nil
		}
	}
	return nil, nil
}

func (q *MemoryQueue) UpdateStatus(ctx context.Context, id string, status Status, result string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, ok := q.jobs[id]
	if !ok {
		return nil
	}

	if !CanTransit(job.Status, status) {
		logger.FromCtx(ctx).Error("illegal job status transition",
			"job_id", id,
			"from", string(job.Status),
			"to", string(status),
		)
		return nil // 幂等忽略
	}

	job.Status = status
	job.Result = result
	job.UpdatedAt = time.Now()
	if status == StatusFailed {
		job.Attempts++
	}
	return nil
}

func (q *MemoryQueue) Get(ctx context.Context, id string) (*Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, ok := q.jobs[id]
	if !ok {
		return nil, nil
	}
	copyJob := *job
	return &copyJob, nil
}
