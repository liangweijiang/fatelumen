package middleware

import (
	"net/http"
	"strings"

	"fatelumen/backend/internal/pkg/jwt"
	"fatelumen/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthMiddleware JWT 校验 + 单设备登录检查。
// 从 Authorization: Bearer <token> 提取 JWT，校验后把 user_id 写入 context。
type AuthMiddleware struct {
	jwtSecret string
	db        *gorm.DB
}

func NewAuthMiddleware(jwtSecret string, db *gorm.DB) *AuthMiddleware {
	return &AuthMiddleware{jwtSecret: jwtSecret, db: db}
}

// Handler 返回 Gin 中间件函数。
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

		// 单设备登录：比对 current_token_id（若启用）
		// MVP 阶段此逻辑在 auth_service 中实现，中间件只做基本校验。
		_ = claims.TokenID

		// 设置用户 ID 到 context
		c.Set("user_id", claims.UserID)
		c.Set("token_id", claims.TokenID)
		c.Next()
	}
}

// OptionalAuth 可选鉴权（有 token 则解析，无则不拦截）。
func (m *AuthMiddleware) OptionalHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.Next()
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims, err := jwt.Parse(m.jwtSecret, tokenStr)
		if err == nil {
			c.Set("user_id", claims.UserID)
			c.Set("token_id", claims.TokenID)
		}
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

// JSONAbort 返回 JSON 并 abort（兼容非标准响应）。
func JSONAbort(c *gin.Context, code int, data interface{}) {
	c.AbortWithStatusJSON(http.StatusOK, data)
}
