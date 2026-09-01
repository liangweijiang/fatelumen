package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/hash"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/renderer"
	"fatelumen/backend/internal/repository"
	"fatelumen/backend/internal/storage"

	"gorm.io/gorm"
)

const emptyFullReportFactsHash = "0000000000000000000000000000000000000000000000000000000000000000"

// FullReportService is the user-facing facade for the new immutable report
// domain. model.Report is returned only as the existing HTTP response DTO; no
// read or write is performed against the legacy reports table.
type FullReportService struct {
	reports       *repository.FullReportRepo
	profiles      *repository.ProfileRepo
	charts        *repository.ChartRepo
	renderer      renderer.Renderer
	storage       storage.Storage
	queue         job.Queue
	unlockCost    int
	concurrency   int
	retentionDays int
}

func NewFullReportService(reports *repository.FullReportRepo, profiles *repository.ProfileRepo, charts *repository.ChartRepo, imageRenderer renderer.Renderer, fileStorage storage.Storage, queue job.Queue, unlockCost, concurrency, retentionDays int) *FullReportService {
	if concurrency < 1 || concurrency > 10 {
		concurrency = 3
	}
	if retentionDays < 1 {
		retentionDays = 30
	}
	return &FullReportService{reports: reports, profiles: profiles, charts: charts, renderer: imageRenderer, storage: fileStorage, queue: queue, unlockCost: unlockCost, concurrency: concurrency, retentionDays: retentionDays}
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
	if err := s.reports.Create(ctx, report); err != nil {
		logger.FromCtx(ctx).Error("full report create failed", "err", err, "user_id", userID, "profile_id", profileID)
		return nil, err
	}
	payload, err := json.Marshal(fullReportPayload{ReportID: report.ID, UserID: userID, ProfileID: profileID, Locale: locale})
	if err != nil {
		return nil, err
	}
	reportJob := &job.Job{Type: "full_report_v2", Payload: payload, MaxAttempts: 1}
	if err := s.queue.Enqueue(ctx, reportJob); err != nil {
		logger.FromCtx(ctx).Error("full report enqueue failed", "err", err, "report_id", report.ID)
		_ = s.reports.Fail(ctx, report.ID, "enqueue_failed", "report job could not be queued", time.Now().UTC())
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

func (s *FullReportService) ListReports(ctx context.Context, userID uint64, limit, offset int) ([]model.Report, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.reports.ListByUserOffset(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]model.Report, len(rows))
	for i := range rows {
		out[i] = *fullReportDTO(&rows[i], nil)
	}
	return out, nil
}

func (s *FullReportService) UnlockWithCredits(ctx context.Context, userID, reportID uint64) error {
	err := s.reports.UnlockWithCredits(ctx, userID, reportID, s.unlockCost)
	if err != nil {
		logger.FromCtx(ctx).Error("full report unlock failed", "err", err, "user_id", userID, "report_id", reportID)
	}
	return err
}

func (s *FullReportService) ExportReportPDF(ctx context.Context, userID, reportID uint64) (string, error) {
	report, result, chart, err := s.renderInputs(ctx, userID, reportID)
	if err != nil {
		return "", err
	}
	if result.PDFURL != "" {
		return result.PDFURL, nil
	}
	pdfData := renderer.BuildReportPDFData(&chart.ChartData, result.Content, report.CompletedAt.Format("2006-01-02"))
	pdfBytes, err := renderer.RenderReportPDF(ctx, s.renderer, pdfData)
	if err != nil {
		return "", fmt.Errorf("render pdf: %w", err)
	}
	key := storage.ReportKey(userID, reportID)
	url, err := s.storage.Put(ctx, key, pdfBytes, "application/pdf")
	if err != nil {
		logger.FromCtx(ctx).Error("full report pdf upload failed", "err", err, "report_id", reportID, "key", key)
		return "", err
	}
	pdfHash, err := hash.CanonicalJSONSHA256(pdfBytes)
	if err != nil {
		return "", err
	}
	if err := s.reports.UpdatePDF(ctx, reportID, key, url, pdfHash); err != nil {
		return "", err
	}
	return url, nil
}

func (s *FullReportService) RenderReportHTML(ctx context.Context, userID, reportID uint64) (string, error) {
	report, result, chart, err := s.renderInputs(ctx, userID, reportID)
	if err != nil {
		return "", err
	}
	pdfData := renderer.BuildReportPDFData(&chart.ChartData, result.Content, report.CompletedAt.Format("2006-01-02"))
	return renderer.RenderReportHTML(ctx, pdfData)
}

func (s *FullReportService) renderInputs(ctx context.Context, userID, reportID uint64) (*model.FullReport, *model.FullReportResult, *model.Chart, error) {
	report, err := s.reports.GetByID(ctx, reportID, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	if report.Status != model.FullReportStatusCompleted || report.ChartID == nil {
		return nil, nil, nil, repository.ErrFullReportNotReady
	}
	result, err := s.reports.GetResult(ctx, reportID)
	if err != nil {
		return nil, nil, nil, err
	}
	chart, err := s.charts.FindByID(*report.ChartID)
	if err != nil {
		return nil, nil, nil, err
	}
	return report, result, chart, nil
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
