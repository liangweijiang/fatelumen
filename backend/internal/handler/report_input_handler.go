package handler

import (
	"errors"

	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// ReportInputHandler accepts report data independently from a personal profile.
// save_action is explicit: existing reuses an unchanged owned profile, none
// keeps a hidden report subject, new creates a saved profile, and update
// requires an owned target_profile_id.
type ReportInputHandler struct {
	profiles *service.ProfileService
	reports  *service.ReportService
}

func NewReportInputHandler(p *service.ProfileService, r *service.ReportService) *ReportInputHandler {
	return &ReportInputHandler{profiles: p, reports: r}
}

func (h *ReportInputHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}
	var in struct {
		Profile         service.CreateProfileInput `json:"profile"`
		SaveAction      string                     `json:"save_action"`
		TargetProfileID uint64                     `json:"target_profile_id"`
		Locale          string                     `json:"locale"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid request body")
		return
	}
	var (
		profileID uint64
		err       error
		saved     bool
	)
	switch in.SaveAction {
	case "existing":
		if in.TargetProfileID == 0 {
			response.Fail(c, response.CodeBadRequest, "target_profile_id is required for existing profile")
			return
		}
		profile, getErr := h.profiles.Get(c.Request.Context(), userID, in.TargetProfileID)
		if getErr != nil || profile == nil || !profile.Saved {
			response.Fail(c, response.CodeNotFound, "target profile not found")
			return
		}
		profileID = profile.ID
		saved = true
	case "none":
		profile, createErr := h.profiles.CreateReportSubject(c.Request.Context(), userID, in.Profile)
		if createErr != nil {
			err = createErr
		} else {
			profileID = profile.ID
		}
	case "new":
		profile, createErr := h.profiles.Create(c.Request.Context(), userID, in.Profile)
		if createErr != nil {
			err = createErr
		} else {
			profileID = profile.ID
			saved = true
		}
	case "update":
		if in.TargetProfileID == 0 {
			response.Fail(c, response.CodeBadRequest, "target_profile_id is required for update")
			return
		}
		profile, updateErr := h.profiles.Update(c.Request.Context(), userID, in.TargetProfileID, in.Profile)
		if updateErr != nil {
			err = updateErr
		} else {
			profileID = profile.ID
			saved = true
		}
	default:
		response.Fail(c, response.CodeBadRequest, "save_action must be existing, none, new, or update")
		return
	}
	if err != nil {
		if errors.Is(err, service.ErrProfileNotFound) {
			response.Fail(c, response.CodeNotFound, "target profile not found")
			return
		}
		response.Fail(c, response.CodeBadRequest, err.Error())
		return
	}
	report, err := h.reports.CreateReport(c.Request.Context(), userID, profileID, in.Locale)
	if err != nil {
		response.Error(c, "report creation failed")
		return
	}
	response.OK(c, gin.H{
		"report_id": report.ID, "status": report.Status, "profile_id": profileID,
		"profile_saved": saved, "save_action": in.SaveAction,
	})
}
