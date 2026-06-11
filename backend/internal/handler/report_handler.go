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
	response.OK(c, report)
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
