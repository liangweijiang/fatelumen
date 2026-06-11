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

// ReadingHandler 简单测算 HTTP 处理器。
type ReadingHandler struct {
	svc *service.ReadingService
}

func NewReadingHandler(svc *service.ReadingService) *ReadingHandler {
	return &ReadingHandler{svc: svc}
}

// CreateQuick POST /api/v1/readings/quick
func (h *ReadingHandler) CreateQuick(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}

	var in service.CreateQuickInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid request body")
		return
	}
	if in.ProfileID == 0 {
		response.Fail(c, response.CodeBadRequest, "profile_id is required")
		return
	}
	in.IsAdmin = middleware.IsAdmin(c)

	reading, err := h.svc.CreateQuick(c.Request.Context(), userID, in)
	if err != nil {
		if errors.Is(err, service.ErrQuotaExceeded) {
			response.Fail(c, response.CodeQuotaExhausted, "daily free quota exceeded")
			return
		}
		response.Error(c, err.Error())
		return
	}
	response.OK(c, reading)
}

// GetByID GET /api/v1/readings/:id
func (h *ReadingHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid reading id")
		return
	}

	reading, err := h.svc.GetByID(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, response.CodeNotFound, "reading not found")
			return
		}
		response.Error(c, err.Error())
		return
	}
	response.OK(c, reading)
}

// ListByUser GET /api/v1/readings
func (h *ReadingHandler) ListByUser(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	readings, err := h.svc.ListByUser(c.Request.Context(), userID, limit, offset)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, readings)
}
