package handler

import (
	"context"
	"errors"
	"strconv"

	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// reportSvc allows test fakes to replace the real ReportService.
type reportSvc interface {
	CreateReport(ctx context.Context, userID, profileID uint64, locale string) (*model.Report, error)
	GetReport(ctx context.Context, userID, reportID uint64) (*model.Report, error)
	ListReports(ctx context.Context, userID uint64, limit, offset int) ([]model.Report, error)
}

// ReportHandler 深度报告 HTTP 处理器。
type ReportHandler struct {
	svc reportSvc
}

func NewReportHandler(svc *service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

type createReportIn struct {
	ProfileID uint64 `json:"profile_id"`
	Locale    string `json:"locale"`
}

// reportDetailResponse Get /:id 返回的报告详情 DTO。
type reportDetailResponse struct {
	ID          uint64                `json:"id"`
	Status      string                `json:"status"`
	Locale      string                `json:"locale"`
	Paid        bool                  `json:"paid"`
	Locked      bool                  `json:"locked"`
	SummaryLine string                `json:"summary_line"`
	Summary     string                `json:"summary"`
	Content     *model.ReportContent  `json:"content,omitempty"`
	PDFURL      string                `json:"pdf_url,omitempty"`
	ProfileID   uint64                `json:"profile_id"`
	CreatedAt   interface{}           `json:"created_at"`
	UpdatedAt   interface{}           `json:"updated_at"`
}

// buildReportDetail 根据 unlocked 标志组装报告详情响应。
// 纯函数，便于单测。unlocked 时返回全文，否则按付费状态门控。
func buildReportDetail(r *model.Report, unlocked bool) reportDetailResponse {
	resp := reportDetailResponse{
		ID:          r.ID,
		Status:      r.Status,
		Locale:      r.Locale,
		Paid:        r.Paid,
		SummaryLine: r.Content.SummaryLine,
		Summary:     r.Content.Summary,
		ProfileID:   r.ProfileID,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	// 报告未完成时不触发门控，直接返回生成状态
	if r.Status != "done" {
		return resp
	}

	// unlocked（已付费 或 admin 豁免）→ 返回全文
	if unlocked {
		resp.Locked = false
		resp.Content = &r.Content
		resp.PDFURL = r.PDFURL
		return resp
	}

	// 已完成但未付费且非 admin → 锁定态：仅摘要钩子，无深度内容
	resp.Locked = true
	return resp
}

// Create POST /api/v1/reports
func (h *ReportHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}

	var in createReportIn
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid request body")
		return
	}
	if in.ProfileID == 0 {
		response.Fail(c, response.CodeBadRequest, "profile_id is required")
		return
	}

	report, err := h.svc.CreateReport(c.Request.Context(), userID, in.ProfileID, in.Locale)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, gin.H{
		"report_id": report.ID,
		"status":    report.Status,
	})
}

// Get GET /api/v1/reports/:id
func (h *ReportHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid report id")
		return
	}

	report, err := h.svc.GetReport(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, response.CodeNotFound, "report not found")
			return
		}
		response.Error(c, err.Error())
		return
	}
	unlocked := report.Paid || middleware.IsAdmin(c)
	response.OK(c, buildReportDetail(report, unlocked))
}

// List GET /api/v1/reports
func (h *ReportHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	reports, err := h.svc.ListReports(c.Request.Context(), userID, limit, offset)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, reports)
}
