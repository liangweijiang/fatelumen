package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"fatelumen/backend/internal/cache"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/jwt"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AdminAuthHandler owns the administrator-only login flow. It deliberately has
// no Google provider or registration endpoint.
type AdminAuthHandler struct {
	db          *gorm.DB
	cache       cache.Cache
	secret      string
	expireHours int
}

func NewAdminAuthHandler(db *gorm.DB, c cache.Cache, secret string, expireHours int) *AdminAuthHandler {
	return &AdminAuthHandler{db: db, cache: c, secret: secret, expireHours: expireHours}
}

func randomHex(n int) string { b := make([]byte, n); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func captchaText() string {
	const chars = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	var b strings.Builder
	for i := 0; i < 4; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b.WriteByte(chars[n.Int64()])
	}
	return b.String()
}

func captchaSVG(text string) string {
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="144" height="48" viewBox="0 0 144 48"><rect width="144" height="48" rx="7" fill="#f4e6c5"/><path d="M5 11L137 37M8 39L134 9" stroke="#c8a85f" opacity=".55"/><text x="20" y="33" font-family="serif" font-size="25" font-style="italic" letter-spacing="8" fill="#4f3a17">%s</text></svg>`, text)
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg))
}

// Captcha returns a short-lived challenge. The frontend renders its text in a
// canvas; the answer remains only in cache and is consumed on login.
func (h *AdminAuthHandler) Captcha(c *gin.Context) {
	id, text := randomHex(16), captchaText()
	if err := h.cache.Set(c.Request.Context(), "admin:captcha:"+id, text, 3*time.Minute); err != nil {
		logger.FromCtx(c.Request.Context()).Error("admin captcha cache failed", "err", err)
		response.Error(c, "service unavailable")
		return
	}
	response.OK(c, gin.H{"captcha_id": id, "image": captchaSVG(text), "expires_in": 180})
}

func (h *AdminAuthHandler) Login(c *gin.Context) {
	var in struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		CaptchaID     string `json:"captcha_id"`
		CaptchaAnswer string `json:"captcha_answer"`
	}
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, response.CodeBadRequest, "invalid request")
		return
	}
	answer, err := h.cache.Take(c.Request.Context(), "admin:captcha:"+in.CaptchaID)
	if err != nil || answer == "" || !strings.EqualFold(answer, strings.TrimSpace(in.CaptchaAnswer)) {
		response.JSON(c, http.StatusUnauthorized, response.Resp{Code: response.CodeUnauthorized, Msg: "invalid credentials or captcha"})
		return
	}
	var admin model.AdminUser
	if err := h.db.Where("username = ?", strings.TrimSpace(in.Username)).First(&admin).Error; err != nil || admin.Status != "active" || bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(in.Password)) != nil {
		response.JSON(c, http.StatusUnauthorized, response.Resp{Code: response.CodeUnauthorized, Msg: "invalid credentials or captcha"})
		return
	}
	tokenID := randomHex(16)
	now := time.Now()
	if err := h.db.Model(&admin).Updates(map[string]interface{}{"current_token_id": tokenID, "last_login_at": now}).Error; err != nil {
		logger.FromCtx(c.Request.Context()).Error("admin login update failed", "err", err, "admin_id", admin.ID)
		response.Error(c, "login failed")
		return
	}
	token, err := jwt.GenerateAdmin(h.secret, h.expireHours, admin.ID, admin.Username, admin.RoleID, tokenID)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("admin token generation failed", "err", err, "admin_id", admin.ID)
		response.Error(c, "login failed")
		return
	}
	response.OK(c, gin.H{"token": token, "expires_in": h.expireHours * 3600, "admin": gin.H{"id": admin.ID, "username": admin.Username, "display_name": admin.DisplayName}})
}
func (h *AdminAuthHandler) Logout(c *gin.Context) {
	id, ok := c.Get("admin_id")
	if !ok {
		response.Fail(c, response.CodeUnauthorized, "admin authentication required")
		return
	}
	if err := h.db.Model(&model.AdminUser{}).Where("id = ?", id).Update("current_token_id", "").Error; err != nil {
		logger.FromCtx(c.Request.Context()).Error("admin logout failed", "err", err)
		response.Error(c, "logout failed")
		return
	}
	response.OK(c, gin.H{"logged_out": true})
}
func (h *AdminAuthHandler) Me(c *gin.Context) {
	id := c.GetUint64("admin_id")
	var admin model.AdminUser
	if err := h.db.Select("id,username,display_name,status,last_login_at").First(&admin, id).Error; err != nil {
		c.Status(http.StatusUnauthorized)
		return
	}
	response.OK(c, admin)
}
