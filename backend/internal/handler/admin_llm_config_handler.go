package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminLLMConfigHandler struct {
	db           *gorm.DB
	audit        *repository.AuditRepo
	secretCipher *llm.ConfigSecretCipher
}

func NewAdminLLMConfigHandler(db *gorm.DB, audit *repository.AuditRepo, encryptionSecret string) *AdminLLMConfigHandler {
	return &AdminLLMConfigHandler{db: db, audit: audit, secretCipher: llm.NewConfigSecretCipher(encryptionSecret)}
}

type providerInput struct {
	Code    string `json:"code" binding:"max=64"`
	Name    string `json:"name" binding:"required,max=100"`
	BaseURL string `json:"base_url" binding:"required,max=500"`
	APIKey  string `json:"api_key"`
	Enabled *bool  `json:"enabled"`
}

type modelInput struct {
	ProviderID uint64 `json:"provider_id" binding:"required"`
	Name       string `json:"name" binding:"required,max=120"`
	ModelID    string `json:"model_id" binding:"required,max=200"`
	Priority   int    `json:"priority" binding:"min=1,max=9999"`
	MaxRetries int    `json:"max_retries" binding:"min=0,max=10"`
	Enabled    *bool  `json:"enabled"`
}

func (h *AdminLLMConfigHandler) Catalog(c *gin.Context) {
	response.OK(c, gin.H{"providers": llm.ProviderCatalog()})
}

func (h *AdminLLMConfigHandler) ListProviders(c *gin.Context) {
	page, size := llmPageArgs(c)
	query := h.db.WithContext(c.Request.Context()).Model(&model.LLMProviderConfig{})
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		like := "%" + q + "%"
		query = query.Where("name LIKE ? OR code LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.FromCtx(c).Error("list llm providers count failed", "err", err)
		response.Error(c, "读取供应商失败")
		return
	}
	var rows []model.LLMProviderConfig
	if err := query.Order("id ASC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		logger.FromCtx(c).Error("list llm providers failed", "err", err)
		response.Error(c, "读取供应商失败")
		return
	}
	response.OK(c, gin.H{"items": rows, "total": total, "page": page, "page_size": size})
}

func (h *AdminLLMConfigHandler) CreateProvider(c *gin.Context) {
	var in providerInput
	if err := c.ShouldBindJSON(&in); err != nil || !validBaseURL(in.BaseURL) || strings.TrimSpace(in.APIKey) == "" {
		response.Fail(c, response.CodeBadRequest, "供应商名称、有效接口地址和 API Key 为必填项")
		return
	}
	ciphertext, err := h.secretCipher.Encrypt(strings.TrimSpace(in.APIKey))
	if err != nil {
		logger.FromCtx(c).Error("encrypt llm provider key failed", "err", err)
		response.Error(c, "保存供应商失败")
		return
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	code := normalizedCode(in.Code)
	if code == "" || code == "custom" {
		code, err = newCustomProviderCode()
		if err != nil {
			logger.FromCtx(c).Error("generate llm provider code failed", "err", err)
			response.Error(c, "保存供应商失败")
			return
		}
	}
	row := model.LLMProviderConfig{Code: code, Name: strings.TrimSpace(in.Name), BaseURL: strings.TrimRight(strings.TrimSpace(in.BaseURL), "/"), APIKeyCiphertext: ciphertext, APIKeyHint: keyHint(in.APIKey), Enabled: enabled}
	if err := h.db.WithContext(c.Request.Context()).Create(&row).Error; err != nil {
		logger.FromCtx(c).Error("create llm provider failed", "err", err)
		response.Error(c, "保存供应商失败")
		return
	}
	h.writeAudit(c, "create", "llm_provider_config", row.ID)
	response.OK(c, row)
}

func (h *AdminLLMConfigHandler) UpdateProvider(c *gin.Context) {
	id, ok := llmID(c)
	if !ok {
		return
	}
	var row model.LLMProviderConfig
	if err := h.db.WithContext(c).First(&row, id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, response.CodeNotFound, "供应商不存在")
		return
	} else if err != nil {
		logger.FromCtx(c).Error("get llm provider failed", "err", err, "provider_id", id)
		response.Error(c, "读取供应商失败")
		return
	}
	var in providerInput
	if err := c.ShouldBindJSON(&in); err != nil || !validBaseURL(in.BaseURL) {
		response.Fail(c, response.CodeBadRequest, "供应商配置不完整")
		return
	}
	// Code is an immutable internal identifier. Changing it would detach this
	// provider from its preset model catalogue and frozen routing references.
	row.Name, row.BaseURL = strings.TrimSpace(in.Name), strings.TrimRight(strings.TrimSpace(in.BaseURL), "/")
	if in.Enabled != nil {
		row.Enabled = *in.Enabled
	}
	if strings.TrimSpace(in.APIKey) != "" {
		ciphertext, err := h.secretCipher.Encrypt(strings.TrimSpace(in.APIKey))
		if err != nil {
			logger.FromCtx(c).Error("encrypt llm provider key failed", "err", err, "provider_id", id)
			response.Error(c, "保存供应商失败")
			return
		}
		row.APIKeyCiphertext, row.APIKeyHint = ciphertext, keyHint(in.APIKey)
	}
	if err := h.db.WithContext(c).Save(&row).Error; err != nil {
		logger.FromCtx(c).Error("update llm provider failed", "err", err, "provider_id", id)
		response.Error(c, "保存供应商失败")
		return
	}
	h.writeAudit(c, "update", "llm_provider_config", row.ID)
	response.OK(c, row)
}

