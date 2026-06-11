package service

import (
	"context"
	"encoding/json"
	"time"

	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

// 私有接口：ReportService 内部仅依赖接口，便于单测 fake
type reportStore interface {
	Create(*model.Report) error
	GetByID(uint64, uint64) (*model.Report, error)
	ListByUser(uint64, int, int) ([]model.Report, error)
	UpdateStatus(uint64, string, string) error
	UpdateResult(uint64, model.ReportContent, string) error
}

type reportQueue interface {
	Enqueue(ctx context.Context, job *job.Job) error
}

// reportPayload job 载荷。
type reportPayload struct {
	ReportID  uint64 `json:"report_id"`
	UserID    uint64 `json:"user_id"`
	ProfileID uint64 `json:"profile_id"`
	Locale    string `json:"locale"`
}

// ReportService 深度报告业务编排（入队侧）。
type ReportService struct {
	reportRepo reportStore
	queue      reportQueue
}

func NewReportService(reportRepo *repository.ReportRepo, queue job.Queue) *ReportService {
	return &ReportService{
		reportRepo: reportRepo,
		queue:      queue,
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
		Status:    "pending",
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

// GetReport 获取报告详情（带归属校验）。
func (s *ReportService) GetReport(ctx context.Context, userID, reportID uint64) (*model.Report, error) {
	return s.reportRepo.GetByID(reportID, userID)
}

// ListReports 列出用户报告。
func (s *ReportService) ListReports(ctx context.Context, userID uint64, limit, offset int) ([]model.Report, error) {
	return s.reportRepo.ListByUser(userID, limit, offset)
}
