package handler

import (
	"errors"
	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/service"
	"github.com/gin-gonic/gin"
	"strconv"
)

type FreeChartHandler struct{ svc *service.FreeChartService }

func NewFreeChartHandler(svc *service.FreeChartService) *FreeChartHandler {
	return &FreeChartHandler{svc: svc}
}
func (h *FreeChartHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}
	var in service.FreeChartInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.Create(c.Request.Context(), userID, in)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("free chart request failed", "err", err, "user_id", userID)
		if errors.Is(err, birthchart.ErrLocationConflict) {
			response.Fail(c, response.CodeBadRequest, "location_conflict")
			return
		}
		if errors.Is(err, birthchart.ErrLocationSearchUnavailable) {
			response.Fail(c, response.CodeBadRequest, "location_validation_unavailable")
			return
		}
		response.Fail(c, response.CodeBadRequest, "free_chart_failed")
		return
	}
	response.OK(c, result)
}

func (h *FreeChartHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
	result, err := h.svc.List(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Error(c, "free_chart_list_failed")
		return
	}
	response.OK(c, result)
}
func (h *FreeChartHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid_free_chart_id")
		return
	}
	result, err := h.svc.Get(c.Request.Context(), userID, id)
	if errors.Is(err, service.ErrFreeChartNotFound) {
		response.Fail(c, response.CodeNotFound, "free_chart_not_found")
		return
	}
	if err != nil {
		response.Error(c, "free_chart_get_failed")
		return
	}
	response.OK(c, result)
}
func (h *FreeChartHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid_free_chart_id")
		return
	}
	err = h.svc.Delete(c.Request.Context(), userID, id)
	if errors.Is(err, service.ErrFreeChartNotFound) {
		response.Fail(c, response.CodeNotFound, "free_chart_not_found")
		return
	}
	if err != nil {
		response.Error(c, "free_chart_delete_failed")
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
func (h *FreeChartHandler) BatchDelete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var in struct {
		IDs []uint64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid_batch_delete")
		return
	}
	count, err := h.svc.BatchDelete(c.Request.Context(), userID, in.IDs)
	if errors.Is(err, service.ErrInvalidBatchDelete) {
		response.Fail(c, response.CodeBadRequest, "invalid_batch_delete")
		return
	}
	if err != nil {
		response.Error(c, "free_chart_batch_delete_failed")
		return
	}
	response.OK(c, gin.H{"deleted_count": count})
}