func (h *AdminLLMConfigHandler) DeleteProvider(c *gin.Context) {
	id, ok := llmID(c)
	if !ok {
		return
	}
	var count int64
	if err := h.db.WithContext(c).Model(&model.LLMModelConfig{}).Where("provider_id = ?", id).Count(&count).Error; err != nil {
		logger.FromCtx(c).Error("count provider models failed", "err", err, "provider_id", id)
		response.Error(c, "删除供应商失败")
		return
	}
	if count > 0 {
		response.Fail(c, response.CodeBadRequest, "该供应商仍关联模型，请先删除关联模型")
		return
	}
	result := h.db.WithContext(c).Delete(&model.LLMProviderConfig{}, id)
	if result.Error != nil {
		logger.FromCtx(c).Error("delete llm provider failed", "err", result.Error, "provider_id", id)
		response.Error(c, "删除供应商失败")
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c, response.CodeNotFound, "供应商不存在")
		return
	}
	h.writeAudit(c, "delete", "llm_provider_config", id)
	response.OK(c, gin.H{"deleted": id})
}

func (h *AdminLLMConfigHandler) ListModels(c *gin.Context) {
	page, size := llmPageArgs(c)
	query := h.db.WithContext(c).Model(&model.LLMModelConfig{})
	if providerID, _ := strconv.ParseUint(c.Query("provider_id"), 10, 64); providerID > 0 {
		query = query.Where("provider_id = ?", providerID)
	}
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		like := "%" + q + "%"
		query = query.Where("name LIKE ? OR model_id LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.FromCtx(c).Error("list llm models count failed", "err", err)
		response.Error(c, "读取模型失败")
		return
	}
	var rows []model.LLMModelConfig
	if err := query.Preload("Provider").Order("priority ASC, id ASC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		logger.FromCtx(c).Error("list llm models failed", "err", err)
		response.Error(c, "读取模型失败")
		return
	}
	response.OK(c, gin.H{"items": rows, "total": total, "page": page, "page_size": size})
}

func (h *AdminLLMConfigHandler) CreateModel(c *gin.Context) { h.saveModel(c, 0) }
func (h *AdminLLMConfigHandler) UpdateModel(c *gin.Context) {
	id, ok := llmID(c)
	if ok {
		h.saveModel(c, id)
	}
}

