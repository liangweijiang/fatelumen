package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/renderer"
	"fatelumen/backend/internal/repository"
	"fatelumen/backend/internal/storage"
)

// 私有接口：ReportService 内部仅依赖接口，便于单测 fake
type reportStore interface {
	Create(*model.Report) error
	GetByID(uint64, uint64) (*model.Report, error)
	ListByUser(uint64, int, int) ([]model.Report, error)
	UpdateStatus(uint64, string, string) error
	UpdateResult(uint64, model.ReportContent, string) error
	UnlockReportWithCredits(userID, reportID uint64, cost int) error
}

type reportQueue interface {
	Enqueue(ctx context.Context, job *job.Job) error
}

type chartFinder interface {
	FindByID(uint64) (*model.Chart, error)
}

// reportPayload job 载荷。
type reportPayload struct {
	ReportID  uint64 `json:"report_id"`
	UserID    uint64 `json:"user_id"`
	ProfileID uint64 `json:"profile_id"`
	Locale    string `json:"locale"`
}

// ReportService 深度报告业务编排（入队侧 + 按需渲染）。
type ReportService struct {
	reportRepo  reportStore
	chartRepo   chartFinder
	renderer    renderer.Renderer
	fileStorage storage.Storage
	queue       reportQueue
	unlockCost  int
}

func NewReportService(
	reportRepo *repository.ReportRepo,
	chartRepo *repository.ChartRepo,
	imgRenderer renderer.Renderer,
	fileStorage storage.Storage,
	queue job.Queue,
	unlockCost int,
) *ReportService {
	return &ReportService{
		reportRepo:  reportRepo,
		chartRepo:   chartRepo,
		renderer:    imgRenderer,
		fileStorage: fileStorage,
		queue:       queue,
		unlockCost:  unlockCost,
	}
}

