package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"

	"gorm.io/gorm"
)

const emptyFullReportFactsHash = "0000000000000000000000000000000000000000000000000000000000000000"

// FullReportService is the user-facing facade for the new immutable report
// domain. model.Report is returned only as the existing HTTP response DTO; no
// read or write is performed against the legacy reports table.
type FullReportService struct {
	reports       *repository.FullReportRepo
	profiles      *repository.ProfileRepo
	queue         job.Queue
	reportCost    int
	concurrency   int
	retentionDays int
}

func NewFullReportService(reports *repository.FullReportRepo, profiles *repository.ProfileRepo, queue job.Queue, reportCost, concurrency, retentionDays int) *FullReportService {
	if reportCost <= 0 {
		reportCost = 10
	}
	if concurrency < 1 || concurrency > 10 {
		concurrency = 3
	}
	if retentionDays < 1 {
		retentionDays = 30
	}
	return &FullReportService{reports: reports, profiles: profiles, queue: queue, reportCost: reportCost, concurrency: concurrency, retentionDays: retentionDays}
}

func (s *FullReportService) CreateReport(ctx context.Context, userID, profileID uint64, locale string) (*model.Report, error) {
	if _, err := s.profiles.FindByIDAndUserID(profileID, userID); err != nil {
		logger.FromCtx(ctx).Warn("full report profile ownership check failed", "err", err, "user_id", userID, "profile_id", profileID)
		return nil, err
	}
	locale = normalizeReportLocale(locale)
	now := time.Now().UTC()
	publicID, err := newFullReportPublicID()
	if err != nil {
		return nil, err
	}
	report := &model.FullReport{
		PublicID: publicID, UserID: userID, ProfileID: &profileID, Locale: locale,
		PayMethod: "credit", Status: model.FullReportStatusPending, CurrentStage: model.FullReportStatusPending,
		ChapterTotal: 10, ProviderChainKey: "default", ChapterConcurrency: uint8(s.concurrency),
		FactsHash: emptyFullReportFactsHash, RetentionPolicy: fmt.Sprintf("days:%d", s.retentionDays),
		ExpiresAt: now.AddDate(0, 0, s.retentionDays), CreatedAt: now, UpdatedAt: now,
	}
	if err := s.reports.CreateWithCreditCharge(ctx, report, s.reportCost); err != nil {
		if errors.Is(err, repository.ErrInsufficientCredits) {
			logger.FromCtx(ctx).Warn("full report create rejected: insufficient credits", "user_id", userID, "profile_id", profileID)
		} else {
			logger.FromCtx(ctx).Error("full report create failed", "err", err, "user_id", userID, "profile_id", profileID)
		}
		return nil, err
	}
	payload, err := json.Marshal(fullReportPayload{ReportID: report.ID, UserID: userID, ProfileID: profileID, Locale: locale})
	if err != nil {
		return nil, err
	}
	reportJob := &job.Job{Type: FullReportJobType, Lane: job.LaneReportGeneration, Payload: payload, MaxAttempts: 1}
	if err := s.queue.Enqueue(ctx, reportJob); err != nil {
		logger.FromCtx(ctx).Error("full report enqueue failed", "err", err, "report_id", report.ID)
		if refundErr := s.reports.CancelPendingCreditCharge(ctx, report.ID, "enqueue_failed", "report job could not be queued", time.Now().UTC()); refundErr != nil {
			logger.FromCtx(ctx).Error("full report enqueue compensation failed", "err", refundErr, "report_id", report.ID, "user_id", userID)
		}
		return nil, err
	}
	logger.FromCtx(ctx).Info("full report created and enqueued", "report_id", report.ID, "user_id", userID, "profile_id", profileID, "job_id", reportJob.ID)
	return fullReportDTO(report, nil), nil
}

func (s *FullReportService) GetReport(ctx context.Context, userID, reportID uint64) (*model.Report, error) {
	report, err := s.reports.GetByID(ctx, reportID, userID)
	if err != nil {
		return nil, err
	}
	var result *model.FullReportResult
	if report.Status == model.FullReportStatusCompleted {
		result, err = s.reports.GetResult(ctx, reportID)
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	return fullReportDTO(report, result), nil
}

func (s *FullReportService) ListReports(ctx context.Context, userID uint64, limit int, cursor *repository.FullReportCursor) ([]model.Report, bool, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.reports.ListByUser(ctx, userID, limit+1, cursor)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	out := make([]model.Report, len(rows))
	for i := range rows {
		out[i] = *fullReportDTO(&rows[i], nil)
	}
	return out, hasMore, nil
}

func fullReportDTO(report *model.FullReport, result *model.FullReportResult) *model.Report {
	status := report.Status
	switch report.Status {
	case model.FullReportStatusCompleted:
		status = model.ReportStatusDone
	case model.FullReportStatusFailed:
		status = model.ReportStatusFailed
	case model.FullReportStatusPending:
		status = model.ReportStatusPending
	default:
		status = model.ReportStatusProcessing
	}
	dto := &model.Report{ID: report.ID, UserID: report.UserID, Locale: report.Locale, Status: status, PayMethod: report.PayMethod, Paid: report.Paid, ErrorMsg: report.ErrorSummary, CreatedAt: report.CreatedAt, UpdatedAt: report.UpdatedAt}
	if report.ProfileID != nil {
		dto.ProfileID = *report.ProfileID
	}
	if report.ChartID != nil {
		dto.ChartID = *report.ChartID
	}
	if report.OrderID != nil {
		dto.OrderID = report.OrderID
	}
	if result != nil {
		dto.Content = result.Content
		dto.PDFURL = result.PDFURL
	}
	return dto
}

func normalizeReportLocale(locale string) string {
	switch locale {
	case "zh", "en", "ja", "ko":
		return locale
	default:
		return "en"
	}
}

func newFullReportPublicID() (string, error) {
	b := make([]byte, 13)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate full report public id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
