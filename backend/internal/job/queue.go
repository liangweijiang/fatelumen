package job

import (
	"context"
	"time"
)

// Handler 处理某类 Job（Phase 4 后续步骤实现 worker 时使用）。
type Handler func(ctx context.Context, job *Job) error

// Queue 异步任务队列接口（P4 状态机核心，P5 依赖接口）。
type Queue interface {
	// Enqueue 入队。自动从 ctx 取 trace_id 写入 job.TraceID。
	Enqueue(ctx context.Context, job *Job) error

	// Dequeue 取一个 pending 任务并原子置为 processing。
	// 无 pending 任务时返回 nil, nil。
	Dequeue(ctx context.Context) (*Job, error)

	// UpdateStatus 更新状态并写入 result。内部校验状态流转合法性。
	UpdateStatus(ctx context.Context, id string, status Status, result string) error

	// Get 按 ID 获取任务。
	Get(ctx context.Context, id string) (*Job, error)

	// ReclaimStale 回收超时的 processing 任务（进程崩溃后孤儿 job 补偿）。
	// staleThreshold: 超过该阈值的 processing 任务视为孤儿。
	// 对每条：attempts < maxAttempts 则重置 pending 重新调度；
	// attempts >= maxAttempts 则置为 failed 终态。
	// 返回 reclaimed（重新调度）与 failed（永久失败）计数。
	ReclaimStale(ctx context.Context, staleThreshold time.Duration) (reclaimed int, failed int, err error)
}
