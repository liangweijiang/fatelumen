package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/repository"
	"fatelumen/backend/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminCalculationHandler struct {
	db     *gorm.DB
	engine birthchart.Engine
	audit  *repository.AuditRepo
}

func NewAdminCalculationHandler(db *gorm.DB, engine birthchart.Engine, audit *repository.AuditRepo) *AdminCalculationHandler {
	return &AdminCalculationHandler{db: db, engine: engine, audit: audit}
}

type calculationInput struct {
	Name         string                   `json:"name"`
	Gender       int8                     `json:"gender"`
	CalendarType int8                     `json:"calendar_type"`
	Year         int                      `json:"year"`
	Month        int                      `json:"month"`
	Day          int                      `json:"day"`
	Hour         int                      `json:"hour"`
	Minute       int                      `json:"minute"`
	IsLeapMonth  bool                     `json:"is_leap_month"`
	Locale       string                   `json:"locale"`
	Location     birthchart.LocationInput `json:"location"`
}

func (h *AdminCalculationHandler) List(c *gin.Context) {
	var items []model.CalculationArchive
	var total int64
	page, size := pageArgs(c)
	q := h.db.WithContext(c).Model(&model.CalculationArchive{})
	if s := strings.TrimSpace(c.Query("q")); s != "" {
		q = q.Where("name LIKE ?", "%"+s+"%")
	}
	if err := q.Count(&total).Error; err != nil {
		logger.FromCtx(c).Error("list calculation archives failed", "err", err)
		response.Error(c, "读取计算档案失败")
		return
	}
	if err := q.Order("updated_at DESC,id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		logger.FromCtx(c).Error("list calculation archives failed", "err", err)
		response.Error(c, "读取计算档案失败")
		return
	}
	response.OK(c, gin.H{"items": items, "total": total, "page": page, "page_size": size})
}
func (h *AdminCalculationHandler) Create(c *gin.Context) {
	var in calculationInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		response.Fail(c, response.CodeBadRequest, "档案名称和出生资料不能为空")
		return
	}
	raw, _ := json.Marshal(in)
	a := model.CalculationArchive{Name: strings.TrimSpace(in.Name), Status: "calculating", Input: model.JSONRaw(raw), CreatedByAdminID: middleware.GetAdminID(c)}
	if err := h.db.WithContext(c).Create(&a).Error; err != nil {
		logger.FromCtx(c).Error("create calculation archive failed", "err", err)
		response.Error(c, "新增计算档案失败")
		return
	}
	if err := h.calculate(c, &a, in); err != nil {
		response.Fail(c, response.CodeBadRequest, err.Error())
		return
	}
	h.writeAudit(c, "create", a.ID)
	response.OK(c, a)
}
func (h *AdminCalculationHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var a model.CalculationArchive
	if err := h.db.WithContext(c).First(&a, id).Error; err != nil {
		response.Fail(c, response.CodeNotFound, "计算档案不存在")
		return
	}
	var versions []model.CalculationVersion
	h.db.WithContext(c).Where("archive_id=?", id).Order("version_no DESC").Find(&versions)
	response.OK(c, gin.H{"archive": a, "versions": versions})
}
func (h *AdminCalculationHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var a model.CalculationArchive
	if h.db.WithContext(c).First(&a, id).Error != nil {
		response.Fail(c, response.CodeNotFound, "计算档案不存在")
		return
	}
	var in calculationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeBadRequest, "出生资料格式不正确")
		return
	}
	raw, _ := json.Marshal(in)
	a.Name = strings.TrimSpace(in.Name)
	a.Input = model.JSONRaw(raw)
	a.Status = "calculating"
	if err := h.db.WithContext(c).Save(&a).Error; err != nil {
		logger.FromCtx(c).Error("update calculation archive failed", "err", err, "archive_id", id)
		response.Error(c, "修改计算档案失败")
		return
	}
	if err := h.calculate(c, &a, in); err != nil {
		response.Fail(c, response.CodeBadRequest, err.Error())
		return
	}
	h.writeAudit(c, "update", id)
	response.OK(c, a)
}
func (h *AdminCalculationHandler) Recalculate(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var a model.CalculationArchive
	if h.db.WithContext(c).First(&a, id).Error != nil {
		response.Fail(c, response.CodeNotFound, "计算档案不存在")
		return
	}
	var in calculationInput
	if json.Unmarshal(a.Input, &in) != nil {
		response.Error(c, "档案输入无法读取")
		return
	}
	if err := h.calculate(c, &a, in); err != nil {
		response.Fail(c, response.CodeBadRequest, err.Error())
		return
	}
	h.writeAudit(c, "recalculate", id)
	response.OK(c, a)
}
func (h *AdminCalculationHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	err := h.db.WithContext(c).Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("archive_id=?", id).Delete(&model.CalculationVersion{}).Error; e != nil {
			return e
		}
		return tx.Delete(&model.CalculationArchive{}, id).Error
	})
	if err != nil {
		logger.FromCtx(c).Error("delete calculation archive failed", "err", err, "archive_id", id)
		response.Error(c, "删除计算档案失败")
		return
	}
	h.writeAudit(c, "delete", id)
	response.OK(c, gin.H{"deleted": 1})
}
func (h *AdminCalculationHandler) BatchDelete(c *gin.Context) {
	var in struct {
		IDs []uint64 `json:"ids"`
	}
	if c.ShouldBindJSON(&in) != nil || len(in.IDs) == 0 {
		response.Fail(c, response.CodeBadRequest, "请选择要删除的档案")
		return
	}
	err := h.db.WithContext(c).Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("archive_id IN ?", in.IDs).Delete(&model.CalculationVersion{}).Error; e != nil {
			return e
		}
		return tx.Where("id IN ?", in.IDs).Delete(&model.CalculationArchive{}).Error
	})
	if err != nil {
		logger.FromCtx(c).Error("batch delete calculation archives failed", "err", err)
		response.Error(c, "批量删除失败")
		return
	}
	h.writeAudit(c, "batch_delete", 0)
	response.OK(c, gin.H{"deleted": len(in.IDs)})
}

