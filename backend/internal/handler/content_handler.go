package handler

import (
	"encoding/json"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

const maxMarkdownUploadBytes = 2 << 20

// ContentHandler manages the three explicitly supported content collections.
type ContentHandler struct{ db *gorm.DB }

type contentInput struct {
	ContentKey string   `json:"content_key"`
	Slug       string   `json:"slug"`
	Type       string   `json:"type"`
	Category   string   `json:"category"`
	CoverURL   string   `json:"cover_url"`
	Locale     string   `json:"locale"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Tags       []string `json:"tags"`
	Markdown   string   `json:"markdown"`
	SortOrder  int      `json:"sort_order"`
	Pinned     bool     `json:"pinned"`
}

func NewContentHandler(db *gorm.DB) *ContentHandler { return &ContentHandler{db: db} }
func (h *ContentHandler) List(c *gin.Context) {
	h.list(c, c.Query("public") == "1")
}

func (h *ContentHandler) list(c *gin.Context, publishedOnly bool) {
	typ := c.Param("type")
	if !validType(typ) {
		response.Fail(c, response.CodeBadRequest, "invalid content type")
		return
	}
	var rows []model.ContentItem
	q := h.db.Where("type = ?", typ)
	if v := c.Query("locale"); v != "" {
		if publishedOnly && v != "en" {
			q = q.Where("locale = ? OR (locale = ? AND content_key NOT IN (SELECT content_key FROM content_items WHERE type = ? AND locale = ? AND status = ?))", v, "en", typ, v, "published")
		} else {
			q = q.Where("locale = ?", v)
		}
	}
	if publishedOnly {
		q = q.Where("status = ?", "published")
	}
	if err := q.Order("pinned desc,sort_order asc,id desc").Find(&rows).Error; err != nil {
		response.Error(c, "content query failed")
		return
	}
	response.OK(c, rows)
}

func (h *ContentHandler) PublicList(c *gin.Context) {
	h.list(c, true)
}
func (h *ContentHandler) PublicDetail(c *gin.Context) {
	typ, slug, locale := c.Param("type"), c.Param("slug"), c.DefaultQuery("locale", "en")
	if !validType(typ) || !validLocale(locale) {
		response.Fail(c, response.CodeBadRequest, "invalid content query")
		return
	}
	var v model.ContentItem
	err := h.db.Where("type = ? AND slug = ? AND locale = ? AND status = ?", typ, slug, locale, "published").First(&v).Error
	if err != nil && locale != "en" {
		err = h.db.Where("type = ? AND slug = ? AND locale = ? AND status = ?", typ, slug, "en", "published").First(&v).Error
	}
	if err != nil {
		response.Fail(c, response.CodeNotFound, "not found")
		return
	}
	response.OK(c, v)
}
func (h *ContentHandler) Create(c *gin.Context) {
	var in contentInput
	if c.ShouldBindJSON(&in) != nil || !validType(in.Type) {
		response.Fail(c, response.CodeBadRequest, "invalid content")
		return
	}
	in.Locale = strings.TrimSpace(in.Locale)
	in.Title = strings.TrimSpace(in.Title)
	in.Markdown = strings.TrimSpace(in.Markdown)
	tags, _ := json.Marshal(normalizeTags(in.Tags))
	in.Slug = normalizeSlug(in.Slug)
	markdown := sanitizeMarkdown(in.Markdown)
	v := model.ContentItem{ContentKey: in.ContentKey, Slug: in.Slug, Type: in.Type, Category: strings.TrimSpace(in.Category), CoverURL: strings.TrimSpace(in.CoverURL), Locale: in.Locale, Title: in.Title, Summary: resolveContentSummary(in.Summary, markdown), Tags: tags, Markdown: markdown, SortOrder: in.SortOrder, Pinned: in.Pinned}
	if v.ContentKey == "" {
		v.ContentKey = randomHex(12)
	}
	if v.Slug == "" {
		v.Slug = v.ContentKey
	}
	if !validLocale(v.Locale) || v.Title == "" || v.Markdown == "" {
		response.Fail(c, response.CodeBadRequest, "title, locale and markdown are required")
		return
	}
	v.Status = "draft"
	if err := h.db.Create(&v).Error; err != nil {
		response.Error(c, "create failed")
		return
	}
	h.audit(c, "create", v.ID)
	response.OK(c, v)
}
func (h *ContentHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var in contentInput
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, response.CodeBadRequest, "invalid content")
		return
	}
	in.Locale = strings.TrimSpace(in.Locale)
	in.Title = strings.TrimSpace(in.Title)
	in.Markdown = strings.TrimSpace(in.Markdown)
	if !validLocale(in.Locale) || in.Title == "" || in.Markdown == "" {
		response.Fail(c, response.CodeBadRequest, "title, locale and markdown are required")
		return
	}
	tags, _ := json.Marshal(normalizeTags(in.Tags))
	in.Slug = normalizeSlug(in.Slug)
	if in.Slug == "" {
		in.Slug = strconv.FormatUint(id, 10)
	}
	markdown := sanitizeMarkdown(in.Markdown)
	if err := h.db.Model(&model.ContentItem{}).Where("id = ?", id).Updates(map[string]interface{}{"slug": in.Slug, "category": strings.TrimSpace(in.Category), "cover_url": strings.TrimSpace(in.CoverURL), "title": in.Title, "summary": resolveContentSummary(in.Summary, markdown), "tags": tags, "markdown": markdown, "locale": in.Locale, "sort_order": in.SortOrder, "pinned": in.Pinned}).Error; err != nil {
		response.Error(c, "update failed")
		return
	}
	h.audit(c, "update", id)
	h.Detail(c)
}
func (h *ContentHandler) Detail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var v model.ContentItem
	if h.db.First(&v, id).Error != nil {
		response.Fail(c, response.CodeNotFound, "not found")
		return
	}
	response.OK(c, v)
}
func (h *ContentHandler) State(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var v struct {
		Status string `json:"status"`
	}
	if c.ShouldBindJSON(&v) != nil || (v.Status != "published" && v.Status != "draft" && v.Status != "preview" && v.Status != "unpublished") {
		response.Fail(c, response.CodeBadRequest, "invalid status")
		return
	}
	updates := map[string]interface{}{"status": v.Status}
	if v.Status == "published" {
		updates["published_at"] = gorm.Expr("CURRENT_TIMESTAMP")
	}
	if err := h.db.Model(&model.ContentItem{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		logger.FromCtx(c.Request.Context()).Error("content state update failed", "err", err, "content_id", id)
		response.Error(c, "state update failed")
		return
	}
	h.audit(c, "state:"+v.Status, id)
	response.OK(c, gin.H{"id": id, "status": v.Status})
}

// ImportMarkdown validates and parses a Markdown file without persisting it.
// The operator can review and edit every returned field before saving content.
func (h *ContentHandler) ImportMarkdown(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "markdown file is required")
		return
	}
	if strings.ToLower(filepath.Ext(fileHeader.Filename)) != ".md" || fileHeader.Size <= 0 || fileHeader.Size > maxMarkdownUploadBytes {
		response.Fail(c, response.CodeBadRequest, "only non-empty .md files up to 2 MB are allowed")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("markdown upload open failed", "err", err, "size", fileHeader.Size)
		response.Error(c, "markdown import failed")
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxMarkdownUploadBytes+1))
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("markdown upload read failed", "err", err, "size", fileHeader.Size)
		response.Error(c, "markdown import failed")
		return
	}
	if len(raw) == 0 || len(raw) > maxMarkdownUploadBytes {
		response.Fail(c, response.CodeBadRequest, "invalid markdown file size")
		return
	}
	markdown := sanitizeMarkdown(string(raw))
	if markdown == "" {
		response.Fail(c, response.CodeBadRequest, "markdown file is empty")
		return
	}
	response.OK(c, gin.H{
		"filename": fileHeader.Filename,
		"title":    markdownTitle(markdown, fileHeader.Filename),
		"markdown": markdown,
		"summary":  contentSummary(markdown),
	})
}

type contentBatchInput struct {
	ContentKeys []string `json:"content_keys"`
	Pinned      bool     `json:"pinned"`
}

func (h *ContentHandler) BatchPin(c *gin.Context) {
	typ := c.Param("type")
	var in contentBatchInput
	if !validType(typ) || c.ShouldBindJSON(&in) != nil {
		response.Fail(c, response.CodeBadRequest, "invalid batch request")
		return
	}
	keys := normalizeContentKeys(in.ContentKeys)
	if len(keys) == 0 {
		response.Fail(c, response.CodeBadRequest, "content_keys are required")
		return
	}
	var rows []model.ContentItem
	if err := h.db.Where("type = ? AND content_key IN ?", typ, keys).Find(&rows).Error; err != nil {
		logger.FromCtx(c.Request.Context()).Error("content batch pin query failed", "err", err, "content_type", typ, "key_count", len(keys))
		response.Error(c, "batch pin failed")
		return
	}
	result := h.db.Model(&model.ContentItem{}).Where("type = ? AND content_key IN ?", typ, keys).Update("pinned", in.Pinned)
	if result.Error != nil {
		logger.FromCtx(c.Request.Context()).Error("content batch pin failed", "err", result.Error, "content_type", typ, "key_count", len(keys))
		response.Error(c, "batch pin failed")
		return
	}
	for _, row := range rows {
		h.audit(c, "batch_pin:"+strconv.FormatBool(in.Pinned), row.ID)
	}
	response.OK(c, gin.H{"updated": result.RowsAffected, "pinned": in.Pinned})
}

func (h *ContentHandler) BatchDelete(c *gin.Context) {
	typ := c.Param("type")
	var in contentBatchInput
	if !validType(typ) || c.ShouldBindJSON(&in) != nil {
		response.Fail(c, response.CodeBadRequest, "invalid batch request")
		return
	}
	keys := normalizeContentKeys(in.ContentKeys)
	if len(keys) == 0 {
		response.Fail(c, response.CodeBadRequest, "content_keys are required")
		return
	}
	var rows []model.ContentItem
	if err := h.db.Where("type = ? AND content_key IN ?", typ, keys).Find(&rows).Error; err != nil {
		logger.FromCtx(c.Request.Context()).Error("content batch delete query failed", "err", err, "content_type", typ, "key_count", len(keys))
		response.Error(c, "batch delete failed")
		return
	}
	result := h.db.Where("type = ? AND content_key IN ?", typ, keys).Delete(&model.ContentItem{})
	if result.Error != nil {
		logger.FromCtx(c.Request.Context()).Error("content batch delete failed", "err", result.Error, "content_type", typ, "key_count", len(keys))
		response.Error(c, "batch delete failed")
		return
	}
	for _, row := range rows {
		h.audit(c, "batch_delete", row.ID)
	}
	response.OK(c, gin.H{"deleted": result.RowsAffected})
}
func (h *ContentHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if h.db.Delete(&model.ContentItem{}, id).Error != nil {
		response.Error(c, "delete failed")
		return
	}
	h.audit(c, "delete", id)
	response.OK(c, gin.H{"deleted": true})
}
func validType(v string) bool       { return v == "knowledge" || v == "faq" || v == "case" }
func validLocale(v string) bool     { return v == "en" || v == "zh" || v == "ja" || v == "ko" }
func normalizeSlug(v string) string { return strings.ToLower(strings.Trim(strings.TrimSpace(v), "/")) }
func sanitizeMarkdown(v string) string {
	r := strings.NewReplacer("<script", "&lt;script", "</script", "&lt;/script", "javascript:", "", "onerror=", "", "onclick=", "")
	return r.Replace(strings.TrimSpace(v))
}

// contentSummary provides the default. Operators may override it when context needs refinement.
func contentSummary(markdown string) string {
	plain := strings.Join(strings.Fields(strings.NewReplacer("#", " ", "*", " ", "`", " ", ">", " ").Replace(markdown)), " ")
	runes := []rune(plain)
	if len(runes) > 100 {
		return string(runes[:100])
	}
	return plain
}

func resolveContentSummary(summary, markdown string) string {
	if summary = strings.TrimSpace(summary); summary != "" {
		return summary
	}
	return contentSummary(markdown)
}

func markdownTitle(markdown, filename string) string {
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
}

func normalizeContentKeys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result
}

func normalizeTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		if clean := strings.TrimSpace(tag); clean != "" {
			result = append(result, clean)
		}
	}
	return result
}

func (h *ContentHandler) audit(c *gin.Context, action string, id uint64) {
	adminID := middleware.GetAdminID(c)
	if adminID == 0 {
		return
	}
	if err := h.db.Create(&model.AdminAuditLog{AdminID: adminID, AdminName: c.GetString("admin_name"), Action: action, Resource: "content", ResourceID: strconv.FormatUint(id, 10), IP: c.ClientIP()}).Error; err != nil {
		logger.FromCtx(c.Request.Context()).Error("content audit write failed", "err", err, "content_id", id, "action", action)
	}
}
