package job

import (
	"context"
	"fmt"
	"sync"
)

// JobHandler 任务处理器接口。实现者执行具体业务逻辑，返回 result 和 error。
type JobHandler interface {
	Handle(ctx context.Context, job *Job) (result string, err error)
}

// HandlerRegistry 按 job.Type 注册和查找 handler。
type HandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string]JobHandler
}

// NewHandlerRegistry 创建 HandlerRegistry。
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{handlers: make(map[string]JobHandler)}
}

// Register 注册 handler。同一 type 重复注册会覆盖。
func (r *HandlerRegistry) Register(jobType string, h JobHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[jobType] = h
}

// Get 按 job.Type 查找 handler。找不到返回错误。
func (r *HandlerRegistry) Get(jobType string) (JobHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[jobType]
	if !ok {
		return nil, fmt.Errorf("no handler registered for job type: %s", jobType)
	}
	return h, nil
}
