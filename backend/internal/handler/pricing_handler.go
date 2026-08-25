package handler

import (
	"encoding/json"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

type PricingHandler struct{ db *gorm.DB }
type pricingInput struct {
	SKU                 string   `json:"sku"`
	Locale              string   `json:"locale"`
	Title               string   `json:"title"`
	Summary             string   `json:"summary"`
	Benefits            []string `json:"benefits"`
	AmountCents         int      `json:"amount_cents"`
	OriginalAmountCents int      `json:"original_amount_cents"`
	ReportCount         int      `json:"report_count"`
	CreditsGranted      int      `json:"credits_granted"`
	Currency            string   `json:"currency"`
	Enabled             bool     `json:"enabled"`
	SortOrder           int      `json:"sort_order"`
}

func NewPricingHandler(db *gorm.DB) *PricingHandler { return &PricingHandler{db: db} }
func (h *PricingHandler) List(c *gin.Context) {
	h.list(c, c.Query("public") == "1")
}
func (h *PricingHandler) PublicList(c *gin.Context) { h.list(c, true) }
func (h *PricingHandler) list(c *gin.Context, enabledOnly bool) {
	var v []model.PricingPlan
	q := h.db
	if locale := c.Query("locale"); locale != "" {
		q = q.Where("locale = ?", locale)
	}
	if enabledOnly {
		q = q.Where("enabled = ?", true)
	}
	if err := q.Order("sort_order,id").Find(&v).Error; err != nil {
		response.Error(c, "pricing query failed")
		return
	}
	response.OK(c, v)
}
func (h *PricingHandler) Create(c *gin.Context) {
	var in pricingInput
	if c.ShouldBindJSON(&in) != nil || strings.TrimSpace(in.SKU) == "" || !validLocale(in.Locale) || strings.TrimSpace(in.Title) == "" || in.AmountCents < 0 || in.OriginalAmountCents < 0 || in.ReportCount < 0 || in.CreditsGranted < 0 {
		response.Fail(c, response.CodeBadRequest, "invalid plan")
		return
	}
	in.SKU = strings.TrimSpace(in.SKU)
	in.Title = strings.TrimSpace(in.Title)
	in.Currency = strings.ToLower(strings.TrimSpace(in.Currency))
	if in.Currency == "" {
		in.Currency = "usd"
	}
	var duplicate int64
	if err := h.db.Model(&model.PricingPlan{}).Where("sku = ? AND locale = ?", in.SKU, in.Locale).Count(&duplicate).Error; err != nil {
		logger.FromCtx(c.Request.Context()).Error("pricing uniqueness query failed", "err", err, "sku", in.SKU, "locale", in.Locale)
		response.Error(c, "create failed")
		return
	}
	if duplicate > 0 {
		response.Fail(c, response.CodeBadRequest, "sku and locale already exist")
		return
	}
	benefits, _ := json.Marshal(normalizeTags(in.Benefits))
	v := model.PricingPlan{SKU: in.SKU, Locale: in.Locale, Title: in.Title, Summary: in.Summary, Benefits: benefits, AmountCents: in.AmountCents, OriginalAmountCents: in.OriginalAmountCents, ReportCount: in.ReportCount, CreditsGranted: in.CreditsGranted, Currency: in.Currency, Enabled: in.Enabled, SortOrder: in.SortOrder}
	v.ID = 0
	if e := h.db.Create(&v).Error; e != nil {
		response.Error(c, "create failed")
		return
	}
	h.audit(c, "create", v.ID)
	response.OK(c, v)
}
func (h *PricingHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var in pricingInput
	if c.ShouldBindJSON(&in) != nil || !validLocale(in.Locale) || strings.TrimSpace(in.Title) == "" || in.AmountCents < 0 || in.OriginalAmountCents < 0 || in.ReportCount < 0 || in.CreditsGranted < 0 {
		response.Fail(c, response.CodeBadRequest, "invalid plan")
		return
	}
	in.SKU = strings.TrimSpace(in.SKU)
	in.Title = strings.TrimSpace(in.Title)
	in.Currency = strings.ToLower(strings.TrimSpace(in.Currency))
	if in.SKU == "" || in.Currency == "" {
		response.Fail(c, response.CodeBadRequest, "sku and currency are required")
		return
	}
	benefits, _ := json.Marshal(normalizeTags(in.Benefits))
	if e := h.db.Model(&model.PricingPlan{}).Where("id=?", id).Updates(map[string]interface{}{"sku": in.SKU, "locale": in.Locale, "title": in.Title, "summary": in.Summary, "benefits": benefits, "amount_cents": in.AmountCents, "original_amount_cents": in.OriginalAmountCents, "report_count": in.ReportCount, "credits_granted": in.CreditsGranted, "currency": in.Currency, "enabled": in.Enabled, "sort_order": in.SortOrder}).Error; e != nil {
		response.Error(c, "update failed")
		return
	}
	h.audit(c, "update", id)
	h.Detail(c)
}
func (h *PricingHandler) Detail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var v model.PricingPlan
	if h.db.First(&v, id).Error != nil {
		response.Fail(c, response.CodeNotFound, "not found")
		return
	}
	response.OK(c, v)
}
func (h *PricingHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.db.Delete(&model.PricingPlan{}, id).Error; err != nil {
		logger.FromCtx(c.Request.Context()).Error("pricing delete failed", "err", err, "pricing_id", id)
		response.Error(c, "delete failed")
		return
	}
	h.audit(c, "delete", id)
	response.OK(c, gin.H{"deleted": true})
}

func (h *PricingHandler) audit(c *gin.Context, action string, id uint64) {
	adminID := middleware.GetAdminID(c)
	if adminID == 0 {
		return
	}
	if err := h.db.Create(&model.AdminAuditLog{AdminID: adminID, AdminName: c.GetString("admin_name"), Action: action, Resource: "pricing", ResourceID: strconv.FormatUint(id, 10), IP: c.ClientIP()}).Error; err != nil {
		logger.FromCtx(c.Request.Context()).Error("pricing audit write failed", "err", err, "pricing_id", id, "action", action)
	}
}
