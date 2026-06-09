package storage

import (
	"context"
	"strings"
	"testing"
)

func TestReadingKey(t *testing.T) {
	key := ReadingKey(42, 7)
	if key != "readings/42/7.png" {
		t.Errorf("expected readings/42/7.png, got %s", key)
	}
	if !strings.HasPrefix(key, "readings/") {
		t.Error("key should start with readings/")
	}
	if !strings.HasSuffix(key, ".png") {
		t.Error("key should end with .png")
	}
}

func TestReadingKeyUniqueness(t *testing.T) {
	k1 := ReadingKey(1, 100)
	k2 := ReadingKey(100, 1)
	if k1 == k2 {
		t.Error("different user/reading pairs should produce different keys")
	}
	if ReadingKey(1, 1) != ReadingKey(1, 1) {
		t.Error("same input should produce same key")
	}
}

type noopStorage struct{}

func (s *noopStorage) Put(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	return "https://cdn.example.com/" + key, nil
}

func TestNoopStorage_Coverage(t *testing.T) {
	var s Storage = &noopStorage{}
	url, err := s.Put(context.Background(), "readings/1/1.png", []byte("test"), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://cdn.example.com/readings/1/1.png" {
		t.Errorf("unexpected url: %s", url)
	}
}
