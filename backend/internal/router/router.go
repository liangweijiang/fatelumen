package router

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"fatelumen/backend/internal/handler"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthChecker 健康检查抽象，便于单测注入 fake。
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// NewDBHealthChecker creates a health checker backed by a gorm DB.
func NewDBHealthChecker(db *gorm.DB) *DBHealthChecker {
	return &DBHealthChecker{db: db}
}

// DBHealthChecker 基于 *gorm.DB 的探活实现。
type DBHealthChecker struct {
	db *gorm.DB
}

func (h *DBHealthChecker) Ping(ctx context.Context) error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return fmt.Errorf("db conn unavailable: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

// App 包含所有路由依赖的上下文。
type App struct {
	DB             *gorm.DB
	Auth           *middleware.AuthMiddleware
	HealthChecker  HealthChecker
	StaticDir      string
	AuthHandler    *handler.AuthHandler
	ProfHandler    *handler.ProfileHandler
	ChartHandler   *handler.ChartHandler
	ReadingHandler *handler.ReadingHandler
	ReportHandler  *handler.ReportHandler
	OrderHandler   *handler.OrderHandler
	WebhookHandler *handler.WebhookHandler
	AdminHandler   *handler.AdminHandler

	// Pre-built rate-limit middleware (set by main)
	RateLimitAuth    gin.HandlerFunc
	RateLimitReading gin.HandlerFunc
	RateLimitOrder   gin.HandlerFunc
}

// Setup 注册所有 Gin 路由。
func Setup(app *App) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(middleware.Recovery())
	r.Use(middleware.Trace())
	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		if app.HealthChecker == nil {
			response.OK(c, gin.H{"status": "ok"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := app.HealthChecker.Ping(ctx); err != nil {
			logger.FromCtx(c.Request.Context()).Error("health check failed",
				"err", err,
			)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"detail": "database unreachable",
			})
			return
		}
		response.OK(c, gin.H{"status": "ok"})
	})

	if app.StaticDir != "" {
		r.Static("/static", app.StaticDir)
	}

	v1 := r.Group("/api/v1")
	{
		// --- 认证（无需鉴权）---
		authGroup := v1.Group("/auth")
		authGroup.Use(app.RateLimitAuth)
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
			authGroup.POST("/register", app.AuthHandler.Register)
			authGroup.POST("/login", app.AuthHandler.Login)
		}

		// Webhook 路由（无需鉴权，依靠渠道签名字段验证身份）
		v1.POST("/webhooks/stripe", app.WebhookHandler.Stripe)

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

			readings := authed.Group("/readings")
			{
				readings.POST("/quick", app.RateLimitReading, app.ReadingHandler.CreateQuick)
				readings.GET("/:id", app.ReadingHandler.GetByID)
				readings.GET("", app.ReadingHandler.ListByUser)
			}

			reports := authed.Group("/reports")
			{
				reports.POST("", app.RateLimitReading, app.ReportHandler.Create)
				reports.GET("/:id", app.ReportHandler.Get)
				reports.GET("", app.ReportHandler.List)
				reports.GET("/:id/html", app.ReportHandler.ViewHTML)
				reports.POST("/:id/pdf", app.ReportHandler.ExportPDF)
			}

			orders := authed.Group("/orders")
			{
				orders.POST("", app.RateLimitOrder, app.OrderHandler.Create)
				orders.GET("/:id", app.OrderHandler.Get)
				orders.GET("", app.OrderHandler.List)
			}

			// Admin routes: JWT auth + AdminOnly guard
			admin := authed.Group("/admin")
			admin.Use(middleware.AdminOnly(app.DB))
			{
				admin.GET("/ping", app.AdminHandler.Ping)
				admin.GET("/stats", app.AdminHandler.Stats)
				admin.GET("/users", app.AdminHandler.ListUsers)
				admin.GET("/users/:id", app.AdminHandler.GetUser)
				admin.PATCH("/users/:id/active", app.AdminHandler.SetActive)
				admin.PATCH("/users/:id/unlimited", app.AdminHandler.SetUnlimited)
				admin.GET("/orders", app.AdminHandler.ListOrders)
				admin.GET("/orders/:id", app.AdminHandler.GetOrder)
				admin.GET("/reports", app.AdminHandler.ListReports)
				admin.GET("/reports/:id", app.AdminHandler.GetReport)
				admin.POST("/reports/:id/unlock", app.AdminHandler.UnlockReport)
			}
		}
	}

	_ = r.Group("/admin/api/v1")

	return r
}
