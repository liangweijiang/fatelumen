//go:build integration

package storage

import (
	"context"
	"os"
	"testing"
)

// TestR2Put 真实上传 R2，需设置环境变量：
//
//	R2_ACCOUNT_ID       — Cloudflare 账户 ID
//	R2_ACCESS_KEY_ID     — R2 access key
//	R2_SECRET_ACCESS_KEY — R2 secret
//	R2_BUCKET            — bucket 名称
//	R2_PUBLIC_BASE       — 公开 URL 前缀
//
// 运行：go test -tags=integration -run TestR2Put ./internal/storage/
func TestR2Put(t *testing.T) {
	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKey := os.Getenv("R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucket := os.Getenv("R2_BUCKET")
	publicBase := os.Getenv("R2_PUBLIC_BASE")

	if accountID == "" || accessKey == "" || secretKey == "" || bucket == "" {
		t.Skip("R2 credentials not set, skipping integration test")
	}

	r2, err := NewR2Storage(accountID, accessKey, secretKey, bucket, publicBase)
	if err != nil {
		t.Fatalf("NewR2Storage: %v", err)
	}

	key := "test/integration_test.png"
	data := []byte("hello R2 integration test")
	url, err := r2.Put(context.Background(), key, data, "text/plain")
	if err != nil {
		t.Fatalf("R2 Put: %v", err)
	}
	t.Logf("Uploaded to: %s", url)
	if url == "" {
		t.Error("empty URL returned")
	}
}
