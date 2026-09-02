package handler

import (
	"encoding/json"
	"errors"
	"strconv"

	"fatelumen/backend/internal/llm/prompts"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminReportTraceHandler struct {
	reports *repository.FullReportRepo
	audit   *repository.AuditRepo
}

func NewAdminReportTraceHandler(reports *repository.FullReportRepo, audit *repository.AuditRepo) *AdminReportTraceHandler {
	return &AdminReportTraceHandler{reports: reports, audit: audit}
}

func (h *AdminReportTraceHandler) Facts(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	trace, err := h.reports.AdminGetExecutionTrace(c.Request.Context(), reportID)
	if err != nil {
		h.readError(c, err, "report facts unavailable", reportID)
		return
	}
	var input model.ReportInputSnapshot
	var tc model.TimeCalculationSnapshot
	var chart model.ChartSnapshot
	var facts model.InterpretationFacts
	if err := decodeExecutionTrace(trace, &input, &tc, &chart, &facts); err != nil {
		logger.FromCtx(c.Request.Context()).Error("decode report fact snapshot failed", "err", err, "report_id", reportID)
		response.Error(c, "report facts unavailable")
		return
	}
	h.auditRead(c, "view_facts", reportID, "")
	response.OK(c, gin.H{
		"report_id":   reportID,
		"model_stats": trace.ModelStats,
		"snapshot": gin.H{
			"metadata": trace.Snapshot, "input_snapshot": input, "time_calculation_snapshot": tc,
			"chart_snapshot": chart, "facts": facts, "preflight_result": trace.Payload.PreflightResult,
			"chapter_plan_snapshot": trace.Payload.ChapterPlanSnapshot, "runtime_config_snapshot": trace.Payload.RuntimeConfigSnapshot,
		},
	})
}

func (h *AdminReportTraceHandler) LLMCalls(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	page, size := 1, 20
	if v, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil && v > 0 && v <= 100 {
		size = v
	}
	rows, total, err := h.reports.AdminListAttempts(c.Request.Context(), reportID, size, (page-1)*size)
	if err != nil {
		h.readError(c, err, "report call traces unavailable", reportID)
		return
	}
	stats, err := h.reports.AdminAttemptStats(c.Request.Context(), reportID)
	if err != nil {
		h.readError(c, err, "report call statistics unavailable", reportID)
		return
	}
	h.auditRead(c, "view_llm_calls", reportID, "")
	response.OK(c, gin.H{"report_id": reportID, "items": rows, "model_stats": stats, "total": total, "page": page, "page_size": size})
}

func (h *AdminReportTraceHandler) LLMCall(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	callID, ok := reportTraceID(c, "callId")
	if !ok {
		return
	}
	row, err := h.reports.AdminGetAttemptTrace(c.Request.Context(), reportID, callID)
	if err != nil {
		h.readError(c, err, "report call trace unavailable", reportID)
		return
	}
	h.auditRead(c, "view_llm_call", reportID, strconv.FormatUint(callID, 10))
	response.OK(c, row)
}

func (h *AdminReportTraceHandler) Validations(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	rows, err := h.reports.AdminListValidationRuns(c.Request.Context(), reportID)
	if err != nil {
		h.readError(c, err, "report validations unavailable", reportID)
		return
	}
	h.auditRead(c, "view_report_validations", reportID, "")
	response.OK(c, gin.H{"report_id": reportID, "items": rows})
}

func (h *AdminReportTraceHandler) Validation(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	validationID, ok := reportTraceID(c, "validationId")
	if !ok {
		return
	}
	row, err := h.reports.AdminGetValidationTrace(c.Request.Context(), reportID, validationID)
	if err != nil {
		h.readError(c, err, "report validation unavailable", reportID)
		return
	}
	h.auditRead(c, "view_report_validation", reportID, strconv.FormatUint(validationID, 10))
	response.OK(c, row)
}

func (h *AdminReportTraceHandler) Chapters(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	rows, err := h.reports.AdminListChapters(c.Request.Context(), reportID)
	if err != nil {
		h.readError(c, err, "report chapters unavailable", reportID)
		return
	}
	h.auditRead(c, "view_report_chapters", reportID, "")
	response.OK(c, gin.H{"report_id": reportID, "items": rows})
}

func (h *AdminReportTraceHandler) Chapter(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	chapterID, ok := reportTraceID(c, "chapterId")
	if !ok {
		return
	}
	row, err := h.reports.AdminGetChapterTrace(c.Request.Context(), reportID, chapterID)
	if err != nil {
		h.readError(c, err, "report chapter unavailable", reportID)
		return
	}
	h.auditRead(c, "view_report_chapter", reportID, strconv.FormatUint(chapterID, 10))
	response.OK(c, row)
}

type promptPreviewRequest struct {
	ChapterKey string `json:"chapter_key" binding:"required"`
	Locale     string `json:"locale" binding:"required"`
}

// PromptPreview builds the exact chapter prompt and filtered fact payload
// without invoking an external provider or mutating the immutable snapshot.
func (h *AdminReportTraceHandler) PromptPreview(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	var req promptPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeBadRequest, "chapter_key and locale are required")
		return
	}
	trace, err := h.reports.AdminGetExecutionTrace(c.Request.Context(), reportID)
	if err != nil {
		h.readError(c, err, "report facts unavailable", reportID)
		return
	}
	var input model.ReportInputSnapshot
	var tc model.TimeCalculationSnapshot
	var chart model.ChartSnapshot
	var facts model.InterpretationFacts
	if err := decodeExecutionTrace(trace, &input, &tc, &chart, &facts); err != nil {
		logger.FromCtx(c.Request.Context()).Error("decode prompt preview facts failed", "err", err, "report_id", reportID)
		response.Error(c, "prompt preview unavailable")
		return
	}
	preview, err := prompts.BuildChapterPromptPreview(req.Locale, req.ChapterKey, facts)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Warn("prompt preview rejected", "err", err, "report_id", reportID, "chapter_key", req.ChapterKey, "locale", req.Locale)
		response.Fail(c, response.CodeBadRequest, err.Error())
		return
	}
	h.auditRead(c, "preview_prompt", reportID, "")
	response.OK(c, preview)
}

