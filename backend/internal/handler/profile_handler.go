package handler

import (
	"strconv"

	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// ProfileHandler 出生档案 HTTP 处理器。
type ProfileHandler struct {
	svc *service.ProfileService
}

func NewProfileHandler(svc *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

// Create POST /api/v1/profiles
func (h *ProfileHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}
	var in service.CreateProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid request body")
		return
	}
	profile, err := h.svc.Create(c.Request.Context(), userID, in)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, profile)
}

// List GET /api/v1/profiles
func (h *ProfileHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}
	profiles, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	if profiles == nil {
		profiles = make([]model.BirthProfile, 0)
	}
	response.OK(c, profiles)
}

// Get GET /api/v1/profiles/:id
func (h *ProfileHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid profile id")
		return
	}
	profile, err := h.svc.Get(c.Request.Context(), userID, id)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "profile not found")
		return
	}
	if profile == nil {
		response.Fail(c, response.CodeNotFound, "profile not found")
		return
	}
	response.OK(c, profile)
}

// Delete DELETE /api/v1/profiles/:id
func (h *ProfileHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid profile id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), userID, id); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, gin.H{"status": "deleted"})
}
