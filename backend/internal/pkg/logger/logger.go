package logger

import (
	"log/slog"
	"os"
)

// Logger 基于 log/slog 的薄封装。业务代码只依赖此类型，不直接 import log/slog。
type Logger struct {
	slog *slog.Logger
}

// New 创建 Logger 实例。level: "debug" / "info" / "warn" / "error"。
func New(level string) *Logger {
	var l slog.Leveler
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l})
	return &Logger{slog: slog.New(handler)}
}

// Info 打印 info 级别日志。args 为 key-value 对。
func (l *Logger) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
}

// Warn 打印 warn 级别日志。
func (l *Logger) Warn(msg string, args ...any) {
	l.slog.Warn(msg, args...)
}

// Error 打印 error 级别日志。
func (l *Logger) Error(msg string, args ...any) {
	l.slog.Error(msg, args...)
}

// Debug 打印 debug 级别日志。
func (l *Logger) Debug(msg string, args ...any) {
	l.slog.Debug(msg, args...)
}

// Fatal 打印 error 级别日志后退出进程。
func (l *Logger) Fatal(msg string, args ...any) {
	l.slog.Error(msg, args...)
	os.Exit(1)
}

// With 返回一个带有预设结构化字段的子 Logger。
func (l *Logger) With(args ...any) *Logger {
	return &Logger{slog: l.slog.With(args...)}
}
