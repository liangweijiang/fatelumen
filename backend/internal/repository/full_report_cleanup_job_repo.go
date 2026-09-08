package repository

import (
	"context"
	"errors"
	"time"

	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrFullReportCleanupJobNotClaimed = errors.New("full report cleanup job was not claimed")

type FullReportCleanupJobRepo struct{ db *gorm.DB }

func NewFullReportCleanupJobRepo(db *gorm.DB) *FullReportCleanupJobRepo {
	return &FullReportCleanupJobRepo{db: db}
}

// ScheduleExpired atomically hides a bounded batch of expired terminal reports
// behind the deleting state and creates one durable cleanup task per report.
// Payload deletion is deliberately handled by the next implementation stage.
func (r *FullReportCleanupJobRepo) ScheduleExpired(ctx context.Context, expiredAt time.Time, limit int, maxAttempts uint16, now time.Time) ([]model.FullReportCleanupJob, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	tasks := make([]model.FullReportCleanupJob, 0, limit)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reports []model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Select("id").
			Where("expires_at <= ? AND status IN ?", expiredAt, []string{model.FullReportStatusCompleted, model.FullReportStatusFailed}).
			Order("expires_at ASC, status ASC, id ASC").Limit(limit).Find(&reports).Error; err != nil {
			return err
		}
		for _, report := range reports {
			task := model.FullReportCleanupJob{
				ReportID: report.ID, Status: model.FullReportCleanupJobStatusQueued, Stage: model.FullReportCleanupStageQueued,
				MaxAttempts: maxAttempts, CreatedAt: now, UpdatedAt: now,
			}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "report_id"}}, DoNothing: true}).Create(&task).Error; err != nil {
				return err
			}
			res := tx.Model(&model.FullReport{}).
				Where("id = ? AND status IN ?", report.ID, []string{model.FullReportStatusCompleted, model.FullReportStatusFailed}).
				Updates(map[string]any{"status": model.FullReportStatusDeleting, "current_stage": model.FullReportStatusDeleting, "updated_at": now})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected != 1 {
				return ErrFullReportImmutableWrite
			}
			if task.ID == 0 {
				if err := tx.Where("report_id = ?", report.ID).First(&task).Error; err != nil {
					return err
				}
			}
			tasks = append(tasks, task)
		}
		return nil
	})
	return tasks, err
}

// Claim grants exactly one worker ownership of a queued cleanup task.
func (r *FullReportCleanupJobRepo) Claim(ctx context.Context, id uint64, now time.Time) (*model.FullReportCleanupJob, error) {
	res := r.db.WithContext(ctx).Model(&model.FullReportCleanupJob{}).
		Where("id = ? AND status = ? AND attempt_count < max_attempts", id, model.FullReportCleanupJobStatusQueued).
		Updates(map[string]any{
			"status": model.FullReportCleanupJobStatusRunning, "attempt_count": gorm.Expr("attempt_count + 1"),
			"started_at": now, "finished_at": nil, "error_code": "", "error_summary": "", "updated_at": now,
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected != 1 {
		return nil, ErrFullReportCleanupJobNotClaimed
	}
	var task model.FullReportCleanupJob
	if err := r.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *FullReportCleanupJobRepo) Get(ctx context.Context, id uint64) (*model.FullReportCleanupJob, error) {
	var task model.FullReportCleanupJob
	if err := r.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}
