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

// AuthMiddleware JWT 校验 + 单设备登录检查。
type AuthMiddleware struct {
	jwtSecret string
	db        *gorm.DB
}

func NewAuthMiddleware(jwtSecret string, db *gorm.DB) *AuthMiddleware {
	return &AuthMiddleware{jwtSecret: jwtSecret, db: db}
}

// Handler 返回 Gin 中间件函数：提取 JWT → 校验签名 → 单设备登录检查 → 写入 context。
func (m *AuthMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			response.Fail(c, response.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		claims, err := jwt.Parse(m.jwtSecret, tokenStr)
		if err != nil {
			response.Fail(c, response.CodeUnauthorized, "token invalid or expired")
			c.Abort()
			return
		}

		// 单设备登录：比对 DB 中的 current_token_id
		var user model.User
		if err := m.db.Select("current_token_id").First(&user, claims.UserID).Error; err != nil {
			response.Fail(c, response.CodeUnauthorized, "user not found")
			c.Abort()
			return
		}
		if user.CurrentTokenID != "" && user.CurrentTokenID != claims.TokenID {
			response.Fail(c, response.CodeKicked, "logged in from another device")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("token_id", claims.TokenID)
		c.Next()
	}
}

// OptionalHandler 可选鉴权（有 token 则解析，无则不拦截）。
func (m *AuthMiddleware) OptionalHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.Next()
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims, err := jwt.Parse(m.jwtSecret, tokenStr)
		if err != nil {
			c.Next()
			return
		}

		// 单设备检查（可选鉴权时同样校验）
		var user model.User
		if err := m.db.Select("current_token_id").First(&user, claims.UserID).Error; err != nil {
			c.Next()
			return
		}
		if user.CurrentTokenID != "" && user.CurrentTokenID != claims.TokenID {
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("token_id", claims.TokenID)
		c.Next()
	}
}

// GetUserID 从 gin.Context 中获取当前用户 ID。
func GetUserID(c *gin.Context) uint64 {
	v, _ := c.Get("user_id")
	if v == nil {
		return 0
	}
	return v.(uint64)
}

// JSONAbort 返回 JSON 并 abort。
func JSONAbort(c *gin.Context, code int, data interface{}) {
	c.AbortWithStatusJSON(http.StatusOK, data)
}
