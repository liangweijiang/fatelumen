package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

// ---------- DTOs ----------

// AdminReportItem 报告列表项（不含正文 Content/PDFURL）。
type AdminReportItem struct {
	ID        uint64    `json:"id"`
	UserID    uint64    `json:"user_id"`
	UserEmail string    `json:"user_email"`
	ProfileID uint64    `json:"profile_id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Paid      bool      `json:"paid"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminReportDetail 报告详情（含摘要/PDF 但不含完整章节正文）。
type AdminReportDetail struct {
	ID          uint64    `json:"id"`
	UserID      uint64    `json:"user_id"`
	ProfileID   uint64    `json:"profile_id"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Paid        bool      `json:"paid"`
	CreatedAt   time.Time `json:"created_at"`
	OrderID     *uint64   `json:"order_id"`
	SummaryLine string    `json:"summary_line"`
	Summary     string    `json:"summary"`
	PDFURL      string    `json:"pdf_url"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AdminReportsPage 分页结果。
type AdminReportsPage struct {
	Items    []AdminReportItem `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// ---------- private interfaces ----------

type adminReportStore interface {
	AdminListReports(filter repository.ReportFilter, limit, offset int) ([]model.Report, int64, error)
	AdminGetReportByID(id uint64) (*model.Report, error)
	MarkPaid(reportID, orderID uint64) error
}

type adminReportUserStore interface {
	EmailsByIDs(ids []uint64) (map[uint64]string, error)
}

// ---------- AdminReportService ----------

type AdminReportService struct {
	reportRepo adminReportStore
	userRepo   adminReportUserStore
}

func NewAdminReportService(reportRepo *repository.ReportRepo, userRepo *repository.UserRepo) *AdminReportService {
	return &AdminReportService{reportRepo: reportRepo, userRepo: userRepo}
}

// ListReports admin 分页报告列表。
func (s *AdminReportService) ListReports(ctx context.Context, status string, paid *bool, userID uint64, page, pageSize int) (*AdminReportsPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	reports, total, err := s.reportRepo.AdminListReports(repository.ReportFilter{
		Status: status,
		Paid:   paid,
		UserID: userID,
	}, pageSize, offset)
	if err != nil {
		logger.FromCtx(ctx).Error("admin list reports failed", "err", err)
		return nil, err
	}

	ids := make([]uint64, 0, len(reports))
	for _, r := range reports {
		ids = append(ids, r.UserID)
	}
	emails, err := s.userRepo.EmailsByIDs(ids)
	if err != nil {
		logger.FromCtx(ctx).Error("admin list reports: load emails failed", "err", err)
		emails = map[uint64]string{}
	}

	items := make([]AdminReportItem, len(reports))
	for i, r := range reports {
		items[i] = AdminReportItem{
			ID:        r.ID,
			UserID:    r.UserID,
			UserEmail: emails[r.UserID],
			ProfileID: r.ProfileID,
			Type:      r.PayMethod,
			Status:    r.Status,
			Paid:      r.Paid,
			CreatedAt: r.CreatedAt,
		}
	}
	return &AdminReportsPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetReportDetail admin 报告详情。
func (s *AdminReportService) GetReportDetail(ctx context.Context, reportID uint64) (*AdminReportDetail, error) {
	report, err := s.reportRepo.AdminGetReportByID(reportID)
	if err != nil {
		logger.FromCtx(ctx).Error("admin get report detail failed",
			"err", err,
			"report_id", reportID,
		)
		return nil, err
	}

	return &AdminReportDetail{
		ID:          report.ID,
		UserID:      report.UserID,
		ProfileID:   report.ProfileID,
		Type:        report.PayMethod,
		Status:      report.Status,
		Paid:        report.Paid,
		CreatedAt:   report.CreatedAt,
		OrderID:     report.OrderID,
		SummaryLine: report.Content.SummaryLine,
		Summary:     report.Content.Summary,
		PDFURL:      report.PDFURL,
		UpdatedAt:   report.UpdatedAt,
	}, nil
}

// UnlockReport admin 人工解锁报告（标记 paid，不关联订单）。
func (s *AdminReportService) UnlockReport(ctx context.Context, operatorID, reportID uint64, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return errors.New("reason required")
	}

	report, err := s.reportRepo.AdminGetReportByID(reportID)
	if err != nil {
		return err
	}

	if report.Paid {
		logger.FromCtx(ctx).Info("admin unlock report skipped (already unlocked)",
			"operator_id", operatorID,
			"report_id", reportID,
		)
		return nil
	}

	if err := s.reportRepo.MarkPaid(reportID, 0); err != nil {
		logger.FromCtx(ctx).Error("admin manual unlock failed",
			"err", err,
			"operator_id", operatorID,
			"target_report_id", reportID,
			"reason", reason,
		)
		return err
	}

	logger.FromCtx(ctx).Info("admin manual unlock report",
		"operator_id", operatorID,
		"target_report_id", reportID,
		"reason", reason,
	)
	return nil
}
