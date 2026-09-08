package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
	"fatelumen/backend/internal/storage"
)

const FullReportCleanupJobType = "full_report_cleanup_v1"

type fullReportCleanupPayload struct {
	CleanupJobID uint64 `json:"cleanup_job_id"`
}

type FullReportCleanupRunner struct {
	tasks   *repository.FullReportCleanupJobRepo
	deleter storage.Deleter
}

func NewFullReportCleanupRunner(tasks *repository.FullReportCleanupJobRepo, deleter storage.Deleter) *FullReportCleanupRunner {
	return &FullReportCleanupRunner{tasks: tasks, deleter: deleter}
}

func (r *FullReportCleanupRunner) Run(ctx context.Context, task *model.FullReportCleanupJob) error {
	if r == nil || r.tasks == nil || r.deleter == nil || task == nil {
		return errors.New("full report cleanup runner is not configured")
	}
	now := func() time.Time { return time.Now().UTC() }
	stage := task.Stage
	if stage == model.FullReportCleanupStageQueued {
		key, err := r.tasks.GetPDFStorageKey(ctx, task.ReportID)
		if err != nil {
			return fmt.Errorf("load report PDF key: %w", err)
		}
		if key != "" {
			if err := r.deleter.Delete(ctx, key); err != nil {
				logger.FromCtx(ctx).Error("delete expired report PDF failed", "err", err, "report_id", task.ReportID, "key", key)
				return fmt.Errorf("delete report PDF: %w", err)
			}
		}
		if err := r.tasks.AdvanceObjectDeleted(ctx, task.ID, now()); err != nil {
			return fmt.Errorf("advance report cleanup object stage: %w", err)
		}
		stage = model.FullReportCleanupStageObjectDeleted
	}
	steps := []struct{ from, to string }{
		{model.FullReportCleanupStageObjectDeleted, model.FullReportCleanupStageAttemptsGone},
		{model.FullReportCleanupStageAttemptsGone, model.FullReportCleanupStageValidationsGone},
		{model.FullReportCleanupStageValidationsGone, model.FullReportCleanupStageChaptersGone},
		{model.FullReportCleanupStageChaptersGone, model.FullReportCleanupStageExecutionGone},
		{model.FullReportCleanupStageExecutionGone, model.FullReportCleanupStageResultGone},
		{model.FullReportCleanupStageResultGone, model.FullReportCleanupStageRenderJobsGone},
		{model.FullReportCleanupStageRenderJobsGone, model.FullReportCleanupStageReportGone},
	}
	for _, step := range steps {
		if stage != step.from {
			continue
		}
		if err := r.tasks.DeleteDatabaseStage(ctx, task.ID, step.from, step.to, now()); err != nil {
			return fmt.Errorf("delete report cleanup stage %s: %w", step.to, err)
		}
		stage = step.to
	}
	if stage != model.FullReportCleanupStageReportGone {
		return fmt.Errorf("unknown report cleanup stage %q", stage)
	}
	return r.tasks.Succeed(ctx, task.ID, now())
}

type FullReportCleanupJobHandler struct {
	tasks  *repository.FullReportCleanupJobRepo
	runner *FullReportCleanupRunner
}

func NewFullReportCleanupJobHandler(tasks *repository.FullReportCleanupJobRepo, runner *FullReportCleanupRunner) *FullReportCleanupJobHandler {
	return &FullReportCleanupJobHandler{tasks: tasks, runner: runner}
}

func (h *FullReportCleanupJobHandler) Handle(ctx context.Context, queued *job.Job) (string, error) {
	var payload fullReportCleanupPayload
	if err := json.Unmarshal(queued.Payload, &payload); err != nil || payload.CleanupJobID == 0 {
		logger.FromCtx(ctx).Error("report cleanup payload invalid", "err", err, "job_id", queued.ID)
		return "", errors.New("invalid report cleanup payload")
	}
	task, err := h.tasks.Claim(ctx, payload.CleanupJobID, time.Now().UTC())
	if errors.Is(err, repository.ErrFullReportCleanupJobNotClaimed) {
		current, getErr := h.tasks.Get(ctx, payload.CleanupJobID)
		if getErr != nil {
			logger.FromCtx(ctx).Error("load unclaimed report cleanup task failed", "err", getErr, "cleanup_job_id", payload.CleanupJobID)
			return "", getErr
		}
		return current.Status, nil
	}
	if err != nil {
		logger.FromCtx(ctx).Error("claim report cleanup task failed", "err", err, "cleanup_job_id", payload.CleanupJobID)
		return "", err
	}
	if err := h.runner.Run(ctx, task); err != nil {
		finished := time.Now().UTC()
		terminal, saveErr := h.tasks.RecordFailure(context.WithoutCancel(ctx), task.ID, "cleanup_failed", truncateError(err), finished)
		if saveErr != nil {
			logger.FromCtx(ctx).Error("record report cleanup failure failed", "err", saveErr, "cleanup_job_id", task.ID, "report_id", task.ReportID)
			return "", saveErr
		}
		logger.FromCtx(ctx).Warn("report cleanup attempt failed", "err", err, "cleanup_job_id", task.ID, "report_id", task.ReportID, "attempt", task.AttemptCount, "max_attempts", task.MaxAttempts, "terminal", terminal)
		return "", err
	}
	return model.FullReportCleanupJobStatusSucceeded, nil
}

func enqueueFullReportCleanup(ctx context.Context, queue job.Queue, task model.FullReportCleanupJob) error {
	payload, err := json.Marshal(fullReportCleanupPayload{CleanupJobID: task.ID})
	if err != nil {
		return err
	}
	return queue.Enqueue(ctx, &job.Job{Type: FullReportCleanupJobType, Lane: job.LaneReportCleanup, Payload: payload, MaxAttempts: 1})
}

// StartFullReportCleanup schedules expired reports and recovers interrupted
// cleanup work. The business task owns retries; queue deliveries are disposable.
func StartFullReportCleanup(ctx context.Context, tasks *repository.FullReportCleanupJobRepo, queue job.Queue, staleAfter, interval time.Duration) {
	if staleAfter <= 0 {
		staleAfter = 10 * time.Minute
	}
	if interval <= 0 {
		interval = time.Hour
	}
	run := func() {
		now := time.Now().UTC()
		if _, _, err := tasks.RecoverStale(ctx, now.Add(-staleAfter), now); err != nil {
			logger.FromCtx(ctx).Error("recover report cleanup tasks failed", "err", err)
			return
		}
		if _, err := tasks.ScheduleExpired(ctx, now, 100, 3, now); err != nil {
			logger.FromCtx(ctx).Error("schedule expired reports failed", "err", err)
			return
		}
		queued, err := tasks.ListQueued(ctx, 500)
		if err != nil {
			logger.FromCtx(ctx).Error("list queued report cleanup tasks failed", "err", err)
			return
		}
		for _, task := range queued {
			if err := enqueueFullReportCleanup(ctx, queue, task); err != nil {
				logger.FromCtx(ctx).Error("enqueue report cleanup task failed", "err", err, "cleanup_job_id", task.ID, "report_id", task.ReportID)
				return
			}
		}
	}
	run()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}
