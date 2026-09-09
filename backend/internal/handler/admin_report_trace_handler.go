package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"fatelumen/backend/internal/llm/prompts"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/repository"
	"fatelumen/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminReportTraceHandler struct {
	reports    *repository.FullReportRepo
	renderJobs *repository.FullReportRenderJobRepo
	audit      *repository.AuditRepo
}

func NewAdminReportTraceHandler(reports *repository.FullReportRepo, audit *repository.AuditRepo, renderJobs ...*repository.FullReportRenderJobRepo) *AdminReportTraceHandler {
	var jobs *repository.FullReportRenderJobRepo
	if len(renderJobs) > 0 {
		jobs = renderJobs[0]
	}
	return &AdminReportTraceHandler{reports: reports, renderJobs: jobs, audit: audit}
}

func (h *AdminReportTraceHandler) List(c *gin.Context) {
	pageSize := 20
	if v, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil && v > 0 && v <= 100 {
		pageSize = v
	}
	filter := repository.AdminFullReportListFilter{Status: strings.TrimSpace(c.Query("status")), Locale: strings.TrimSpace(c.Query("locale"))}
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		userID, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || userID == 0 {
			response.Fail(c, response.CodeBadRequest, "invalid user_id")
			return
		}
		filter.UserID = userID
	}
	for raw, target := range map[string]**time.Time{"created_from": &filter.CreatedFrom, "created_to": &filter.CreatedTo} {
		if value := strings.TrimSpace(c.Query(raw)); value != "" {
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				response.Fail(c, response.CodeBadRequest, "invalid "+raw)
				return
			}
			*target = &parsed
		}
	}
	if raw := strings.TrimSpace(c.Query("cursor")); raw != "" {
		cursor, err := decodeReportCursor(raw)
		if err != nil {
			response.Fail(c, response.CodeBadRequest, "invalid cursor")
			return
		}
		filter.Cursor = cursor
	}
	items, hasMore, err := h.reports.AdminListPage(c.Request.Context(), filter, pageSize)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("admin list full reports failed", "err", err)
		response.Error(c, "report list unavailable")
		return
	}
	nextCursor := ""
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = encodeReportCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, gin.H{"items": items, "page_size": pageSize, "has_more": hasMore, "next_cursor": nextCursor})
}

func (h *AdminReportTraceHandler) Overview(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	report, err := h.reports.AdminGetByID(c.Request.Context(), reportID)
	if err != nil {
		h.readError(c, err, "report overview unavailable", reportID)
		return
	}
	stats, err := h.reports.AdminAttemptStats(c.Request.Context(), reportID)
	if err != nil {
		h.readError(c, err, "report call statistics unavailable", reportID)
		return
	}
	settlement, err := h.reports.AdminCreditSettlement(c.Request.Context(), report)
	if err != nil {
		h.readError(c, err, "report credit settlement unavailable", reportID)
		return
	}
	var renderJob *model.FullReportRenderJob
	if h.renderJobs != nil {
		renderJob, err = h.renderJobs.GetByReportID(c.Request.Context(), reportID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			h.readError(c, err, "report render status unavailable", reportID)
			return
		}
	}
	h.auditRead(c, "view_report_overview", reportID, "")
	response.OK(c, gin.H{"report": report, "model_stats": stats, "render_job": renderJob, "credit_settlement": settlement})
}

func (h *AdminReportTraceHandler) Result(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	result, err := h.reports.AdminGetResult(c.Request.Context(), reportID)
	if err != nil {
		h.readError(c, err, "report result unavailable", reportID)
		return
	}
	h.auditRead(c, "view_report_result", reportID, "")
	response.OK(c, result)
}

// Integrity recalculates immutable hashes only when an administrator requests
// it, so normal report list and overview reads never load large snapshots.
func (h *AdminReportTraceHandler) Integrity(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	report, err := h.reports.AdminGetByID(c.Request.Context(), reportID)
	if err != nil {
		h.readError(c, err, "report unavailable for integrity check", reportID)
		return
	}
	trace, err := h.reports.AdminGetExecutionTrace(c.Request.Context(), reportID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		h.readError(c, err, "report execution unavailable for integrity check", reportID)
		return
	}
	result, err := h.reports.AdminGetResult(c.Request.Context(), reportID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		h.readError(c, err, "report result unavailable for integrity check", reportID)
		return
	}
	var traceInput *service.FullReportIntegrityTrace
	if trace != nil {
		traceInput = service.NewIntegrityTrace(trace.Snapshot, trace.Payload)
	}
	verification, err := service.VerifyFullReportIntegrity(report, traceInput, result)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("verify report integrity failed", "err", err, "report_id", reportID)
		response.Error(c, "report integrity check unavailable")
		return
	}
	h.auditRead(c, "verify_report_integrity", reportID, "")
	response.OK(c, verification)
}

