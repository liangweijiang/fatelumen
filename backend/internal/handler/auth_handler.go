package handler

import (
	"net/http"

	"fatelumen/backend/internal/auth"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证相关 HTTP 处理器。
type AuthHandler struct {
	svc     *service.AuthService
	authReg *auth.Registry
}

func NewAuthHandler(svc *service.AuthService, authReg *auth.Registry) *AuthHandler {
	return &AuthHandler{svc: svc, authReg: authReg}
}

// GoogleLogin GET /api/v1/auth/google/login
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	url, err := h.svc.GetLoginURL(c.Request.Context(), "google")
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	c.Redirect(http.StatusFound, url)
}

// GoogleLoginJSON GET /api/v1/auth/google/login?format=json
// DECISION: 前端可能通过 fetch 调，返回 JSON URL 而非 302 跳转。
func (h *AuthHandler) GoogleLoginJSON(c *gin.Context) {
	url, err := h.svc.GetLoginURL(c.Request.Context(), "google")
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, gin.H{"auth_url": url})
}

// GoogleCallback GET /api/v1/auth/google/callback
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" {
		response.Fail(c, response.CodeBadRequest, "missing code")
		return
	}

	result, err := h.svc.HandleCallback(c.Request.Context(), "google", code, state)
	if err != nil {
		response.Fail(c, response.CodeUnauthorized, "login failed: "+err.Error())
		return
	}

	response.OK(c, result)
}

// Logout POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}
	if err := h.svc.Logout(c.Request.Context(), userID); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, gin.H{"status": "logged_out"})
}

// GetMe GET /api/v1/me
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}
	user, err := h.svc.GetMe(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "user not found")
		return
	}
	response.OK(c, user)
}

// UpdateMe PATCH /api/v1/me
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}
	var patches map[string]interface{}
	if err := c.ShouldBindJSON(&patches); err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid request body")
		return
	}
	user, err := h.svc.UpdateMe(c.Request.Context(), userID, patches)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, user)
}

// ProvidersList GET /api/v1/auth/providers（可选，返回启用的登录渠道列表）
func (h *AuthHandler) ProvidersList(c *gin.Context) {
	response.OK(c, gin.H{"providers": h.authReg.Enabled()})
}
