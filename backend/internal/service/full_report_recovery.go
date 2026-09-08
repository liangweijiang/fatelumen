package service

import (
	"context"
	"encoding/json"
	"fmt"

	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

// RecoverInterruptedFullReports restores deliveries lost with an in-memory
// queue restart. It runs synchronously before workers start, so recovered jobs
// cannot race with an older worker in the same process.
func RecoverInterruptedFullReports(ctx context.Context, reports *repository.FullReportRepo, queue job.Queue) (int, error) {
	if reports == nil || queue == nil {
		return 0, fmt.Errorf("full report recovery is not configured")
	}
	const pageSize = 100
	afterID := uint64(0)
	recovered := 0
	for {
		rows, err := reports.ListInterruptedExecutions(ctx, afterID, pageSize)
		if err != nil {
			logger.FromCtx(ctx).Error("list interrupted full reports failed", "err", err, "after_id", afterID)
			return recovered, err
		}
		for _, report := range rows {
			if report.ProfileID == nil {
				return recovered, fmt.Errorf("interrupted report %d has no profile", report.ID)
			}
			payload, err := json.Marshal(fullReportPayload{ReportID: report.ID, UserID: report.UserID, ProfileID: *report.ProfileID, Locale: report.Locale})
			if err != nil {
				return recovered, err
			}
			if err := queue.Enqueue(ctx, &job.Job{Type: FullReportJobType, Lane: job.LaneReportGeneration, Payload: payload, Attempts: 1, MaxAttempts: 1}); err != nil {
				logger.FromCtx(ctx).Error("enqueue interrupted full report failed", "err", err, "report_id", report.ID)
				return recovered, err
			}
			recovered++
			afterID = report.ID
		}
		if len(rows) < pageSize {
			break
		}
	}
	return recovered, nil
}