func (h *AdminLLMConfigHandler) saveModel(c *gin.Context, id uint64) {
	var in modelInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeBadRequest, "模型配置不完整")
		return
	}
	var provider model.LLMProviderConfig
	if err := h.db.WithContext(c).First(&provider, in.ProviderID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, response.CodeBadRequest, "所选供应商不存在")
		return
	} else if err != nil {
		logger.FromCtx(c).Error("get model provider failed", "err", err, "provider_id", in.ProviderID)
		response.Error(c, "读取供应商失败")
		return
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	row := model.LLMModelConfig{ID: id, ProviderID: in.ProviderID, Name: strings.TrimSpace(in.Name), ModelID: strings.TrimSpace(in.ModelID), Priority: in.Priority, MaxRetries: in.MaxRetries, Enabled: enabled}
	if id == 0 {
		if err := h.db.WithContext(c).Create(&row).Error; err != nil {
			logger.FromCtx(c).Error("create llm model failed", "err", err, "provider_id", in.ProviderID)
			response.Error(c, "保存模型失败")
			return
		}
	} else {
		var existing model.LLMModelConfig
		if err := h.db.WithContext(c).First(&existing, id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, response.CodeNotFound, "模型不存在")
			return
		} else if err != nil {
			logger.FromCtx(c).Error("get llm model failed", "err", err, "model_config_id", id)
			response.Error(c, "读取模型失败")
			return
		}
		if in.Enabled == nil {
			row.Enabled = existing.Enabled
		}
		if err := h.db.WithContext(c).Model(&existing).Updates(map[string]any{"provider_id": row.ProviderID, "name": row.Name, "model_id": row.ModelID, "priority": row.Priority, "max_retries": row.MaxRetries, "enabled": row.Enabled}).Error; err != nil {
			logger.FromCtx(c).Error("update llm model failed", "err", err, "model_config_id", id)
			response.Error(c, "保存模型失败")
			return
		}
	}
	h.writeAudit(c, map[bool]string{true: "update", false: "create"}[id > 0], "llm_model_config", row.ID)
	if err := h.db.WithContext(c).Preload("Provider").First(&row, row.ID).Error; err != nil {
		logger.FromCtx(c).Error("reload llm model failed", "err", err, "model_config_id", row.ID)
		response.Error(c, "读取模型失败")
		return
	}
	response.OK(c, row)
}

func (h *AdminLLMConfigHandler) DeleteModel(c *gin.Context) {
	id, ok := llmID(c)
	if !ok {
		return
	}
	result := h.db.WithContext(c).Delete(&model.LLMModelConfig{}, id)
	if result.Error != nil {
		logger.FromCtx(c).Error("delete llm model failed", "err", result.Error, "model_config_id", id)
		response.Error(c, "删除模型失败")
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c, response.CodeNotFound, "模型不存在")
		return
	}
	h.writeAudit(c, "delete", "llm_model_config", id)
	response.OK(c, gin.H{"deleted": id})
}

func (h *AdminLLMConfigHandler) writeAudit(c *gin.Context, action, resource string, id uint64) {
	if h.audit == nil {
		return
	}
	h.audit.Write(c.Request.Context(), model.AdminAuditLog{AdminID: middleware.GetAdminID(c), AdminName: c.GetString("admin_name"), Action: action, Resource: resource, ResourceID: strconv.FormatUint(id, 10), IP: c.ClientIP()})
}

func llmPageArgs(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	return page, size
}
func llmID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, response.CodeBadRequest, "无效编号")
		return 0, false
	}
	return id, true
}
func validBaseURL(value string) bool {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(value))
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}
func normalizedCode(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), " ", "-"))
}
func newCustomProviderCode() (string, error) {
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return "custom-" + hex.EncodeToString(random), nil
}
func keyHint(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 4 {
		return "••••"
	}
	return "••••••••" + key[len(key)-4:]
}
