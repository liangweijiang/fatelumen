package job

import "context"

// Job 是一个可执行的异步任务（如「生成完整报告」）。
type Job struct {
	Type    string // "generate_report"
	Payload []byte // JSON 序列化的任务参数（如 report_id）
}

// Handler 处理某类 Job。
type Handler func(ctx context.Context, payload []byte) error

// JobQueue 抽象异步任务调度（P4 落地）。
// MVP 用 goroutine + worker pool 实现；量大时换 Asynq(Redis) 实现，
// report 生成逻辑零改动——只换 main 里注入的实现。
type JobQueue interface {
	// Register 注册某类任务的处理器（启动时调用）
	Register(jobType string, h Handler)
	// Enqueue 入队一个任务（异步执行，立即返回）
	Enqueue(ctx context.Context, job Job) error
	// Start 启动 worker（阻塞或后台）
	Start(ctx context.Context) error
	// Shutdown 优雅停机：等待在途任务完成
	Shutdown(ctx context.Context) error
}
