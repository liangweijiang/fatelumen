package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fatelumen/backend/internal/admin/resource"
	"fatelumen/backend/internal/auth"
	"fatelumen/backend/internal/bazi/basedata"
	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/cache"
	"fatelumen/backend/internal/config"
	"fatelumen/backend/internal/handler"
	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/notify"
	"fatelumen/backend/internal/payment"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/ratelimit"
	"fatelumen/backend/internal/renderer"
	"fatelumen/backend/internal/repository"
	"fatelumen/backend/internal/router"
	"fatelumen/backend/internal/service"
	"fatelumen/backend/internal/storage"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"

	"github.com/gin-gonic/gin"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

const (
	shutdownHTTPTimeout = 30 * time.Second
	shutdownJobTimeout  = 30 * time.Second
)

type parameterizedGORMLogger struct{ gormLogger.Interface }

func (l parameterizedGORMLogger) ParamsFilter(_ context.Context, sql string, _ ...interface{}) (string, []interface{}) {
	return sql, nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	log := logger.Init(cfg.LogLevel)

	// Fail-fast on missing critical config
	if missing := cfg.Validate(); len(missing) > 0 {
		log.Fatal("missing required configuration", "keys", missing)
	}

	dbLogLevel := gormLogger.Warn
	if cfg.AppEnv == "development" || cfg.AppEnv == "dev" {
		dbLogLevel = gormLogger.Info
	}
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: parameterizedGORMLogger{Interface: gormLogger.Default.LogMode(dbLogLevel)},
	})
	if err != nil {
		log.Fatal("failed to connect database", "err", err)
	}
	annualCalendarRepo := repository.NewAnnualCalendarRepo(db)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	if cfg.AppEnv == "development" || cfg.AppEnv == "dev" {
		if err := autoMigrate(db); err != nil {
			log.Fatal("failed to auto migrate", "err", err)
		}
		log.Info("auto migrate completed")
		if err := repository.SeedDevelopmentPricingPlans(db); err != nil {
			log.Fatal("failed to seed development pricing plans", "err", err)
		}
		log.Info("development pricing plans ready", "plan_count", 2, "locale_count", 4)
		if err := annualCalendarRepo.Seed(context.Background(), time.Now().Year(), 60); err != nil {
			log.Fatal("failed to seed annual calendar", "err", err)
		}
		log.Info("annual calendar ready", "start_year", time.Now().Year(), "year_count", 60)
	}

	// 依赖注入
	authMW := middleware.NewAuthMiddleware(cfg.JWTSecret, db)
	adminAuthMW := middleware.NewAdminAuthMiddleware(cfg.AdminJWTSecret, db)

	userRepo := repository.NewUserRepo(db)
	profileRepo := repository.NewProfileRepo(db)
	chartRepo := repository.NewChartRepo(db)
	freeChartRepo := repository.NewFreeChartRepo(db)
	geoRepo := repository.NewGeoRepo(db)
	readingRepo := repository.NewReadingRepo(db)
	fullReportRepo := repository.NewFullReportRepo(db)
	orderRepo := repository.NewOrderRepo(db)

	authReg := auth.NewRegistry()
	if contains(cfg.AuthProviders, "google") {
		authReg.Register(auth.NewGoogleProvider(
			cfg.GoogleClientID,
			cfg.GoogleClientSecret,
			cfg.GoogleRedirectURL,
			log,
		))
	}

	var c cache.Cache
	if cfg.RedisAddr != "" {
		redisClient := redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
		})
		c = cache.NewRedisCache(redisClient)
		log.Info("cache initialized", "type", "redis")
	} else {
		c = cache.NewMemoryCache()
		log.Info("cache initialized", "type", "memory")
	}

	authSvc := service.NewAuthService(userRepo, authReg, cfg.JWTSecret, cfg.JWTExpireHours, cfg.AdminEmails, c, log)
	profileSvc := service.NewProfileService(profileRepo)
	var locationResolver birthchart.LocationResolver = repository.NewGeoLocationResolver(geoRepo)
	profileSvc.SetLocationResolver(locationResolver)
	log.Info("location resolver initialized", "type", "fixed_geonames_database", "version", repository.GeoLocationVersion)
	baseChartEngine := birthchart.NewEngine(birthchart.DefaultInputNormalizer{}, locationResolver, birthchart.IANATimezoneResolver{}, birthchart.DefaultSolarTimeEngine{}, birthchart.LunarGoCalculator{AnnualCalendar: annualCalendarRepo})
	freeChartEngine := birthchart.NewCachedEngine(baseChartEngine, c, 30*24*time.Hour)
	chartSvc := service.NewChartService(chartRepo, profileRepo, baseChartEngine)
	freeChartSvc := service.NewFreeChartService(freeChartEngine, freeChartRepo)
	freeChartSvc.SetLocationValidator(locationResolver)

	var llmProvider llm.LLMProvider
	switch cfg.LLMProvider {
	case "openai":
		llmProvider = llm.NewOpenAIProvider(cfg.OpenAIAPIKey, cfg.OpenAIModel)
	default:
		llmProvider = llm.NewDeepSeekProvider(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL, cfg.DeepSeekModel)
	}
	log.Info("llm provider initialized", "name", llmProvider.Name())

	var imgRenderer renderer.Renderer
	switch cfg.Renderer {
	default:
		imgRenderer = renderer.NewChromedpRenderer(cfg.ChromiumPath)
	}
	log.Info("renderer initialized", "type", cfg.Renderer)

	var fileStorage storage.Storage
	switch {
	case cfg.R2AccountID != "":
		r2, err := storage.NewR2Storage(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket, cfg.R2PublicBase)
		if err != nil {
			log.Fatal("failed to init R2 storage", "err", err)
		}
		fileStorage = r2
		log.Info("storage initialized", "type", "r2")
	case cfg.LocalStorageDir != "":
		publicBase := cfg.AppBaseURL
		if publicBase == "" {
			publicBase = "http://localhost:" + cfg.AppPort
		}
		lfs, err := storage.NewLocalFSStorage(cfg.LocalStorageDir, publicBase)
		if err != nil {
			log.Fatal("failed to init local storage", "err", err)
		}
		fileStorage = lfs
		log.Info("storage initialized", "type", "localfs", "dir", cfg.LocalStorageDir)
	default:
		fileStorage = &storage.NoopStorage{}
		log.Info("storage initialized", "type", "noop")
	}

	quotaSvc := service.NewQuotaService(c, cfg.QuotaDailyLimit)
	readingSvc := service.NewReadingService(readingRepo, profileRepo, chartSvc, quotaSvc, llmProvider, imgRenderer, fileStorage)

	// Job Queue（异步报告任务队列 — Phase 4 状态机核心）
	var jobQueue job.Queue
	switch cfg.JobQueue {
	case "db":
		jobQueue = job.NewDBQueue(db)
		log.Info("job queue initialized", "type", "db")
	default:
		jobQueue = job.NewMemoryQueue()
		log.Info("job queue initialized", "type", "memory")
	}

	// Full report service + immutable ten-chapter executor. The legacy report
	// tables are deliberately not part of this execution path.
	reportSvc := service.NewFullReportService(fullReportRepo, profileRepo, jobQueue, cfg.ReportUnlockCredits, cfg.ReportChapterConcurrency, cfg.ReportRetentionDays)
	fullReportRenderJobRepo := repository.NewFullReportRenderJobRepo(db)
	fullReportCleanupJobRepo := repository.NewFullReportCleanupJobRepo(db)
	reportOutcomeNotifier := service.NewFullReportOutcomeNotifier(fullReportRepo, notify.NewNoopNotifier())
	pdfPipeline := service.NewFullReportPDFPipeline(fullReportRepo, jobQueue, imgRenderer, fileStorage, reportOutcomeNotifier)
	fileDeleter, ok := fileStorage.(storage.Deleter)
	if !ok {
		log.Fatal("storage does not support report cleanup")
	}
	cleanupRunner := service.NewFullReportCleanupRunner(fullReportCleanupJobRepo, fileDeleter)
	llmConfigRepo := repository.NewLLMConfigRepo(db)
	systemSettingRepo := repository.NewSystemSettingRepo(db)
	fullReportRoutes := service.NewDatabaseFullReportRouteResolver(llmConfigRepo, llm.NewConfigSecretCipher(cfg.AdminJWTSecret), time.Duration(cfg.ReportChapterTimeoutSeconds)*time.Second)
	fullReportExecutor := service.NewFullReportExecutor(profileRepo, chartRepo, fullReportRepo, fullReportRoutes, baseChartEngine, service.FullReportRuntimeConfig{
		ChapterConcurrency: cfg.ReportChapterConcurrency,
		ChapterTimeout:     time.Duration(cfg.ReportChapterTimeoutSeconds) * time.Second,
	}, pdfPipeline, reportOutcomeNotifier, systemSettingRepo)

	// Handler registry + worker
	handlerReg := job.NewHandlerRegistry()
	handlerReg.Register(service.FullReportJobType, fullReportExecutor)
	handlerReg.Register(service.FullReportPDFJobType, service.NewFullReportPDFJobHandler(fullReportRenderJobRepo, pdfPipeline, 1, fullReportRepo, reportOutcomeNotifier))
	handlerReg.Register(service.FullReportCleanupJobType, service.NewFullReportCleanupJobHandler(fullReportCleanupJobRepo, cleanupRunner))
	reportWorker := job.NewLaneWorker(jobQueue, handlerReg, 0, 3, time.Duration(cfg.JobStaleThresholdMinutes)*time.Minute, job.LaneDefault, job.LaneReportGeneration)
	pdfWorker := job.NewLaneWorker(jobQueue, handlerReg, 0, 1, 0, job.LanePDFRender)
	cleanupWorker := job.NewLaneWorker(jobQueue, handlerReg, 0, 1, 0, job.LaneReportCleanup)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if cfg.JobQueue != "db" {
		recovered, recoveryErr := service.RecoverInterruptedFullReports(ctx, fullReportRepo, jobQueue)
		if recoveryErr != nil {
			log.Error("recover interrupted full reports failed", "err", recoveryErr)
		} else if recovered > 0 {
			log.Info("interrupted full reports requeued", "count", recovered)
		}
	}
	service.StartFullReportPDFRecovery(ctx, fullReportRenderJobRepo, fullReportRepo, jobQueue, time.Duration(cfg.JobStaleThresholdMinutes)*time.Minute, time.Minute, reportOutcomeNotifier)
	service.StartFullReportCleanup(ctx, fullReportCleanupJobRepo, jobQueue, time.Duration(cfg.JobStaleThresholdMinutes)*time.Minute, time.Hour)
	reportWorker.Start(ctx)
	pdfWorker.Start(ctx)
	cleanupWorker.Start(ctx)
	log.Info("job workers started", "report_workers", 3, "pdf_workers", 1, "cleanup_workers", 1)

	reportHTTPHandler := handler.NewReportHandler(reportSvc)

	// Payment providers (P5: 多渠道注册进 Registry)
	payReg := payment.NewRegistry()
	if contains(cfg.PaymentProviders, "stripe") {
		payReg.Register("stripe", payment.NewStripeProvider(cfg.StripeSecretKey, cfg.StripeWebhookSecret))
		log.Info("payment provider initialized", "type", "stripe")
	}
	if contains(cfg.PaymentProviders, "alipay") {
		aliProv, err := payment.NewAlipayProvider(cfg.AlipayAppID, cfg.AlipayPrivateKey, cfg.AlipayPublicKey, cfg.AlipayNotifyURL, cfg.AlipayReturnURL, cfg.AlipayProduction)
		if err != nil {
			log.Error("alipay provider init failed", "err", err)
			os.Exit(1)
		}
		payReg.Register("alipay", aliProv)
		log.Info("payment provider initialized", "type", "alipay")
	}
	if contains(cfg.PaymentProviders, "paypal") {
		payReg.Register("paypal", payment.NewPaypalProvider(cfg.PaypalClientID, cfg.PaypalSecret, cfg.PaypalWebhookID, cfg.PaypalProduction))
		log.Info("payment provider initialized", "type", "paypal")
	}

	paySvc := service.NewPaymentService(payReg, orderRepo)
	webhookHandler := handler.NewWebhookHandler(paySvc)

	// Dev-only Mock 支付渠道（PAYMENT_MOCK_ENABLED=true 时注册，生产绝不开启）
	var devPayHandler *handler.DevPayHandler
	if cfg.PaymentMockEnabled {
		mockBaseURL := cfg.AppBaseURL
		if mockBaseURL == "" {
			mockBaseURL = "http://localhost:" + cfg.AppPort
		}
		mockRef := payment.NewMockProviderRef(cfg.MockWebhookSecret, mockBaseURL)
		payReg.Register("mock", payment.NewMockProvider(cfg.MockWebhookSecret, mockBaseURL))
		log.Info("payment provider initialized", "type", "mock", "base_url", mockBaseURL)
		devPayHandler = handler.NewDevPayHandler(paySvc, mockRef)
	}

	orderSvc := service.NewOrderService(
		orderRepo, fullReportRepo, payReg,
		cfg.PaymentSuccessURL,
		cfg.PaymentCancelURL,
	)
	orderHTTPHandler := handler.NewOrderHandler(orderSvc)

	statsRepo := repository.NewStatsRepo(db)
	statsSvc := service.NewStatsService(statsRepo)
	adminUserSvc := service.NewAdminUserService(userRepo, orderRepo, fullReportRepo)
	adminOrderSvc := service.NewAdminOrderService(orderRepo, userRepo)
	adminReportSvc := service.NewAdminReportService(fullReportRepo, userRepo)
	adminHTTPHandler := handler.NewAdminHandler(statsSvc, adminUserSvc, adminOrderSvc, adminReportSvc)

	auditRepo := repository.NewAuditRepo(db)
	adminRegistry := resource.NewRegistry()
	adminRegistry.Register(resource.NewUsersResource(adminUserSvc))
	adminRegistry.Register(resource.NewOrdersResource(adminOrderSvc))
	adminRegistry.Register(resource.NewReportsResource(adminReportSvc))
	resourceHandler := handler.NewResourceHandler(adminRegistry, auditRepo)

	authHandler := handler.NewAuthHandler(authSvc, authReg, cfg.WebBaseURL)
	adminAuthHandler := handler.NewAdminAuthHandler(db, c, cfg.AdminJWTSecret, cfg.AdminJWTExpireHours)
	contentHandler := handler.NewContentHandler(db)
	pricingHandler := handler.NewPricingHandler(db)
	reportInputHandler := handler.NewReportInputHandler(profileSvc, reportSvc)
	profileHandler := handler.NewProfileHandler(profileSvc)
	chartHandler := handler.NewChartHandler(chartSvc)
	freeChartHandler := handler.NewFreeChartHandler(freeChartSvc)
	geoHandler := handler.NewGeoHandler(geoRepo)
	adminGeoHandler := handler.NewAdminGeoHandler(geoRepo, auditRepo)
	baziBaseCatalog := basedata.V1()
	if err := baziBaseCatalog.Validate(); err != nil {
		log.Error("bazi base data validation failed", "err", err, "version", baziBaseCatalog.Version)
		return
	}
	adminBaziBaseHandler := handler.NewAdminBaziBaseHandler(baziBaseCatalog, annualCalendarRepo)
	adminReportTraceHandler := handler.NewAdminReportTraceHandler(fullReportRepo, auditRepo, fullReportRenderJobRepo)
	adminCalculationHandler := handler.NewAdminCalculationHandler(db, baseChartEngine, auditRepo)
	adminLLMConfigHandler := handler.NewAdminLLMConfigHandler(db, auditRepo, cfg.AdminJWTSecret, cfg.AppEnv)
	adminSystemSettingHandler := handler.NewAdminSystemSettingHandler(systemSettingRepo, auditRepo, cfg.ReportChapterConcurrency, cfg.AppEnv)
	locationHandler := handler.NewLocationHandler(locationResolver)
	readingHandler := handler.NewReadingHandler(readingSvc)

	// Rate limiters — always non-nil (no-op passthrough when disabled)
	rlNoop := func(c *gin.Context) { c.Next() }
	rlAuth, rlReading, rlOrder := rlNoop, rlNoop, rlNoop
	if cfg.RateLimitEnabled {
		authLim := ratelimit.NewMemoryLimiter(rate.Limit(cfg.RateLimitAuthPerMin)/60, cfg.RateLimitAuthPerMin)
		readingLim := ratelimit.NewMemoryLimiter(rate.Limit(cfg.RateLimitReadingPerMin)/60, cfg.RateLimitReadingPerMin)
		orderLim := ratelimit.NewMemoryLimiter(rate.Limit(cfg.RateLimitOrderPerMin)/60, cfg.RateLimitOrderPerMin)
		rlAuth = middleware.RateLimit(authLim, middleware.KeyByIP)
		rlReading = middleware.RateLimit(readingLim, middleware.KeyByUser)
		rlOrder = middleware.RateLimit(orderLim, middleware.KeyByUser)
		log.Info("rate limiters initialized",
			"auth_per_min", cfg.RateLimitAuthPerMin,
			"reading_per_min", cfg.RateLimitReadingPerMin,
			"order_per_min", cfg.RateLimitOrderPerMin,
		)
	}

	app := &router.App{
		StaticDir:                 cfg.LocalStorageDir,
		DB:                        db,
		Auth:                      authMW,
		AdminAuth:                 adminAuthMW,
		HealthChecker:             router.NewDBHealthChecker(db),
		AuthHandler:               authHandler,
		AdminAuthHandler:          adminAuthHandler,
		ContentHandler:            contentHandler,
		PricingHandler:            pricingHandler,
		ReportInputHandler:        reportInputHandler,
		ProfHandler:               profileHandler,
		ChartHandler:              chartHandler,
		FreeChartHandler:          freeChartHandler,
		GeoHandler:                geoHandler,
		AdminGeoHandler:           adminGeoHandler,
		AdminBaziBaseHandler:      adminBaziBaseHandler,
		AdminReportTraceHandler:   adminReportTraceHandler,
		AdminCalculationHandler:   adminCalculationHandler,
		AdminLLMConfigHandler:     adminLLMConfigHandler,
		AdminSystemSettingHandler: adminSystemSettingHandler,
		LocationHandler:           locationHandler,
		ReadingHandler:            readingHandler,
		ReportHandler:             reportHTTPHandler,
		OrderHandler:              orderHTTPHandler,
		WebhookHandler:            webhookHandler,
		DevPayHandler:             devPayHandler,
		AdminHandler:              adminHTTPHandler,
		ResourceHandler:           resourceHandler,
		RateLimitAuth:             rlAuth,
		RateLimitReading:          rlReading,
		RateLimitOrder:            rlOrder,
	}
	engine := router.Setup(app)

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	srv := &http.Server{
		Addr:              addr,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start HTTP in goroutine
	go func() {
		log.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", "err", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("shutdown signal received", "signal", sig.String())

	// Phase 1: Stop HTTP server (stop accepting new requests)
	log.Info("shutting down HTTP server", "timeout", shutdownHTTPTimeout)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownHTTPTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP shutdown error", "err", err)
	} else {
		log.Info("HTTP server stopped")
	}

	// Phase 2: Stop job worker with timeout
	// Worker.Stop() blocks on wg.Wait(); wrap with timeout
	log.Info("shutting down job worker", "timeout", shutdownJobTimeout)
	jobDone := make(chan struct{})
	go func() {
		reportWorker.Stop()
		pdfWorker.Stop()
		cleanupWorker.Stop()
		close(jobDone)
	}()
	select {
	case <-jobDone:
		log.Info("job worker stopped cleanly")
	case <-time.After(shutdownJobTimeout):
		log.Warn("job worker stop timeout — in-flight jobs remain in processing state, will not be marked as completed")
	}

	log.Info("shutdown complete")
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.UserIdentity{},
		&model.BirthProfile{},
		&model.Chart{},
		&model.FreeChartRecord{},
		&model.GeoCountry{}, &model.GeoCity{},
		&model.Reading{},
		&model.FullReport{},
		&model.FullReportExecutionSnapshot{},
		&model.FullReportExecutionPayload{},
		&model.FullReportChapter{},
		&model.FullReportChapterPayload{},
		&model.FullReportAttempt{},
		&model.FullReportAttemptPayload{},
		&model.FullReportResult{},
		&model.FullReportRenderJob{},
		&model.FullReportCleanupJob{},
		&model.SystemSetting{},
		&model.CalculationArchive{},
		&model.CalculationVersion{},
		&model.PromptChapterConfig{},
		&model.LLMProviderConfig{},
		&model.LLMModelConfig{},
		&model.AnnualCalendarYear{},
		&model.Order{},
		&model.PaymentEvent{},
		&model.CreditLedger{},
		&model.DailyQuota{},
		&model.AdminUser{},
		&model.AdminRole{},
		&model.AdminAuditLog{},
		&model.ContentItem{},
		&model.PricingPlan{},
		&model.ProcessedWebhookEvent{},
		&job.Job{},
	)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
