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
)

const FullReportPDFJobType = "full_report_pdf_v1"

type FullReportPDFTaskRunner interface {
	Run(ctx context.Context, task *model.FullReportRenderJob) error
}

type fullReportPDFJobPayload struct {
	RenderJobID uint64 `json:"render_job_id"`
}

type FullReportPDFJobHandler struct {
	tasks    *repository.FullReportRenderJobRepo
	reports  *repository.FullReportRepo
	runner   FullReportPDFTaskRunner
	gate     chan struct{}
	outcomes FullReportOutcomeNotifier
}

func NewFullReportPDFJobHandler(tasks *repository.FullReportRenderJobRepo, runner FullReportPDFTaskRunner, concurrency int, reports *repository.FullReportRepo, outcomes ...FullReportOutcomeNotifier) *FullReportPDFJobHandler {
	if concurrency < 1 {
		concurrency = 1
	}
	var outcomeNotifier FullReportOutcomeNotifier
	if len(outcomes) > 0 {
		outcomeNotifier = outcomes[0]
	}
	return &FullReportPDFJobHandler{tasks: tasks, reports: reports, runner: runner, gate: make(chan struct{}, concurrency), outcomes: outcomeNotifier}
}

func (h *FullReportPDFJobHandler) Handle(ctx context.Context, queued *job.Job) (string, error) {
	var payload fullReportPDFJobPayload
	if err := json.Unmarshal(queued.Payload, &payload); err != nil || payload.RenderJobID == 0 {
		logger.FromCtx(ctx).Error("pdf render job payload invalid", "err", err, "job_id", queued.ID)
		return "", errors.New("invalid PDF task payload")
	}
	select {
	case h.gate <- struct{}{}:
		defer func() { <-h.gate }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	now := time.Now().UTC()
	task, err := h.tasks.Claim(ctx, payload.RenderJobID, now)
	if errors.Is(err, repository.ErrFullReportRenderJobNotClaimed) {
		current, getErr := h.tasks.Get(ctx, payload.RenderJobID)
		if getErr != nil {
			logger.FromCtx(ctx).Error("load unclaimed pdf render job failed", "err", getErr, "render_job_id", payload.RenderJobID)
			return "", getErr
		}
		// Another delivery owns it or it already reached a terminal state.
		return current.Status, nil
	}
	if err != nil {
		logger.FromCtx(ctx).Error("claim pdf render job failed", "err", err, "render_job_id", payload.RenderJobID)
		return "", err
	}

	if err := h.runner.Run(ctx, task); err != nil {
		finished := time.Now().UTC()
		terminal, saveErr := h.tasks.RecordFailure(context.WithoutCancel(ctx), task.ID, "render_failed", truncateError(err), finished)
		if saveErr != nil {
			logger.FromCtx(ctx).Error("record pdf render failure failed", "err", saveErr, "render_job_id", task.ID, "report_id", task.ReportID)
			return "", saveErr
		}
		logger.FromCtx(ctx).Warn("pdf render attempt failed", "render_job_id", task.ID, "report_id", task.ReportID, "attempt", task.AttemptCount, "max_attempts", task.MaxAttempts, "terminal", terminal, "err", err)
		if terminal && h.reports != nil {
			if failErr := h.reports.Fail(context.WithoutCancel(ctx), task.ReportID, "pdf_render_failed", truncateError(err), finished); failErr != nil && !errors.Is(failErr, repository.ErrFullReportTerminal) {
				logger.FromCtx(ctx).Error("mark terminal pdf report failed", "err", failErr, "report_id", task.ReportID)
			} else if failErr == nil && h.outcomes != nil {
				h.outcomes.Failed(context.WithoutCancel(ctx), task.ReportID, "pdf_render_failed")
			}
		}
		return "", fmt.Errorf("pdf render attempt: %w", err)
	}
	if err := h.tasks.Succeed(context.WithoutCancel(ctx), task.ID, time.Now().UTC()); err != nil {
		logger.FromCtx(ctx).Error("complete pdf render job failed", "err", err, "render_job_id", task.ID, "report_id", task.ReportID)
		return "", err
	}
	return task.RenderVersion, nil
}

func StartFullReportPDFRecovery(ctx context.Context, tasks *repository.FullReportRenderJobRepo, reports *repository.FullReportRepo, queue job.Queue, staleAfter, interval time.Duration, outcomes ...FullReportOutcomeNotifier) {
	if staleAfter <= 0 {
		staleAfter = 10 * time.Minute
	}
	if interval <= 0 {
		interval = time.Minute
	}
	var outcomeNotifier FullReportOutcomeNotifier
	if len(outcomes) > 0 {
		outcomeNotifier = outcomes[0]
	}
	run := func() {
		queued, _, err := RecoverFullReportPDFJobs(ctx, tasks, queue, staleAfter)
		if err == nil && queued > 0 {
			logger.FromCtx(ctx).Info("pdf render tasks recovered", "queued", queued)
		}
		failed, listErr := tasks.ListFailedUnfinishedReports(ctx)
		if listErr != nil {
			logger.FromCtx(ctx).Error("list terminal pdf tasks failed", "err", listErr)
			return
		}
		for _, task := range failed {
			if err := reports.Fail(ctx, task.ReportID, "pdf_render_failed", task.ErrorSummary, time.Now().UTC()); err != nil && !errors.Is(err, repository.ErrFullReportTerminal) {
				logger.FromCtx(ctx).Error("mark recovered pdf report failed", "err", err, "report_id", task.ReportID)
			} else if err == nil && outcomeNotifier != nil {
				outcomeNotifier.Failed(ctx, task.ReportID, "pdf_render_failed")
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

// RecoverFullReportPDFJobs resets interrupted tasks and makes sure every
// durable queued task has a generic queue delivery after a service restart.
func RecoverFullReportPDFJobs(ctx context.Context, tasks *repository.FullReportRenderJobRepo, queue job.Queue, staleAfter time.Duration) (int, int, error) {
	now := time.Now().UTC()
	_, failed, err := tasks.RecoverStale(ctx, now.Add(-staleAfter), now)
	if err != nil {
		logger.FromCtx(ctx).Error("recover stale pdf tasks failed", "err", err)
		return 0, 0, err
	}
	queued, err := tasks.ListQueued(ctx)
	if err != nil {
		logger.FromCtx(ctx).Error("list queued pdf tasks failed", "err", err)
		return 0, failed, err
	}
	for _, task := range queued {
		payload, marshalErr := json.Marshal(fullReportPDFJobPayload{RenderJobID: task.ID})
		if marshalErr != nil {
			return 0, failed, marshalErr
		}
		if err := queue.Enqueue(ctx, &job.Job{Type: FullReportPDFJobType, Payload: payload, MaxAttempts: int(task.MaxAttempts)}); err != nil {
			logger.FromCtx(ctx).Error("enqueue recovered pdf task failed", "err", err, "render_job_id", task.ID, "report_id", task.ReportID)
			return 0, failed, err
		}
	}
	return len(queued), failed, nil
}
