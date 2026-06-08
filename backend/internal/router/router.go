package router

import (
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// App 包含所有路由依赖的上下文。
type App struct {
	DB  *gorm.DB
	Auth *middleware.AuthMiddleware
}

// Setup 注册所有 Gin 路由（薄抽象，集中所有路由）。
// 未来按 handler 完善后，在此逐一挂载。
func Setup(app *App) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 全局中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok"})
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
		// 无需鉴权
		_ = v1.Group("") // 后续添加公开路由

		// 需鉴权
		auth := v1.Group("")
		auth.Use(app.Auth.Handler())
		{
			_ = auth // 后续添加需鉴权路由
		}
	}

	// Admin API（独立路由前缀，独立鉴权）
	_ = r.Group("/admin/api/v1") // 后续添加后台路由

	return r
}
