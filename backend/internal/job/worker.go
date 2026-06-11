package job

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fatelumen/backend/internal/pkg/logger"
)

const (
	defaultPollInterval = time.Second
	defaultWorkerCount  = 3
)

// Worker 异步任务消费者。
type Worker struct {
	q          Queue
	registry   *HandlerRegistry
	interval   time.Duration
	numWorkers int
	wg         sync.WaitGroup
	cancel     context.CancelFunc
}

// NewWorker 创建 Worker。interval 为 0 时用默认 1s，workers 为 0 时用默认 3。
func NewWorker(q Queue, registry *HandlerRegistry, interval time.Duration, workers int) *Worker {
	if interval <= 0 {
		interval = defaultPollInterval
	}
	if workers <= 0 {
		workers = defaultWorkerCount
	}
	return &Worker{
		q:          q,
		registry:   registry,
		interval:   interval,
		numWorkers: workers,
	}
}

// Start 启动 N 个 goroutine 消费队列。
func (w *Worker) Start(ctx context.Context) {
	ctx, w.cancel = context.WithCancel(ctx)
	for i := 0; i < w.numWorkers; i++ {
		w.wg.Add(1)
		go w.loop(ctx, i)
	}
}

// Stop 优雅停机：取消 ctx，等待所有 in-flight 任务跑完。
func (w *Worker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
}

func (w *Worker) loop(ctx context.Context, idx int) {
	defer w.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, err := w.q.Dequeue(ctx)
		if err != nil {
			logger.FromCtx(ctx).Error("worker dequeue failed", "worker_idx", idx, "err", err)
			continue
		}
		if job == nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.interval):
			}
			continue
		}

		w.process(ctx, idx, job)
	}
}

func (w *Worker) process(parentCtx context.Context, idx int, job *Job) {
	// 重建带 trace_id 的 ctx，续上异步链路
	jobCtx := logger.WithTraceID(parentCtx, job.TraceID)

	handler, err := w.registry.Get(job.Type)
	if err != nil {
		logger.FromCtx(jobCtx).Error("job handler not found",
			"job_id", job.ID, "job_type", job.Type, "err", err)
		if err := w.q.UpdateStatus(jobCtx, job.ID, StatusFailed, err.Error()); err != nil {
			logger.FromCtx(jobCtx).Error("failed to update job status after handler-not-found",
				"job_id", job.ID, "status", StatusFailed, "err", err)
		}
		return
	}

	result, err := w.invokeHandler(jobCtx, handler, job)
	if err == nil {
		logger.FromCtx(jobCtx).Info("job completed", "job_id", job.ID)
		if err := w.q.UpdateStatus(jobCtx, job.ID, StatusDone, result); err != nil {
			logger.FromCtx(jobCtx).Error("failed to update job status to done",
				"job_id", job.ID, "status", StatusDone, "err", err)
		}
		return
	}

	// 失败路径：判断重试
	currentAttempt := job.Attempts + 1
	if currentAttempt < job.MaxAttempts {
		logger.FromCtx(jobCtx).Warn("job failed, will retry",
			"job_id", job.ID, "attempt", currentAttempt, "max_attempts", job.MaxAttempts, "err", err)
		// 两步状态流转：processing → failed → pending（状态机不支持 processing 直接到 pending）
		if err := w.q.UpdateStatus(jobCtx, job.ID, StatusFailed, err.Error()); err != nil {
			logger.FromCtx(jobCtx).Error("failed to update job status to failed before retry",
				"job_id", job.ID, "status", StatusFailed, "err", err)
		} else if err := w.q.UpdateStatus(jobCtx, job.ID, StatusPending, ""); err != nil {
			logger.FromCtx(jobCtx).Error("failed to update job status to pending for retry",
				"job_id", job.ID, "status", StatusPending, "err", err)
		}
	} else {
		logger.FromCtx(jobCtx).Error("job permanently failed",
			"job_id", job.ID, "attempts", currentAttempt, "err", err)
		if err := w.q.UpdateStatus(jobCtx, job.ID, StatusFailed, err.Error()); err != nil {
			logger.FromCtx(jobCtx).Error("failed to update job status to failed",
				"job_id", job.ID, "status", StatusFailed, "err", err)
		}
	}
}

// invokeHandler 执行 handler，defer/recover 兜底 panic。
func (w *Worker) invokeHandler(ctx context.Context, handler JobHandler, job *Job) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handler panic: %v", r)
			result = ""
		}
	}()
	return handler.Handle(ctx, job)
}
