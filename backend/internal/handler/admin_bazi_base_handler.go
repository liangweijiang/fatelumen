package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	"fatelumen/backend/internal/bazi/basedata"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/repository"

	"github.com/gin-gonic/gin"
)

type AdminBaziBaseHandler struct {
	catalog  basedata.Catalog
	calendar *repository.AnnualCalendarRepo
}

func NewAdminBaziBaseHandler(c basedata.Catalog, calendars ...*repository.AnnualCalendarRepo) *AdminBaziBaseHandler {
	h := &AdminBaziBaseHandler{catalog: c}
	if len(calendars) > 0 {
		h.calendar = calendars[0]
	}
	return h
}

func (h *AdminBaziBaseHandler) Summary(c *gin.Context) {
	if err := h.catalog.Validate(); err != nil {
		logger.FromCtx(c.Request.Context()).Error("bazi base data validation failed", "err", err, "version", h.catalog.Version)
	}
	response.OK(c, h.catalog.Summary())
}
func (h *AdminBaziBaseHandler) Elements(c *gin.Context) { h.list(c, toAny(h.catalog.Elements)) }
func (h *AdminBaziBaseHandler) Stems(c *gin.Context)    { h.list(c, toAny(h.catalog.Stems)) }
func (h *AdminBaziBaseHandler) Branches(c *gin.Context) { h.list(c, toAny(h.catalog.Branches)) }
func (h *AdminBaziBaseHandler) TenGods(c *gin.Context)  { h.list(c, toAny(h.catalog.TenGodRules)) }
func (h *AdminBaziBaseHandler) Relations(c *gin.Context) {
	items := h.catalog.Relations
	if typ := strings.TrimSpace(c.Query("type")); typ != "" {
		filtered := make([]basedata.Relation, 0)
		for _, v := range items {
			if v.Type == typ {
				filtered = append(filtered, v)
			}
		}
		items = filtered
	}
	h.list(c, toAny(items))
}
func (h *AdminBaziBaseHandler) StrengthRules(c *gin.Context) {
	items := []any{}
	for _, v := range h.catalog.PositionWeights {
		items = append(items, gin.H{"kind": "position_weight", "code": v.Code, "value": v.Weight, "names": v.Names})
	}
	for _, v := range h.catalog.Thresholds {
		items = append(items, gin.H{"kind": "threshold", "code": v.Code, "value": v.Value, "names": v.Names})
	}
	for month, row := range h.catalog.MonthScores {
		for day, value := range row {
			items = append(items, gin.H{"kind": "month_score", "code": month + "_to_" + day, "month_element": month, "day_element": day, "value": value, "names": basedata.Names{ZH: month + "月令 → " + day + "日主", EN: month + " month → " + day + " day master", JA: month + "月令 → " + day + "日主", KO: month + " 월령 → " + day + " 일간"}})
		}
	}
	h.list(c, items)
}

func (h *AdminBaziBaseHandler) AnnualCalendar(c *gin.Context) {
	if h.calendar == nil {
		response.Error(c, "干支日历数据源未配置")
		return
	}
	page, size := geoPage(c)
	startYear, _ := strconv.Atoi(c.Query("start_year"))
	endYear, _ := strconv.Atoi(c.Query("end_year"))
	rows, total, err := h.calendar.List(c.Request.Context(), startYear, endYear, page, size)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("list annual calendar failed", "err", err, "start_year", startYear, "end_year", endYear)
		response.Error(c, "干支日历读取失败")
		return
	}
	response.OK(c, gin.H{"items": rows, "total": total, "page": page, "page_size": size, "version": "sexagenary-calendar-v1"})
}

func (h *AdminBaziBaseHandler) list(c *gin.Context, items []any) {
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	if q != "" {
		filtered := make([]any, 0)
		for _, v := range items {
			if strings.Contains(strings.ToLower(toSearchText(v)), q) {
				filtered = append(filtered, v)
			}
		}
		items = filtered
	}
	page, size := geoPage(c)
	total := len(items)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	response.OK(c, gin.H{"items": items[start:end], "total": total, "page": page, "page_size": size, "version": h.catalog.Version})
}
func toAny[T any](in []T) []any {
	out := make([]any, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}
func toSearchText(v any) string {
	return strings.ReplaceAll(strings.TrimSpace(strings.ToLower(strings.Join(strings.Fields(toJSON(v)), " "))), "\\u", "")
}
func toJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
