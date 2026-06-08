package main

import (
	"fmt"
	"log"

	"fatelumen/backend/internal/config"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/router"
	pkgLogger "fatelumen/backend/internal/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. 初始化日志
	zapLog, err := pkgLogger.New(cfg.AppEnv)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer zapLog.Sync()

	// 3. 连接数据库
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

	// 4. AutoMigrate 建表（Development 环境）
	if cfg.AppEnv == "development" {
		if err := autoMigrate(db); err != nil {
			zapLog.Fatal("failed to auto migrate", zap.Error(err))
		}
		zapLog.Info("auto migrate completed")
	}

	// 5. 依赖注入：初始化各接口实现（Phase 0 仅骨架）
	_ = initProviders(cfg, db, zapLog)

	// 6. 创建 Auth 中间件
	authMW := middleware.NewAuthMiddleware(cfg.JWTSecret, db)

	// 7. 组装路由
	app := &router.App{DB: db, Auth: authMW}
	engine := router.Setup(app)

	// 8. 启动服务
	addr := fmt.Sprintf(":%s", cfg.AppPort)
	zapLog.Info("server starting", zap.String("addr", addr))
	if err := engine.Run(addr); err != nil {
		zapLog.Fatal("server failed", zap.Error(err))
	}
}

// autoMigrate 自动建表（Development）。
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

// initProviders 占位：Phase 1-5 逐步注入具体实现。
// DECISION: 每个 Provider 按 .env 的 *_PROVIDER / *_PROVIDERS 选择实现注入。
func initProviders(cfg *config.Config, db *gorm.DB, log *zap.Logger) interface{} {
	// TODO Phase 1: AuthProvider (Google)
	// TODO Phase 3: LLMProvider (DeepSeek)
	// TODO Phase 4: JobQueue (goroutine), Renderer (chromedp), Notifier (Resend/noop)
	// TODO Phase 5: PaymentProvider (Stripe)
	// TODO Phase 6: Admin Resource Registry
	return nil
}
