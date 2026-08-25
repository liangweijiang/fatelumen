package handler

import (
	"errors"
	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"
	"github.com/gin-gonic/gin"
	"strings"
)

type LocationHandler struct{ resolver birthchart.LocationResolver }

func NewLocationHandler(resolver birthchart.LocationResolver) *LocationHandler {
	return &LocationHandler{resolver: resolver}
}
func (h *LocationHandler) Search(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len([]rune(q)) < 2 {
		response.Fail(c, response.CodeBadRequest, "location keyword is too short")
		return
	}
	items, err := h.resolver.Search(c.Request.Context(), q, c.DefaultQuery("locale", "en"))
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("location search failed", "err", err)
		if errors.Is(err, birthchart.ErrLocationSearchUnavailable) {
			response.Fail(c, response.CodeServerError, "location search is not configured")
			return
		}
		response.Error(c, "location search failed")
		return
	}
	response.OK(c, items)
}
func (h *LocationHandler) Resolve(c *gin.Context) {
	var in birthchart.LocationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid request body")
		return
	}
	item, err := h.resolver.Resolve(c.Request.Context(), in)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Warn("location resolve failed", "err", err)
		if errors.Is(err, birthchart.ErrLocationConflict) {
			response.Fail(c, response.CodeBadRequest, "location_conflict")
			return
		}
		if errors.Is(err, birthchart.ErrLocationSearchUnavailable) {
			response.Fail(c, response.CodeBadRequest, "location_validation_unavailable")
			return
		}
		response.Fail(c, response.CodeBadRequest, err.Error())
		return
	}
	response.OK(c, item)
}
