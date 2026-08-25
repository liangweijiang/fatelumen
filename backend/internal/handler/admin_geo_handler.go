package handler

import (
	"encoding/json"
	"errors"
	"strconv"

	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminGeoHandler struct {
	repo  *repository.GeoRepo
	audit *repository.AuditRepo
}

func NewAdminGeoHandler(repo *repository.GeoRepo, audit *repository.AuditRepo) *AdminGeoHandler {
	return &AdminGeoHandler{repo: repo, audit: audit}
}

func (h *AdminGeoHandler) List(c *gin.Context) {
	page, size := geoPage(c)
	rows, total, err := h.repo.AdminCities(c.Request.Context(), c.Query("q"), page, size)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("list admin geo cities failed", "err", err)
		response.Error(c, "geo data unavailable")
		return
	}
	response.OK(c, gin.H{"items": rows, "total": total, "page": page, "page_size": size})
}

func (h *AdminGeoHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid geo id")
		return
	}
	var in struct {
		Latitude  *float64 `json:"latitude" binding:"required,gte=-90,lte=90"`
		Longitude *float64 `json:"longitude" binding:"required,gte=-180,lte=180"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid latitude or longitude")
		return
	}
	before, err := h.repo.City(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, response.CodeNotFound, "geo data not found")
			return
		}
		logger.FromCtx(c.Request.Context()).Error("get geo city before update failed", "err", err, "geo_name_id", id)
		response.Error(c, "geo data unavailable")
		return
	}
	row, err := h.repo.UpdateCoordinates(c.Request.Context(), id, *in.Latitude, *in.Longitude)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("update geo coordinates failed", "err", err, "geo_name_id", id)
		response.Error(c, "update geo coordinates failed")
		return
	}
	detail, _ := json.Marshal(gin.H{
		"before": gin.H{"latitude": before.Latitude, "longitude": before.Longitude},
		"after":  gin.H{"latitude": row.Latitude, "longitude": row.Longitude},
	})
	h.audit.Write(c.Request.Context(), model.AdminAuditLog{
		AdminID: middleware.GetAdminID(c), AdminName: c.GetString("admin_name"),
		Action: "update_coordinates", Resource: "geo_city", ResourceID: strconv.FormatUint(id, 10),
		Detail: model.JSONRaw(detail), IP: c.ClientIP(),
	})
	response.OK(c, row)
}
