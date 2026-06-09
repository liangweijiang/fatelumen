package router

import (
	"fatelumen/backend/internal/handler"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// App 包含所有路由依赖的上下文。
type App struct {
	DB           *gorm.DB
	Auth         *middleware.AuthMiddleware
	AuthHandler  *handler.AuthHandler
	ProfHandler  *handler.ProfileHandler
	ChartHandler *handler.ChartHandler
}

// Setup 注册所有 Gin 路由。
func Setup(app *App) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	{
		// --- 认证（无需鉴权）---
		authGroup := v1.Group("/auth")
		{
			// 同时支持 302 跳转和 JSON 返回
			authGroup.GET("/google/login", func(c *gin.Context) {
				if c.Query("format") == "json" || c.GetHeader("Accept") == "application/json" {
					app.AuthHandler.GoogleLoginJSON(c)
					return
				}
				app.AuthHandler.GoogleLogin(c)
			})
			authGroup.GET("/google/callback", app.AuthHandler.GoogleCallback)
			authGroup.GET("/providers", app.AuthHandler.ProvidersList)
		}

		// --- 需鉴权 ---
		authed := v1.Group("")
		authed.Use(app.Auth.Handler())
		{
			authed.POST("/auth/logout", app.AuthHandler.Logout)
			authed.GET("/me", app.AuthHandler.GetMe)
			authed.PATCH("/me", app.AuthHandler.UpdateMe)

			profiles := authed.Group("/profiles")
			{
				profiles.POST("", app.ProfHandler.Create)
				profiles.GET("", app.ProfHandler.List)
				profiles.GET("/:id", app.ProfHandler.Get)
				profiles.DELETE("/:id", app.ProfHandler.Delete)
			}

			charts := authed.Group("/charts")
			{
				charts.POST("", app.ChartHandler.Create)
				charts.GET("/:id", app.ChartHandler.Get)
			}
		}
	}

	_ = r.Group("/admin/api/v1")

	return r
}
