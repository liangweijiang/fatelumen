package handler

import (
	"errors"
	"strconv"

	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChartHandler struct {
	svc *service.ChartService
}

func NewChartHandler(svc *service.ChartService) *ChartHandler {
	return &ChartHandler{svc: svc}
}

func (h *ChartHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}
	var in service.CreateChartInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid request body")
		return
	}
	chart, err := h.svc.Calculate(c.Request.Context(), userID, in)
	if err != nil {
		if err.Error() == "profile not found" {
			response.Fail(c, response.CodeNotFound, err.Error())
			return
		}
		response.Error(c, err.Error())
		return
	}
	response.OK(c, chart)
}

func (h *ChartHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid chart id")
		return
	}
	chart, err := h.svc.GetByID(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, response.CodeNotFound, "chart not found")
			return
		}
		if err.Error() == "chart not found" {
			response.Fail(c, response.CodeNotFound, err.Error())
			return
		}
		response.Error(c, err.Error())
		return
	}
	response.OK(c, chart)
}
