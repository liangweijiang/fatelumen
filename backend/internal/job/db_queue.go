package job

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"fatelumen/backend/internal/pkg/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DBQueue 基于 GORM 的持久化队列实现。
// Dequeue 用 FOR UPDATE SKIP LOCKED 保证多 worker 不重复消费。
type DBQueue struct {
	db *gorm.DB
}

func NewDBQueue(db *gorm.DB) *DBQueue {
	return &DBQueue{db: db}
}

func (q *DBQueue) Enqueue(ctx context.Context, job *Job) error {
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
	now := time.Now()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = now
	}
	if job.Payload == nil {
		job.Payload = json.RawMessage("{}")
	}

	if err := q.db.WithContext(ctx).Create(job).Error; err != nil {
		return err
	}
	return nil
}

func (q *DBQueue) Dequeue(ctx context.Context) (*Job, error) {
	var job Job
	err := q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "SKIP LOCKED",
		}).Where("status = ?", StatusPending).
			Order("created_at ASC").
			First(&job).Error
		if err != nil {
			return err
		}

		return tx.Model(&job).Updates(map[string]interface{}{
			"status":     StatusProcessing,
			"updated_at": time.Now(),
		}).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

func (q *DBQueue) UpdateStatus(ctx context.Context, id string, status Status, result string) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job Job
		if err := tx.Where("id = ?", id).First(&job).Error; err != nil {
			return err
		}

		if !CanTransit(job.Status, status) {
			logger.FromCtx(ctx).Error("illegal job status transition",
				"job_id", id,
				"from", string(job.Status),
				"to", string(status),
			)
			return nil
		}

		updates := map[string]interface{}{
			"status":     status,
			"result":     result,
			"updated_at": time.Now(),
		}
		if status == StatusFailed {
			updates["attempts"] = job.Attempts + 1
		}

		return tx.Model(&Job{}).Where("id = ?", id).Updates(updates).Error
	})
}

func (q *DBQueue) Get(ctx context.Context, id string) (*Job, error) {
	var job Job
	if err := q.db.WithContext(ctx).Where("id = ?", id).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}
