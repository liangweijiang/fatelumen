package handler

import (
	"encoding/json"
	"strconv"

	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/repository"

	"github.com/gin-gonic/gin"
)

type AdminSystemSettingHandler struct {
	settings    *repository.SystemSettingRepo
	audit       *repository.AuditRepo
	fallback    int
	environment string
}

func (h *AdminSystemSettingHandler) Audit(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	resource := c.Query("resource")
	allowed := map[string]bool{"": true, "llm_provider_config": true, "llm_model_config": true, "report_setting": true}
	if !allowed[resource] {
		response.Fail(c, response.CodeBadRequest, "不支持的审计资源")
		return
	}
	var items []model.AdminAuditLog
	var total int64
	var err error
	if resource == "" {
		items, total, err = h.audit.ListAuditResources(c.Request.Context(), []string{"llm_provider_config", "llm_model_config", "report_setting"}, size, (page-1)*size)
	} else {
		items, total, err = h.audit.ListAudit(c.Request.Context(), resource, size, (page-1)*size)
	}
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("list system setting audits failed", "err", err, "resource", resource)
		response.Error(c, "读取配置变更记录失败")
		return
	}
	response.OK(c, gin.H{"items": items, "total": total, "page": page, "page_size": size})
}

func NewAdminSystemSettingHandler(settings *repository.SystemSettingRepo, audit *repository.AuditRepo, fallback int, environment ...string) *AdminSystemSettingHandler {
	env := "unknown"
	if len(environment) > 0 && environment[0] != "" {
		env = environment[0]
	}
	return &AdminSystemSettingHandler{settings: settings, audit: audit, fallback: fallback, environment: env}
}

func (h *AdminSystemSettingHandler) GetReport(c *gin.Context) {
	value, err := h.settings.ReportChapterConcurrency(c.Request.Context(), h.fallback)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("read report settings failed", "err", err)
		response.Error(c, "读取报告设置失败")
		return
	}
	response.OK(c, gin.H{"chapter_concurrency": value})
}

func (h *AdminSystemSettingHandler) SaveReport(c *gin.Context) {
	var in struct {
		ChapterConcurrency int `json:"chapter_concurrency" binding:"required,min=1,max=10"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeBadRequest, "十章并发数必须为1～10")
		return
	}
	before, err := h.settings.ReportChapterConcurrency(c.Request.Context(), h.fallback)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("read report settings before update failed", "err", err)
		response.Error(c, "保存报告设置失败")
		return
	}
	if err := h.settings.SaveReportChapterConcurrency(c.Request.Context(), in.ChapterConcurrency); err != nil {
		logger.FromCtx(c.Request.Context()).Error("save report settings failed", "err", err)
		response.Error(c, "保存报告设置失败")
		return
	}
	if h.audit != nil {
		detail, _ := json.Marshal(gin.H{"environment": h.environment, "before": gin.H{"chapter_concurrency": before}, "after": gin.H{"chapter_concurrency": in.ChapterConcurrency}})
		h.audit.Write(c.Request.Context(), model.AdminAuditLog{AdminID: middleware.GetAdminID(c), AdminName: c.GetString("admin_name"), Action: "update", Resource: "report_setting", ResourceID: repository.ReportChapterConcurrencyKey, Detail: model.JSONRaw(detail), IP: c.ClientIP()})
	}
	response.OK(c, gin.H{"chapter_concurrency": in.ChapterConcurrency})
}
