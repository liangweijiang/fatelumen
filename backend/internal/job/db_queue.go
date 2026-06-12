package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func (q *DBQueue) ReclaimStale(ctx context.Context, staleThreshold time.Duration) (int, int, error) {
	cutoff := time.Now().Add(-staleThreshold)
	log := logger.FromCtx(ctx)

	var reclaimed, failed int
	err := q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var jobs []Job
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ? AND updated_at < ?", StatusProcessing, cutoff).
			Find(&jobs).Error; err != nil {
			log.Error("reclaim stale query failed", "err", err)
			return err
		}

		for _, job := range jobs {
			newAttempts := job.Attempts + 1
			if job.Attempts < job.MaxAttempts {
				result := fmt.Sprintf("reclaimed stale processing job (attempt %d/%d)", newAttempts, job.MaxAttempts)
				if err := tx.Model(&Job{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
					"status":     StatusPending,
					"attempts":   newAttempts,
					"result":     result,
					"updated_at": time.Now(),
				}).Error; err != nil {
					log.Error("reclaim stale update failed", "err", err, "job_id", job.ID)
					return err
				}
				reclaimed++
				log.Info("reclaimed stale processing job",
					"job_id", job.ID,
					"job_type", job.Type,
					"attempts", newAttempts,
					"max_attempts", job.MaxAttempts,
				)
			} else {
				// TODO P2: 触发退款（调用 payment.PaymentProvider.Refund）并通知用户
				result := fmt.Sprintf("stale job exceeded max attempts (%d/%d), requires manual refund for order: check payment events linked to this job", job.Attempts, job.MaxAttempts)
				if err := tx.Model(&Job{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
					"status":     StatusFailed,
					"result":     result,
					"updated_at": time.Now(),
				}).Error; err != nil {
					log.Error("reclaim stale final-fail update failed", "err", err, "job_id", job.ID)
					return err
				}
				failed++
				log.Error("stale job permanently failed — manual refund required",
					"job_id", job.ID,
					"job_type", job.Type,
					"attempts", job.Attempts,
					"max_attempts", job.MaxAttempts,
				)
			}
		}
		return nil
	})

	if err != nil {
		return 0, 0, err
	}

	if reclaimed > 0 || failed > 0 {
		log.Info("reclaim stale completed",
			"reclaimed", reclaimed,
			"failed", failed,
			"stale_threshold", staleThreshold.String(),
		)
	}
	return reclaimed, failed, nil
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