// CreateReport 创建报告（pending）并入队异步生成。
// 返回 Report（含 id 和 pending 状态），不阻塞等生成完成。
func (s *ReportService) CreateReport(ctx context.Context, userID, profileID uint64, locale string) (*model.Report, error) {
	if locale == "" {
		locale = "en"
	}

	report := &model.Report{
		UserID:    userID,
		ProfileID: profileID,
		ChartID:   0, // handler 排盘后回写
		Locale:    locale,
		Status:    model.ReportStatusPending,
		PayMethod: "credit",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.reportRepo.Create(report); err != nil {
		logger.FromCtx(ctx).Error("report create failed", "err", err, "user_id", userID, "profile_id", profileID)
		return nil, err
	}

	payloadBytes, err := json.Marshal(reportPayload{
		ReportID:  report.ID,
		UserID:    userID,
		ProfileID: profileID,
		Locale:    locale,
	})
	if err != nil {
		logger.FromCtx(ctx).Error("report payload marshal failed", "err", err, "report_id", report.ID)
		return nil, err
	}

	reportJob := &job.Job{
		Type:    "report",
		Payload: payloadBytes,
	}

	if err := s.queue.Enqueue(ctx, reportJob); err != nil {
		logger.FromCtx(ctx).Error("report job enqueue failed", "err", err, "report_id", report.ID)
		return nil, err
	}

	logger.FromCtx(ctx).Info("report created and enqueued",
		"report_id", report.ID,
		"user_id", userID,
		"profile_id", profileID,
		"job_id", reportJob.ID,
	)
	return report, nil
}

// UnlockWithCredits 用积分解锁报告。校验报告归属与状态后,委托 repo 在单事务内扣积分+写流水+标记解锁。
// 报告不存在或非本人返回 gorm.ErrRecordNotFound;报告尚未生成完成(status != done)返回错误;余额不足返回 repository.ErrInsufficientCredits。
func (s *ReportService) UnlockWithCredits(ctx context.Context, userID, reportID uint64) error {
	report, err := s.reportRepo.GetByID(reportID, userID)
	if err != nil {
		logger.FromCtx(ctx).Warn("unlock with credits: report not found", "err", err, "user_id", userID, "report_id", reportID)
		return err
	}
	if report.Paid {
		// 已解锁,幂等返回
		return nil
	}
	if report.Status != model.ReportStatusDone {
		logger.FromCtx(ctx).Warn("unlock with credits: report not ready", "report_id", reportID, "status", report.Status)
		return fmt.Errorf("report not ready for unlock")
	}
	if err := s.reportRepo.UnlockReportWithCredits(userID, reportID, s.unlockCost); err != nil {
		logger.FromCtx(ctx).Error("unlock with credits failed", "err", err, "user_id", userID, "report_id", reportID, "cost", s.unlockCost)
		return err
	}
	logger.FromCtx(ctx).Info("report unlocked with credits", "user_id", userID, "report_id", reportID, "cost", s.unlockCost)
	return nil
}

// GetReport 获取报告详情（带归属校验）。
func (s *ReportService) GetReport(ctx context.Context, userID, reportID uint64) (*model.Report, error) {
	return s.reportRepo.GetByID(reportID, userID)
}

// ListReports 列出用户报告。
func (s *ReportService) ListReports(ctx context.Context, userID uint64, limit, offset int) ([]model.Report, error) {
	return s.reportRepo.ListByUser(userID, limit, offset)
}

// ExportReportPDF 按需懒生成 PDF：若已有缓存 URL 直接返回，否则渲染→上传→回写→返回。
func (s *ReportService) ExportReportPDF(ctx context.Context, userID, reportID uint64) (string, error) {
	report, err := s.GetReport(ctx, userID, reportID)
	if err != nil {
		return "", fmt.Errorf("report not found: %w", err)
	}
	if report.Status != model.ReportStatusDone {
		return "", fmt.Errorf("report not ready, status: %s", report.Status)
	}
	if report.PDFURL != "" {
		return report.PDFURL, nil
	}

	chart, err := s.chartRepo.FindByID(report.ChartID)
	if err != nil {
		logger.FromCtx(ctx).Error("chart not found for export pdf", "err", err, "chart_id", report.ChartID, "report_id", reportID)
		return "", fmt.Errorf("chart not found: %w", err)
	}

	pdfData := renderer.BuildReportPDFData(&chart.ChartData, report.Content, time.Now().UTC().Format("2006-01-02"))
	pdf, err := renderer.RenderReportPDF(ctx, s.renderer, pdfData)
	if err != nil {
		return "", fmt.Errorf("render pdf: %w", err)
	}

	key := storage.ReportKey(userID, reportID)
	pdfURL, err := s.fileStorage.Put(ctx, key, pdf, "application/pdf")
	if err != nil {
		logger.FromCtx(ctx).Error("storage upload failed for export pdf", "err", err, "key", key, "report_id", reportID)
		return "", fmt.Errorf("storage upload: %w", err)
	}

	if err := s.reportRepo.UpdateResult(reportID, report.Content, pdfURL); err != nil {
		logger.FromCtx(ctx).Error("report pdf_url update failed", "err", err, "report_id", reportID)
		return "", fmt.Errorf("update pdf_url: %w", err)
	}

	logger.FromCtx(ctx).Info("report pdf exported lazily",
		"report_id", reportID,
		"user_id", userID,
	)
	return pdfURL, nil
}

// RenderReportHTML 在线报告 HTML：从库读内容 + 排盘数据，渲染完整 HTML。
func (s *ReportService) RenderReportHTML(ctx context.Context, userID, reportID uint64) (string, error) {
	report, err := s.GetReport(ctx, userID, reportID)
	if err != nil {
		return "", fmt.Errorf("report not found: %w", err)
	}
	if report.Status != model.ReportStatusDone {
		return "", fmt.Errorf("report not ready, status: %s", report.Status)
	}

	chart, err := s.chartRepo.FindByID(report.ChartID)
	if err != nil {
		logger.FromCtx(ctx).Error("chart not found for view html", "err", err, "chart_id", report.ChartID, "report_id", reportID)
		return "", fmt.Errorf("chart not found: %w", err)
	}

	pdfData := renderer.BuildReportPDFData(&chart.ChartData, report.Content, time.Now().UTC().Format("2006-01-02"))
	html, err := renderer.RenderReportHTML(ctx, pdfData)
	if err != nil {
		return "", fmt.Errorf("render html: %w", err)
	}

	logger.FromCtx(ctx).Info("report html rendered",
		"report_id", reportID,
		"user_id", userID,
	)
	return html, nil
}
