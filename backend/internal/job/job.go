package job

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Status 任务状态。
type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
)

// CanTransit 校验状态流转合法性。
func CanTransit(from, to Status) bool {
	switch from {
	case StatusPending:
		return to == StatusProcessing
	case StatusProcessing:
		return to == StatusDone || to == StatusFailed
	case StatusFailed:
		return to == StatusPending // 重试
	default:
		return false
	}
}

// Job 异步任务模型。
type Job struct {
	ID          string          `gorm:"primaryKey;type:varchar(64)" json:"id"`
	Type        string          `gorm:"type:varchar(32);not null;index" json:"type"`
	Status      Status          `gorm:"type:varchar(16);not null;index" json:"status"`
	Payload     json.RawMessage `gorm:"type:json" json:"payload"`
	TraceID     string          `gorm:"type:varchar(32)" json:"trace_id"`
	Result      string          `gorm:"type:varchar(1024)" json:"result"`
	Attempts    int             `gorm:"not null;default:0" json:"attempts"`
	MaxAttempts int             `gorm:"not null;default:3" json:"max_attempts"`
	CreatedAt   time.Time       `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"not null" json:"updated_at"`
}

func (Job) TableName() string { return "jobs" }

// NewJobID 生成 16 位 hex 任务 ID。
func NewJobID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Transit 执行状态流转并返回新状态，同时更新 Attempts。
// 若流转非法，返回错误。
func (j *Job) Transit(to Status) error {
	if !CanTransit(j.Status, to) {
		return fmt.Errorf("illegal status transition: %s -> %s", j.Status, to)
	}
	j.Status = to
	j.UpdatedAt = time.Now()
	return nil
}

// TransitToFailed 转 failed 并递增 attempt。
func (j *Job) TransitToFailed() error {
	if err := j.Transit(StatusFailed); err != nil {
		return err
	}
	j.Attempts++
	return nil
}

// TransitToRetry 重试：failed → pending（attempt 不变，由 Dequeue 重新触发）。
func (j *Job) TransitToRetry() error {
	return j.Transit(StatusPending)
}
