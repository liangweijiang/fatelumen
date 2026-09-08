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

func (r *FullReportCleanupJobRepo) GetPDFStorageKey(ctx context.Context, reportID uint64) (string, error) {
	var report model.FullReport
	if err := r.db.WithContext(ctx).Select("id").Where("id = ? AND status = ?", reportID, model.FullReportStatusDeleting).First(&report).Error; err != nil {
		return "", err
	}
	var result model.FullReportResult
	err := r.db.WithContext(ctx).Select("pdf_storage_key").Where("report_id = ?", reportID).First(&result).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return result.PDFStorageKey, err
}

func (r *FullReportCleanupJobRepo) AdvanceObjectDeleted(ctx context.Context, taskID uint64, now time.Time) error {
	return r.advance(ctx, taskID, model.FullReportCleanupStageQueued, model.FullReportCleanupStageObjectDeleted, now, nil)
}

// DeleteDatabaseStage removes one dependency layer and advances the durable
// cursor in the same transaction. Replaying a stage after rollback is safe.
func (r *FullReportCleanupJobRepo) DeleteDatabaseStage(ctx context.Context, taskID uint64, from, to string, now time.Time) error {
	return r.advance(ctx, taskID, from, to, now, func(tx *gorm.DB, reportID uint64) error {
		switch to {
		case model.FullReportCleanupStageAttemptsGone:
			var ids []uint64
			if err := tx.Model(&model.FullReportAttempt{}).Where("report_id = ?", reportID).Pluck("id", &ids).Error; err != nil {
				return err
			}
			if len(ids) > 0 {
				if err := tx.Where("attempt_id IN ?", ids).Delete(&model.FullReportAttemptPayload{}).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&model.FullReportChapter{}).Where("report_id = ?", reportID).Update("selected_attempt_id", nil).Error; err != nil {
				return err
			}
			return tx.Where("report_id = ?", reportID).Delete(&model.FullReportAttempt{}).Error
		case model.FullReportCleanupStageValidationsGone:
			var ids []uint64
			if err := tx.Model(&model.FullReportValidationRun{}).Where("report_id = ?", reportID).Pluck("id", &ids).Error; err != nil {
				return err
			}
			if len(ids) > 0 {
				if err := tx.Where("validation_run_id IN ?", ids).Delete(&model.FullReportValidationPayload{}).Error; err != nil {
					return err
				}
			}
			return tx.Where("report_id = ?", reportID).Delete(&model.FullReportValidationRun{}).Error
		case model.FullReportCleanupStageChaptersGone:
			var ids []uint64
			if err := tx.Model(&model.FullReportChapter{}).Where("report_id = ?", reportID).Pluck("id", &ids).Error; err != nil {
				return err
			}
			if len(ids) > 0 {
				if err := tx.Where("chapter_id IN ?", ids).Delete(&model.FullReportChapterPayload{}).Error; err != nil {
					return err
				}
			}
			return tx.Where("report_id = ?", reportID).Delete(&model.FullReportChapter{}).Error
		case model.FullReportCleanupStageExecutionGone:
			var ids []uint64
			if err := tx.Model(&model.FullReportExecutionSnapshot{}).Where("report_id = ?", reportID).Pluck("id", &ids).Error; err != nil {
				return err
			}
			if len(ids) > 0 {
				if err := tx.Where("snapshot_id IN ?", ids).Delete(&model.FullReportExecutionPayload{}).Error; err != nil {
					return err
				}
			}
			return tx.Where("report_id = ?", reportID).Delete(&model.FullReportExecutionSnapshot{}).Error
		case model.FullReportCleanupStageResultGone:
			return tx.Where("report_id = ?", reportID).Delete(&model.FullReportResult{}).Error
		case model.FullReportCleanupStageRenderJobsGone:
			return tx.Where("report_id = ?", reportID).Delete(&model.FullReportRenderJob{}).Error
		case model.FullReportCleanupStageReportGone:
			res := tx.Where("id = ? AND status = ?", reportID, model.FullReportStatusDeleting).Delete(&model.FullReport{})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected > 1 {
				return ErrFullReportImmutableWrite
			}
			return nil
		default:
			return errors.New("unsupported full report cleanup stage")
		}
	})
}