func encodeReportCursor(createdAt time.Time, id uint64) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "|" + strconv.FormatUint(id, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeReportCursor(raw string) (*repository.FullReportCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(decoded), "|")
	if len(parts) != 2 {
		return nil, errors.New("invalid cursor")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || id == 0 {
		return nil, errors.New("invalid cursor id")
	}
	return &repository.FullReportCursor{CreatedAt: createdAt, ID: id}, nil
}

func (h *AdminReportTraceHandler) Facts(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	section := strings.TrimSpace(c.Query("section"))
	columns := map[string]string{"input": "input_snapshot", "time": "time_calculation_snapshot", "chart": "chart_snapshot", "interpretation": "facts_snapshot"}
	column, exists := columns[section]
	if !exists {
		response.Fail(c, response.CodeBadRequest, "invalid facts section")
		return
	}
	raw, err := h.reports.AdminGetExecutionSection(c.Request.Context(), reportID, column)
	if err != nil {
		h.readError(c, err, "report facts unavailable", reportID)
		return
	}
	h.auditRead(c, "view_facts", reportID, "")
	response.OK(c, gin.H{"report_id": reportID, "section": section, "value": raw})
}

func (h *AdminReportTraceHandler) Preflight(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	raw, err := h.reports.AdminGetExecutionSection(c.Request.Context(), reportID, "preflight_result")
	if err != nil {
		h.readError(c, err, "report preflight unavailable", reportID)
		return
	}
	h.auditRead(c, "view_report_preflight", reportID, "")
	response.OK(c, gin.H{"report_id": reportID, "result": raw})
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
	var chapterID uint64
	if raw := strings.TrimSpace(c.Query("chapter_id")); raw != "" {
		var err error
		chapterID, err = strconv.ParseUint(raw, 10, 64)
		if err != nil || chapterID == 0 {
			response.Fail(c, response.CodeBadRequest, "invalid chapter_id")
			return
		}
	}
	rows, total, err := h.reports.AdminListAttempts(c.Request.Context(), reportID, chapterID, size, (page-1)*size)
	if err != nil {
		h.readError(c, err, "report call traces unavailable", reportID)
		return
	}
	h.auditRead(c, "view_llm_calls", reportID, "")
	response.OK(c, gin.H{"report_id": reportID, "items": rows, "total": total, "page": page, "page_size": size})
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

func (h *AdminReportTraceHandler) LLMCallValidation(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	callID, ok := reportTraceID(c, "callId")
	if !ok {
		return
	}
	raw, err := h.reports.AdminGetAttemptValidation(c.Request.Context(), reportID, callID)
	if err != nil {
		h.readError(c, err, "report call validation unavailable", reportID)
		return
	}
	h.auditRead(c, "view_llm_call_validation", reportID, strconv.FormatUint(callID, 10))
	response.OK(c, gin.H{"report_id": reportID, "call_id": callID, "validation_result": raw})
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

func (h *AdminReportTraceHandler) ChapterArtifact(c *gin.Context) {
	reportID, ok := reportTraceID(c, "id")
	if !ok {
		return
	}
	chapterID, ok := reportTraceID(c, "chapterId")
	if !ok {
		return
	}
	artifact := strings.TrimSpace(c.Query("type"))
	if artifact != "content" && artifact != "prompt" && artifact != "terminology" && artifact != "raw_output" && artifact != "validation" {
		response.Fail(c, response.CodeBadRequest, "invalid chapter artifact")
		return
	}
	row, err := h.reports.AdminGetChapterArtifact(c.Request.Context(), reportID, chapterID, artifact)
	if err != nil {
		h.readError(c, err, "report chapter artifact unavailable", reportID)
		return
	}
	h.auditRead(c, "view_report_chapter_artifact", reportID, strconv.FormatUint(chapterID, 10)+":"+artifact)
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
	raw, err := h.reports.AdminGetExecutionSection(c.Request.Context(), reportID, "facts_snapshot")
	if err != nil {
		h.readError(c, err, "report facts unavailable", reportID)
		return
	}
	var facts model.InterpretationFacts
	if err := json.Unmarshal(raw, &facts); err != nil {
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
