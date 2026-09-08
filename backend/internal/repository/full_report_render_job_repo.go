package repository

import (
	"context"
	"errors"
	"time"

	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrFullReportRenderJobNotClaimed = errors.New("full report render job was not claimed")

type FullReportRenderJobRepo struct{ db *gorm.DB }

func NewFullReportRenderJobRepo(db *gorm.DB) *FullReportRenderJobRepo {
	return &FullReportRenderJobRepo{db: db}
}

// Ensure returns the only task for a report and render version. Concurrent
// callers converge on the unique row instead of creating duplicate work.
func (r *FullReportRenderJobRepo) Ensure(ctx context.Context, reportID uint64, renderVersion string, maxAttempts uint16, now time.Time) (*model.FullReportRenderJob, bool, error) {
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	row := &model.FullReportRenderJob{ReportID: reportID, RenderVersion: renderVersion, Status: model.FullReportRenderJobStatusQueued, MaxAttempts: maxAttempts, CreatedAt: now, UpdatedAt: now}
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "report_id"}, {Name: "render_version"}}, DoNothing: true}).Create(row)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 1 {
		return row, true, nil
	}
	if err := r.db.WithContext(ctx).Where("report_id = ? AND render_version = ?", reportID, renderVersion).First(row).Error; err != nil {
		return nil, false, err
	}
	return row, false, nil
}

// Claim atomically grants one delivery the right to render and increments the
// durable attempt number. Duplicate delivery receives Err...NotClaimed.
func (r *FullReportRenderJobRepo) Claim(ctx context.Context, id uint64, now time.Time) (*model.FullReportRenderJob, error) {
	result := r.db.WithContext(ctx).Model(&model.FullReportRenderJob{}).
		Where("id = ? AND status = ? AND attempt_count < max_attempts", id, model.FullReportRenderJobStatusQueued).
		Updates(map[string]any{
			"status":        model.FullReportRenderJobStatusRunning,
			"attempt_count": gorm.Expr("attempt_count + 1"),
			"started_at":    now,
			"finished_at":   nil,
			"error_code":    "",
			"error_summary": "",
			"updated_at":    now,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, ErrFullReportRenderJobNotClaimed
	}
	var row model.FullReportRenderJob
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *FullReportRenderJobRepo) Succeed(ctx context.Context, id uint64, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&model.FullReportRenderJob{}).
		Where("id = ? AND status = ?", id, model.FullReportRenderJobStatusRunning).
		Updates(map[string]any{"status": model.FullReportRenderJobStatusSucceeded, "finished_at": now, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrFullReportRenderJobNotClaimed
	}
	return nil
}

// RecordFailure requeues recoverable work while budget remains. It returns
// true only when the task has reached its terminal failed state.
func (r *FullReportRenderJobRepo) RecordFailure(ctx context.Context, id uint64, code, summary string, now time.Time) (bool, error) {
	terminal := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.FullReportRenderJob
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
			return err
		}
		if row.Status != model.FullReportRenderJobStatusRunning {
			return ErrFullReportRenderJobNotClaimed
		}
		status := model.FullReportRenderJobStatusQueued
		updates := map[string]any{"status": status, "error_code": code, "error_summary": summary, "updated_at": now}
		if row.AttemptCount >= row.MaxAttempts {
			terminal = true
			updates["status"] = model.FullReportRenderJobStatusFailed
			updates["finished_at"] = now
		}
		return tx.Model(&model.FullReportRenderJob{}).Where("id = ? AND status = ?", id, model.FullReportRenderJobStatusRunning).Updates(updates).Error
	})
	return terminal, err
}

// RecoverStale resets interrupted work to queued, or permanently fails it if
// all attempts were already consumed. Returned rows are safe to enqueue again.
func (r *FullReportRenderJobRepo) RecoverStale(ctx context.Context, cutoff, now time.Time) ([]model.FullReportRenderJob, int, error) {
	recovered := make([]model.FullReportRenderJob, 0)
	failed := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []model.FullReportRenderJob
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("status = ? AND updated_at < ?", model.FullReportRenderJobStatusRunning, cutoff).Find(&rows).Error; err != nil {
			return err
		}
		for i := range rows {
			row := &rows[i]
			updates := map[string]any{"error_code": "render_interrupted", "error_summary": "PDF任务执行中断，已由恢复流程接管", "updated_at": now}
			if row.AttemptCount >= row.MaxAttempts {
				updates["status"] = model.FullReportRenderJobStatusFailed
				updates["finished_at"] = now
				failed++
			} else {
				updates["status"] = model.FullReportRenderJobStatusQueued
				row.Status = model.FullReportRenderJobStatusQueued
				row.ErrorCode = "render_interrupted"
				row.ErrorSummary = "PDF任务执行中断，已由恢复流程接管"
				row.UpdatedAt = now
				recovered = append(recovered, *row)
			}
			if err := tx.Model(&model.FullReportRenderJob{}).Where("id = ? AND status = ?", row.ID, model.FullReportRenderJobStatusRunning).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return recovered, failed, err
}

func (r *FullReportRenderJobRepo) Get(ctx context.Context, id uint64) (*model.FullReportRenderJob, error) {
	var row model.FullReportRenderJob
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *FullReportRenderJobRepo) GetByReportID(ctx context.Context, reportID uint64) (*model.FullReportRenderJob, error) {
	var row model.FullReportRenderJob
	if err := r.db.WithContext(ctx).Where("report_id = ?", reportID).Order("id DESC").First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *FullReportRenderJobRepo) ListQueued(ctx context.Context) ([]model.FullReportRenderJob, error) {
	var rows []model.FullReportRenderJob
	err := r.db.WithContext(ctx).Where("status = ?", model.FullReportRenderJobStatusQueued).Order("created_at ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *FullReportRenderJobRepo) ListFailedUnfinishedReports(ctx context.Context) ([]model.FullReportRenderJob, error) {
	var rows []model.FullReportRenderJob
	err := r.db.WithContext(ctx).Table("full_report_render_jobs AS j").Select("j.*").
		Joins("JOIN full_reports AS r ON r.id = j.report_id").
		Where("j.status = ? AND r.status = ?", model.FullReportRenderJobStatusFailed, model.FullReportStatusRendering).
		Find(&rows).Error
	return rows, err
}