func (h *AdminReportTraceHandler) PromptRegistry(c *gin.Context) {
	response.OK(c, gin.H{"version": prompts.ChapterRegistryVersion, "chapters": prompts.ChapterDefinitions(), "locales": prompts.SupportedLocales()})
}

func reportTraceID(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, response.CodeBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func decodeExecutionTrace(trace *repository.FullReportExecutionTrace, input *model.ReportInputSnapshot, tc *model.TimeCalculationSnapshot, chart *model.ChartSnapshot, facts *model.InterpretationFacts) error {
	for _, item := range []struct {
		raw model.JSONRaw
		dst any
	}{{trace.Payload.InputSnapshot, input}, {trace.Payload.TimeCalculationSnapshot, tc}, {trace.Payload.ChartSnapshot, chart}, {trace.Payload.FactsSnapshot, facts}} {
		if err := json.Unmarshal(item.raw, item.dst); err != nil {
			return err
		}
	}
	return nil
}

func (h *AdminReportTraceHandler) readError(c *gin.Context, err error, message string, reportID uint64) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, response.CodeNotFound, "report trace not found")
		return
	}
	logger.FromCtx(c.Request.Context()).Error(message, "err", err, "report_id", reportID)
	response.Error(c, message)
}

func (h *AdminReportTraceHandler) auditRead(c *gin.Context, action string, reportID uint64, callID string) {
	detail, _ := json.Marshal(gin.H{"call_id": callID})
	h.audit.Write(c.Request.Context(), model.AdminAuditLog{AdminID: middleware.GetAdminID(c), AdminName: c.GetString("admin_name"), Action: action, Resource: "report_trace", ResourceID: strconv.FormatUint(reportID, 10), Detail: model.JSONRaw(detail), IP: c.ClientIP()})
}
