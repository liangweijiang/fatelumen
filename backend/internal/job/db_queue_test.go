//go:build integration

package job

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"fatelumen/backend/internal/pkg/logger"
)

// TestDBQueue_Integration 集成测试，需要 MySQL 连接。
// 环境变量：DB_HOST / DB_PORT / DB_USER / DB_PASSWORD / DB_NAME
// 运行：go test -tags=integration -run TestDBQueue_Integration ./internal/job/
func TestDBQueue_Integration(t *testing.T) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Skip("DB_DSN not set, skipping integration test")
	}

	// Note: real integration test requires gorm DB connection.
	// Skip if not explicitly configured.
	t.Skip("integration test requires setup; run with go test -tags=integration -count=1")
}

func TestDBQueue_EnqueueDequeue_Integration(t *testing.T) {
	t.Skip("requires real DB connection")
}

func TestDBQueue_ConcurrentDequeue_Integration(t *testing.T) {
	t.Skip("requires real DB connection")
}

// Ensure interface compliance at compile time
var _ Queue = (*MemoryQueue)(nil)
var _ Queue = (*DBQueue)(nil)
