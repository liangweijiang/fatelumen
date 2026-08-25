package handler

import (
	"strconv"
	"strings"

	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/repository"
	"github.com/gin-gonic/gin"
)

type GeoHandler struct{ repo *repository.GeoRepo }

func NewGeoHandler(repo *repository.GeoRepo) *GeoHandler { return &GeoHandler{repo: repo} }

func geoPage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

func (h *GeoHandler) Countries(c *gin.Context) {
	page, size := geoPage(c)
	rows, total, err := h.repo.Countries(c.Query("q"), c.DefaultQuery("locale", "en"), page, size)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("list geo countries failed", "err", err)
		response.Error(c, "geo countries unavailable")
		return
	}
	response.OK(c, gin.H{"items": rows, "total": total, "page": page, "page_size": size})
}

func (h *GeoHandler) Cities(c *gin.Context) {
	country := strings.ToUpper(strings.TrimSpace(c.Query("country_code")))
	if len(country) != 2 {
		response.Fail(c, response.CodeBadRequest, "country_code is required")
		return
	}
	page, size := geoPage(c)
	rows, total, err := h.repo.Cities(country, c.Query("q"), c.DefaultQuery("locale", "en"), page, size)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("list geo cities failed", "err", err, "country_code", country)
		response.Error(c, "geo cities unavailable")
		return
	}
	response.OK(c, gin.H{"items": rows, "total": total, "page": page, "page_size": size})
}

func (h *GeoHandler) City(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid city id")
		return
	}
	row, err := h.repo.City(id)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "city not found")
		return
	}
	response.OK(c, row)
}