func (h *AdminCalculationHandler) calculate(c *gin.Context, a *model.CalculationArchive, in calculationInput) error {
	ctx := c.Request.Context()
	if in.Locale == "" {
		in.Locale = "zh"
	}
	bi := birthchart.Input{Gender: in.Gender, CalendarType: in.CalendarType, Year: in.Year, Month: in.Month, Day: in.Day, Hour: in.Hour, Minute: in.Minute, IsLeapMonth: in.IsLeapMonth, Location: in.Location}
	result, err := h.engine.Calculate(ctx, bi)
	if err != nil {
		a.Status = "failed"
		a.LastError = err.Error()
		h.db.WithContext(ctx).Save(a)
		logger.FromCtx(ctx).Error("calculate archive failed", "err", err, "archive_id", a.ID)
		return err
	}
	seed, _ := json.Marshal(struct {
		Input         calculationInput
		TrueSolarTime time.Time
	}{in, result.SolarTime.TrueSolarTime})
	sum := sha256.Sum256(seed)
	chartHash := hex.EncodeToString(sum[:])
	inputSnap, timeSnap, chartSnap, factSnap, err := service.BuildDeterministicSnapshot(bi, in.Locale, chartHash, result)
	if err != nil {
		return err
	}
	inputJSON, _ := json.Marshal(inputSnap)
	timeJSON, _ := json.Marshal(timeSnap)
	chartJSON, _ := json.Marshal(chartSnap)
	factsJSON, _ := json.Marshal(factSnap)
	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var max int
		tx.Model(&model.CalculationVersion{}).Where("archive_id=?", a.ID).Select("COALESCE(MAX(version_no),0)").Scan(&max)
		v := model.CalculationVersion{ArchiveID: a.ID, VersionNo: max + 1, InputSnapshot: model.JSONRaw(inputJSON), TimeCalculationSnapshot: model.JSONRaw(timeJSON), ChartSnapshot: model.JSONRaw(chartJSON), FactsSnapshot: model.JSONRaw(factsJSON), ChartHash: chartHash, FactsHash: factSnap.FactsHash}
		if e := tx.Create(&v).Error; e != nil {
			return e
		}
		return tx.Model(a).Updates(map[string]any{"status": "ready", "latest_version_id": v.ID, "latest_version_no": v.VersionNo, "last_error": ""}).Error
	})
	if err != nil {
		logger.FromCtx(ctx).Error("persist calculation version failed", "err", err, "archive_id", a.ID)
	}
	return err
}
func pageArgs(c *gin.Context) (int, int) {
	p, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	s, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if p < 1 {
		p = 1
	}
	if s < 1 || s > 100 {
		s = 20
	}
	return p, s
}
func parseID(c *gin.Context) (uint64, bool) {
	id, e := strconv.ParseUint(c.Param("id"), 10, 64)
	if e != nil {
		response.Fail(c, response.CodeBadRequest, "无效的档案编号")
		return 0, false
	}
	return id, true
}
func (h *AdminCalculationHandler) writeAudit(c *gin.Context, action string, id uint64) {
	h.audit.Write(c.Request.Context(), model.AdminAuditLog{AdminID: middleware.GetAdminID(c), AdminName: c.GetString("admin_name"), Action: action, Resource: "calculation_archive", ResourceID: strconv.FormatUint(id, 10), IP: c.ClientIP()})
}
