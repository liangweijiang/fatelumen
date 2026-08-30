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
	DB                      *gorm.DB
	Auth                    *middleware.AuthMiddleware
	AdminAuth               *middleware.AdminAuthMiddleware
	HealthChecker           HealthChecker
	StaticDir               string
	AuthHandler             *handler.AuthHandler
	AdminAuthHandler        *handler.AdminAuthHandler
	ContentHandler          *handler.ContentHandler
	PricingHandler          *handler.PricingHandler
	ReportInputHandler      *handler.ReportInputHandler
	ProfHandler             *handler.ProfileHandler
	ChartHandler            *handler.ChartHandler
	FreeChartHandler        *handler.FreeChartHandler
	LocationHandler         *handler.LocationHandler
	GeoHandler              *handler.GeoHandler
	AdminGeoHandler         *handler.AdminGeoHandler
	AdminBaziBaseHandler    *handler.AdminBaziBaseHandler
	AdminReportTraceHandler *handler.AdminReportTraceHandler
	AdminCalculationHandler *handler.AdminCalculationHandler
	ReadingHandler          *handler.ReadingHandler
	ReportHandler           *handler.ReportHandler
	OrderHandler            *handler.OrderHandler
	WebhookHandler          *handler.WebhookHandler
	DevPayHandler           *handler.DevPayHandler
	AdminHandler            *handler.AdminHandler
	ResourceHandler         *handler.ResourceHandler

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
		adminAuth := v1.Group("/admin/auth")
		adminAuth.GET("/captcha", app.RateLimitAuth, app.AdminAuthHandler.Captcha)
		adminAuth.POST("/login", app.RateLimitAuth, app.AdminAuthHandler.Login)
		v1.GET("/public/content/:type", app.ContentHandler.PublicList)
		v1.GET("/public/content/:type/:slug", app.ContentHandler.PublicDetail)
		v1.GET("/public/knowledge", func(c *gin.Context) {
			c.Params = append(c.Params, gin.Param{Key: "type", Value: "knowledge"})
			app.ContentHandler.PublicList(c)
		})
		v1.GET("/public/faqs", func(c *gin.Context) {
			c.Params = append(c.Params, gin.Param{Key: "type", Value: "faq"})
			app.ContentHandler.PublicList(c)
		})
		v1.GET("/public/cases", func(c *gin.Context) {
			c.Params = append(c.Params, gin.Param{Key: "type", Value: "case"})
			app.ContentHandler.PublicList(c)
		})
		v1.GET("/public/pricing", app.PricingHandler.PublicList)
		// --- 认证（无需鉴权）---
		authGroup := v1.Group("/auth")
		{
			authLimited := authGroup.Group("")
			authLimited.Use(app.RateLimitAuth)
			// 同时支持 302 跳转和 JSON 返回
			authLimited.GET("/google/login", func(c *gin.Context) {
				if c.Query("format") == "json" || c.GetHeader("Accept") == "application/json" {
					app.AuthHandler.GoogleLoginJSON(c)
					return
				}
				app.AuthHandler.GoogleLogin(c)
			})
			authLimited.GET("/google/callback", app.AuthHandler.GoogleCallback)
			// The code is cryptographically random, one-time and valid for two
			// minutes. It is intentionally outside the generic auth limiter so a
			// successful provider callback can always complete its own session.
			authGroup.POST("/exchange", app.AuthHandler.ExchangeGoogleLogin)
			authLimited.GET("/providers", app.AuthHandler.ProvidersList)
			authLimited.POST("/register", app.AuthHandler.Register)
			authLimited.POST("/login", app.AuthHandler.Login)
		}

		// Webhook 路由（无需鉴权，依靠渠道签名字段验证身份）
		v1.POST("/webhooks/stripe", app.WebhookHandler.Stripe)
		v1.POST("/webhooks/alipay", app.WebhookHandler.Alipay)
		v1.POST("/webhooks/paypal", app.WebhookHandler.Paypal)

		// Dev-only 本地收银台（仅当 main 注入 DevPayHandler 时注册）
		if app.DevPayHandler != nil {
			v1.GET("/dev/pay/:id", app.DevPayHandler.Page)
			v1.POST("/dev/pay/:id/complete", app.DevPayHandler.Complete)
		}

		// --- 需鉴权 ---
		authed := v1.Group("")
		authed.Use(app.Auth.Handler())
		{
			authed.POST("/auth/logout", app.AuthHandler.Logout)
			if app.GeoHandler != nil {
				authed.GET("/geo/countries", app.GeoHandler.Countries)
				authed.GET("/geo/cities", app.GeoHandler.Cities)
				authed.GET("/geo/cities/:id", app.GeoHandler.City)
			}
			authed.GET("/me", app.AuthHandler.GetMe)
			authed.PATCH("/me", app.AuthHandler.UpdateMe)
			authed.GET("/locations/search", app.RateLimitReading, app.LocationHandler.Search)
			authed.POST("/locations/resolve", app.RateLimitReading, app.LocationHandler.Resolve)
			freeCharts := authed.Group("/free-charts")
			{
				freeCharts.POST("", app.RateLimitReading, app.FreeChartHandler.Create)
				freeCharts.GET("", app.FreeChartHandler.List)
				freeCharts.POST("/batch-delete", app.FreeChartHandler.BatchDelete)
				freeCharts.GET("/:id", app.FreeChartHandler.Get)
				freeCharts.DELETE("/:id", app.FreeChartHandler.Delete)
			}

			profiles := authed.Group("/profiles")
			{
				profiles.POST("", app.ProfHandler.Create)
				profiles.GET("", app.ProfHandler.List)
				profiles.GET("/:id", app.ProfHandler.Get)
				profiles.PATCH("/:id", app.ProfHandler.Update)
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
				reports.POST("/from-input", app.ReportInputHandler.Create)
				reports.POST("", app.RateLimitReading, app.ReportHandler.Create)
				reports.GET("/:id", app.ReportHandler.Get)
				reports.GET("", app.ReportHandler.List)
				reports.GET("/:id/html", app.ReportHandler.ViewHTML)
				reports.POST("/:id/pdf", app.ReportHandler.ExportPDF)
				reports.POST("/:id/unlock", app.ReportHandler.UnlockWithCredits)
			}

			orders := authed.Group("/orders")
			{
				orders.POST("", app.RateLimitOrder, app.OrderHandler.Create)
				orders.GET("/:id", app.OrderHandler.Get)
				orders.GET("", app.OrderHandler.List)
			}

		}

		// Admin routes use their own JWT and never inherit C-end authentication.
		admin := v1.Group("/admin")
		admin.Use(app.AdminAuth.Handler())
		{
			admin.POST("/auth/logout", app.AdminAuthHandler.Logout)
			admin.GET("/auth/me", app.AdminAuthHandler.Me)
			admin.GET("/ping", app.AdminHandler.Ping)
			admin.GET("/stats", app.AdminHandler.Stats)
			admin.GET("/users", app.AdminHandler.ListUsers)
			admin.GET("/users/:id", app.AdminHandler.GetUser)
			admin.GET("/content/:type", app.ContentHandler.List)
			admin.POST("/content", app.ContentHandler.Create)
			admin.POST("/content/import-markdown", app.ContentHandler.ImportMarkdown)
			admin.GET("/content/:type/:id", app.ContentHandler.Detail)
			admin.PATCH("/content/:type/:id", app.ContentHandler.Update)
			admin.POST("/content/:type/:id/state", app.ContentHandler.State)
			admin.DELETE("/content/:type/:id", app.ContentHandler.Delete)
			admin.POST("/content/:type/batch-pin", app.ContentHandler.BatchPin)
			admin.POST("/content/:type/batch-delete", app.ContentHandler.BatchDelete)
			admin.GET("/pricing", app.PricingHandler.List)
			admin.POST("/pricing", app.PricingHandler.Create)
			admin.GET("/pricing/:id", app.PricingHandler.Detail)
			admin.PATCH("/pricing/:id", app.PricingHandler.Update)
			admin.DELETE("/pricing/:id", app.PricingHandler.Delete)
			if app.AdminGeoHandler != nil {
				admin.GET("/geo", app.AdminGeoHandler.List)
				admin.PATCH("/geo/:id", app.AdminGeoHandler.Update)
			}
			if app.GeoHandler != nil {
				admin.GET("/geo/countries", app.GeoHandler.Countries)
				admin.GET("/geo/cities", app.GeoHandler.Cities)
			}
			if app.AdminBaziBaseHandler != nil {
				admin.GET("/bazi-base/summary", app.AdminBaziBaseHandler.Summary)
				admin.GET("/bazi-base/elements", app.AdminBaziBaseHandler.Elements)
				admin.GET("/bazi-base/stems", app.AdminBaziBaseHandler.Stems)
				admin.GET("/bazi-base/branches", app.AdminBaziBaseHandler.Branches)
				admin.GET("/bazi-base/ten-gods", app.AdminBaziBaseHandler.TenGods)
				admin.GET("/bazi-base/relations", app.AdminBaziBaseHandler.Relations)
				admin.GET("/bazi-base/strength-rules", app.AdminBaziBaseHandler.StrengthRules)
				admin.GET("/bazi-base/annual-calendar", app.AdminBaziBaseHandler.AnnualCalendar)
			}
			if app.AdminReportTraceHandler != nil {
				admin.GET("/prompt-registry", app.AdminReportTraceHandler.PromptRegistry)
				admin.GET("/reports/:id/facts", app.AdminReportTraceHandler.Facts)
				admin.POST("/reports/:id/prompt-preview", app.AdminReportTraceHandler.PromptPreview)
				admin.GET("/reports/:id/llm-calls", app.AdminReportTraceHandler.LLMCalls)
				admin.GET("/reports/:id/llm-calls/:callId", app.AdminReportTraceHandler.LLMCall)
			}
			if app.AdminCalculationHandler != nil {
				admin.GET("/calculation-archives", app.AdminCalculationHandler.List)
				admin.POST("/calculation-archives", app.AdminCalculationHandler.Create)
				admin.POST("/calculation-archives/batch-delete", app.AdminCalculationHandler.BatchDelete)
				admin.GET("/calculation-archives/:id", app.AdminCalculationHandler.Get)
				admin.PATCH("/calculation-archives/:id", app.AdminCalculationHandler.Update)
				admin.POST("/calculation-archives/:id/recalculate", app.AdminCalculationHandler.Recalculate)
				admin.DELETE("/calculation-archives/:id", app.AdminCalculationHandler.Delete)
			}
			if app.ResourceHandler != nil {
				admin.GET("/resources/:resource/_schema", app.ResourceHandler.Schema)
				admin.GET("/resources/:resource", app.ResourceHandler.List)
				admin.GET("/resources/:resource/:id", app.ResourceHandler.Detail)
				admin.POST("/resources/:resource/:id/actions/:action", app.ResourceHandler.Action)
			}
		}
	}

	_ = r.Group("/admin/api/v1")

	return r
}
