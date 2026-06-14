package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fatelumen/backend/internal/pkg/logger"
)

// LocalFSStorage 把字节写到本地文件系统，返回可被浏览器访问的 HTTP URL。
// 仅用于本地开发/调试，无 R2 时的兜底产物落盘方案。
// publicBase 形如 http://localhost:8080，落盘对象通过 /static/<key> 暴露。
type LocalFSStorage struct {
	baseDir    string
	publicBase string
}

// NewLocalFSStorage 创建本地文件存储，baseDir 为根目录（不存在则创建）。
// publicBase 为对外可访问的基地址（不含尾斜杠），用于拼出 HTTP 下载 URL。
func NewLocalFSStorage(baseDir, publicBase string) (*LocalFSStorage, error) {
	if err := os.MkdirAll(baseDir, 0o750); err != nil {
		return nil, fmt.Errorf("create local storage dir: %w", err)
	}
	return &LocalFSStorage{
		baseDir:    baseDir,
		publicBase: strings.TrimRight(publicBase, "/"),
	}, nil
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
	url := s.publicBase + "/static/" + filepath.ToSlash(key)
	logger.FromCtx(ctx).Info("local storage put completed", "key", key, "url", url, "bytes", len(data))
	return url, nil
}
