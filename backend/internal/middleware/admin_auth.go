package middleware

import (
	"net/http"
	"strings"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/jwt"
	"fatelumen/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminAuthMiddleware authenticates only independently issued administrator JWTs.
type AdminAuthMiddleware struct {
	secret string
	db     *gorm.DB
}

func NewAdminAuthMiddleware(secret string, db *gorm.DB) *AdminAuthMiddleware {
	return &AdminAuthMiddleware{secret: secret, db: db}
}

func (m *AdminAuthMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			response.JSON(c, http.StatusUnauthorized, response.Resp{Code: response.CodeUnauthorized, Msg: "admin authentication required"})
			c.Abort()
			return
		}
		claims, err := jwt.ParseAdmin(m.secret, strings.TrimPrefix(auth, "Bearer "))
		if err != nil {
			response.JSON(c, http.StatusUnauthorized, response.Resp{Code: response.CodeUnauthorized, Msg: "admin token invalid or expired"})
			c.Abort()
			return
		}
		var admin model.AdminUser
		if err := m.db.Select("id, username, status, current_token_id").First(&admin, claims.AdminID).Error; err != nil || admin.Status != "active" || (admin.CurrentTokenID != "" && admin.CurrentTokenID != claims.TokenID) {
			response.JSON(c, http.StatusUnauthorized, response.Resp{Code: response.CodeUnauthorized, Msg: "admin session unavailable"})
			c.Abort()
			return
		}
		c.Set("admin_id", admin.ID)
		c.Set("admin_name", admin.Username)
		c.Next()
	}
}

func GetAdminID(c *gin.Context) uint64 {
	v, _ := c.Get("admin_id")
	if id, ok := v.(uint64); ok {
		return id
	}
	return 0
}
