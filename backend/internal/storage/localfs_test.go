package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalFSStorageDeleteIsIdempotentAndScoped(t *testing.T) {
	root := t.TempDir()
	store, err := NewLocalFSStorage(root, "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put(context.Background(), "reports/example.pdf", []byte("pdf"), "application/pdf"); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "reports", "example.pdf")
	if err := store.Delete(context.Background(), "reports/example.pdf"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("file still exists or stat failed unexpectedly: %v", err)
	}
	if err := store.Delete(context.Background(), "reports/example.pdf"); err != nil {
		t.Fatalf("repeated delete should succeed: %v", err)
	}
	if err := store.Delete(context.Background(), "../outside.pdf"); err == nil {
		t.Fatal("path traversal should be rejected")
	}
}
