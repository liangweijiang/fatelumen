package storage

import "context"

// Storage 抽象对象存储（Cloudflare R2 / S3 / OSS）。
type Storage interface {
	// Put 上传字节流，返回可公开访问 URL。
	Put(ctx context.Context, key string, data []byte, contentType string) (url string, err error)
}
