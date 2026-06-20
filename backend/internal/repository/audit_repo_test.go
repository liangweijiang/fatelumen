package repository

import (
	"context"
	"testing"

	"fatelumen/backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestAuditRepo(t *testing.T) *AuditRepo {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AdminAuditLog{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return NewAuditRepo(db)
}

func TestAuditRepo_WriteThenList(t *testing.T) {
	repo := setupTestAuditRepo(t)
	ctx := context.Background()

	repo.Write(ctx, model.AdminAuditLog{
		AdminID:    1,
		AdminName:  "root",
		Action:     "ban",
		Resource:   "users",
		ResourceID: "42",
		IP:         "127.0.0.1",
	})

	items, total, err := repo.ListAudit(ctx, "users", 20, 0)
	if err != nil {
		t.Fatalf("ListAudit failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total: want 1, got %d", total)
	}
	if len(items) != 1 {
		t.Fatalf("items len: want 1, got %d", len(items))
	}
	got := items[0]
	if got.Action != "ban" || got.Resource != "users" || got.ResourceID != "42" {
		t.Errorf("audit row mismatch: %#v", got)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should be auto-filled")
	}
}

func TestAuditRepo_ListFiltersByResource(t *testing.T) {
	repo := setupTestAuditRepo(t)
	ctx := context.Background()

	repo.Write(ctx, model.AdminAuditLog{AdminID: 1, Action: "ban", Resource: "users", ResourceID: "1"})
	repo.Write(ctx, model.AdminAuditLog{AdminID: 1, Action: "refund", Resource: "orders", ResourceID: "2"})

	items, total, err := repo.ListAudit(ctx, "orders", 20, 0)
	if err != nil {
		t.Fatalf("ListAudit failed: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Resource != "orders" {
		t.Errorf("resource filter failed: total=%d items=%#v", total, items)
	}
}
