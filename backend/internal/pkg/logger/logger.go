package logger

import "go.uber.org/zap"

// New 创建 zap 日志实例。
func New(env string) (*zap.Logger, error) {
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
