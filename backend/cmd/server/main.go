package main

import (
	"fmt"
	"log"

	"fatelumen/backend/internal/auth"
	"fatelumen/backend/internal/config"
	"fatelumen/backend/internal/handler"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/repository"
	"fatelumen/backend/internal/router"
	"fatelumen/backend/internal/service"
	pkgLogger "fatelumen/backend/internal/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	zapLog, err := pkgLogger.New(cfg.AppEnv)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer zapLog.Sync()

	dbLogLevel := gormLogger.Warn
	if cfg.AppEnv == "development" {
		dbLogLevel = gormLogger.Info
	}
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: gormLogger.Default.LogMode(dbLogLevel),
	})
	if err != nil {
		zapLog.Fatal("failed to connect database", zap.Error(err))
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	if cfg.AppEnv == "development" {
		if err := autoMigrate(db); err != nil {
			zapLog.Fatal("failed to auto migrate", zap.Error(err))
		}
		zapLog.Info("auto migrate completed")
	}

	// 依赖注入
	authMW := middleware.NewAuthMiddleware(cfg.JWTSecret, db)

	userRepo := repository.NewUserRepo(db)
	profileRepo := repository.NewProfileRepo(db)

	authReg := auth.NewRegistry()
	if contains(cfg.AuthProviders, "google") {
		authReg.Register(auth.NewGoogleProvider(
			cfg.GoogleClientID,
			cfg.GoogleClientSecret,
			cfg.GoogleRedirectURL,
		))
	}

	authSvc := service.NewAuthService(userRepo, authReg, cfg.JWTSecret, cfg.JWTExpireHours, zapLog)
	profileSvc := service.NewProfileService(profileRepo)

	authHandler := handler.NewAuthHandler(authSvc, authReg)
	profileHandler := handler.NewProfileHandler(profileSvc)

	app := &router.App{
		DB:          db,
		Auth:        authMW,
		AuthHandler: authHandler,
		ProfHandler: profileHandler,
	}
	engine := router.Setup(app)

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	zapLog.Info("server starting", zap.String("addr", addr))
	if err := engine.Run(addr); err != nil {
		zapLog.Fatal("server failed", zap.Error(err))
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
