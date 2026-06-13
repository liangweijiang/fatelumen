package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"fatelumen/backend/internal/pkg/logger"
)

// LocalFSStorage 把字节写到本地文件系统，返回 file:// 绝对路径。
// 仅用于本地开发/调试，无 R2 时的兜底产物落盘方案。
type LocalFSStorage struct {
	baseDir string
}

// NewLocalFSStorage 创建本地文件存储，baseDir 为根目录（不存在则创建）。
func NewLocalFSStorage(baseDir string) (*LocalFSStorage, error) {
	if err := os.MkdirAll(baseDir, 0o750); err != nil {
		return nil, fmt.Errorf("create local storage dir: %w", err)
	}
	return &LocalFSStorage{baseDir: baseDir}, nil
}

func (s *LocalFSStorage) Put(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	fullPath := filepath.Join(s.baseDir, key)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
		logger.FromCtx(ctx).Error("local storage mkdir failed", "err", err, "key", key)
		return "", fmt.Errorf("local storage mkdir: %w", err)
	}
	if err := os.WriteFile(fullPath, data, 0o600); err != nil {
		logger.FromCtx(ctx).Error("local storage write failed", "err", err, "key", key)
		return "", fmt.Errorf("local storage write: %w", err)
	}
	abs, err := filepath.Abs(fullPath)
	if err != nil {
		abs = fullPath
	}
	url := "file://" + abs
	logger.FromCtx(ctx).Info("local storage put completed", "key", key, "path", abs, "bytes", len(data))
	return url, nil
}
