package storage

import (
	"context"
	"fmt"
)

type NoopStorage struct{}

func (s *NoopStorage) Put(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	return fmt.Sprintf("noop://%s", key), nil
}