func (r *FullReportCleanupJobRepo) advance(ctx context.Context, taskID uint64, from, to string, now time.Time, action func(*gorm.DB, uint64) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.FullReportCleanupJob
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, taskID).Error; err != nil {
			return err
		}
		if task.Status != model.FullReportCleanupJobStatusRunning || task.Stage != from {
			return ErrFullReportCleanupJobNotClaimed
		}
		if action != nil {
			if err := action(tx, task.ReportID); err != nil {
				return err
			}
		}
		return tx.Model(&model.FullReportCleanupJob{}).Where("id = ? AND status = ? AND stage = ?", taskID, model.FullReportCleanupJobStatusRunning, from).Updates(map[string]any{"stage": to, "updated_at": now}).Error
	})
}

func (r *FullReportCleanupJobRepo) Succeed(ctx context.Context, id uint64, now time.Time) error {
	res := r.db.WithContext(ctx).Model(&model.FullReportCleanupJob{}).
		Where("id = ? AND status = ? AND stage = ?", id, model.FullReportCleanupJobStatusRunning, model.FullReportCleanupStageReportGone).
		Updates(map[string]any{"status": model.FullReportCleanupJobStatusSucceeded, "finished_at": now, "updated_at": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return ErrFullReportCleanupJobNotClaimed
	}
	return nil
}

func (r *FullReportCleanupJobRepo) RecordFailure(ctx context.Context, id uint64, code, summary string, now time.Time) (bool, error) {
	terminal := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.FullReportCleanupJob
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, id).Error; err != nil {
			return err
		}
		if task.Status != model.FullReportCleanupJobStatusRunning {
			return ErrFullReportCleanupJobNotClaimed
		}
		updates := map[string]any{"status": model.FullReportCleanupJobStatusQueued, "error_code": code, "error_summary": summary, "updated_at": now}
		if task.AttemptCount >= task.MaxAttempts {
			terminal = true
			updates["status"] = model.FullReportCleanupJobStatusFailed
			updates["finished_at"] = now
		}
		return tx.Model(&model.FullReportCleanupJob{}).Where("id = ? AND status = ?", id, model.FullReportCleanupJobStatusRunning).Updates(updates).Error
	})
	return terminal, err
}

func (r *FullReportCleanupJobRepo) RecoverStale(ctx context.Context, cutoff, now time.Time) (int, int, error) {
	requeued, failed := 0, 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tasks []model.FullReportCleanupJob
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("status = ? AND updated_at < ?", model.FullReportCleanupJobStatusRunning, cutoff).Find(&tasks).Error; err != nil {
			return err
		}
		for _, task := range tasks {
			updates := map[string]any{"status": model.FullReportCleanupJobStatusQueued, "error_code": "cleanup_interrupted", "error_summary": "报告清理执行中断，已由恢复流程接管", "updated_at": now}
			if task.AttemptCount >= task.MaxAttempts {
				updates["status"] = model.FullReportCleanupJobStatusFailed
				updates["finished_at"] = now
				failed++
			} else {
				requeued++
			}
			if err := tx.Model(&model.FullReportCleanupJob{}).Where("id = ? AND status = ?", task.ID, model.FullReportCleanupJobStatusRunning).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return requeued, failed, err
}

func (r *FullReportCleanupJobRepo) ListQueued(ctx context.Context, limit int) ([]model.FullReportCleanupJob, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	var tasks []model.FullReportCleanupJob
	err := r.db.WithContext(ctx).Where("status = ?", model.FullReportCleanupJobStatusQueued).Order("created_at ASC, id ASC").Limit(limit).Find(&tasks).Error
	return tasks, err
}
