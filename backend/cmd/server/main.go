package main

import (
	"fmt"

	"fatelumen/backend/internal/auth"
	"fatelumen/backend/internal/config"
	"fatelumen/backend/internal/handler"
	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/renderer"
	"fatelumen/backend/internal/repository"
	"fatelumen/backend/internal/router"
	"fatelumen/backend/internal/service"
	"fatelumen/backend/internal/storage"
	pkgLogger "fatelumen/backend/internal/pkg/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	log := pkgLogger.New(cfg.LogLevel)

	dbLogLevel := gormLogger.Warn
	if cfg.AppEnv == "development" {
		dbLogLevel = gormLogger.Info
	}
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: gormLogger.Default.LogMode(dbLogLevel),
	})
	if err != nil {
		log.Fatal("failed to connect database", "err", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	if cfg.AppEnv == "development" {
		if err := autoMigrate(db); err != nil {
			log.Fatal("failed to auto migrate", "err", err)
		}
		log.Info("auto migrate completed")
	}

	// 依赖注入
	authMW := middleware.NewAuthMiddleware(cfg.JWTSecret, db)

	userRepo := repository.NewUserRepo(db)
	profileRepo := repository.NewProfileRepo(db)
	chartRepo := repository.NewChartRepo(db)

	authReg := auth.NewRegistry()
	if contains(cfg.AuthProviders, "google") {
		authReg.Register(auth.NewGoogleProvider(
			cfg.GoogleClientID,
			cfg.GoogleClientSecret,
			cfg.GoogleRedirectURL,
		))
	}

	authSvc := service.NewAuthService(userRepo, authReg, cfg.JWTSecret, cfg.JWTExpireHours, log)
	profileSvc := service.NewProfileService(profileRepo)
	chartSvc := service.NewChartService(chartRepo, profileRepo)

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
	_ = imgRenderer // will be wired into reading service in sub-step 6
	log.Info("renderer initialized", "type", cfg.Renderer)

	var fileStorage storage.Storage
	if cfg.R2AccountID != "" {
		r2, err := storage.NewR2Storage(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket, cfg.R2PublicBase)
		if err != nil {
			log.Fatal("failed to init R2 storage", "err", err)
		}
		fileStorage = r2
		log.Info("storage initialized", "type", "r2")
	} else {
		fileStorage = &storage.NoopStorage{}
		log.Info("storage initialized", "type", "noop")
	}
	_ = fileStorage // will be wired into reading service in sub-step 6

	authHandler := handler.NewAuthHandler(authSvc, authReg)
	profileHandler := handler.NewProfileHandler(profileSvc)
	chartHandler := handler.NewChartHandler(chartSvc)

	app := &router.App{
		DB:           db,
		Auth:         authMW,
		AuthHandler:  authHandler,
		ProfHandler:  profileHandler,
		ChartHandler: chartHandler,
	}
	engine := router.Setup(app)

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Info("server starting", "addr", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatal("server failed", "err", err)
	}
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.BirthProfile{},
		&model.Chart{},
		&model.Reading{},
		&model.Report{},
		&model.Order{},
		&model.PaymentEvent{},
		&model.CreditLedger{},
		&model.DailyQuota{},
		&model.AdminUser{},
		&model.AdminRole{},
		&model.AdminAuditLog{},
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
