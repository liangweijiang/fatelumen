package repository

import (
	"testing"
	"time"

	"fatelumen/backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestOrderRepo(t *testing.T) (*OrderRepo, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Order{},
		&model.Report{},
		&model.ProcessedWebhookEvent{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return NewOrderRepo(db), db
}

func createTestReport(t *testing.T, db *gorm.DB, id uint64) {
	t.Helper()
	report := &model.Report{
		ID:        id,
		UserID:    1,
		ProfileID: 1,
		Status:    "done",
		Paid:      false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.Create(report).Error; err != nil {
		t.Fatalf("failed to create report: %v", err)
	}
}

func createTestOrder(t *testing.T, db *gorm.DB, id uint64, reportID uint64, status string) *model.Order {
	t.Helper()
	order := &model.Order{
		ID:         id,
		UserID:     1,
		ReportID:   reportID,
		Type:       "report",
		SKU:        "report_single",
		AmountCents: 999,
		Currency:   "usd",
		Provider:   "stripe",
		Status:     status,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("failed to create order: %v", err)
	}
	return order
}

// ---------- TestFulfillPaidOrder_Success ----------

func TestFulfillPaidOrder_Success(t *testing.T) {
	repo, db := setupTestOrderRepo(t)
	createTestReport(t, db, 1)
	createTestOrder(t, db, 10, 1, model.OrderStatusCreated)

	err := repo.FulfillPaidOrder("stripe", "evt_success", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify order is paid
	var order model.Order
	if err := db.First(&order, 10).Error; err != nil {
		t.Fatalf("failed to load order: %v", err)
	}
	if order.Status != model.OrderStatusPaid {
		t.Errorf("order status: want %s, got %s", model.OrderStatusPaid, order.Status)
	}

	// Verify report is unlocked
	var report model.Report
	if err := db.First(&report, 1).Error; err != nil {
		t.Fatalf("failed to load report: %v", err)
	}
	if !report.Paid {
		t.Error("report should be paid")
	}
	if report.OrderID == nil || *report.OrderID != 10 {
		t.Errorf("report order_id should be 10, got %v", report.OrderID)
	}

	// Verify webhook event is recorded
	var count int64
	db.Model(&model.ProcessedWebhookEvent{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 webhook event, got %d", count)
	}
}

// ---------- TestFulfillPaidOrder_DuplicateEvent ----------

func TestFulfillPaidOrder_DuplicateEvent(t *testing.T) {
	repo, db := setupTestOrderRepo(t)
	createTestReport(t, db, 1)
	createTestOrder(t, db, 10, 1, model.OrderStatusCreated)

	// First call: success
	err := repo.FulfillPaidOrder("stripe", "evt_dup", 10)
	if err != nil {
		t.Fatalf("first call unexpected error: %v", err)
	}

	// Snapshot post-first-call state
	var orderAfter model.Order
	db.First(&orderAfter, 10)
	var reportAfter model.Report
	db.First(&reportAfter, 1)

	// Second call: same (provider, event_id) → ErrDuplicateEvent
	err = repo.FulfillPaidOrder("stripe", "evt_dup", 10)
	if err != ErrDuplicateEvent {
		t.Fatalf("expected ErrDuplicateEvent, got: %v", err)
	}

	// Order must not be re-written
	var orderFinal model.Order
	db.First(&orderFinal, 10)
	if orderFinal.UpdatedAt != orderAfter.UpdatedAt {
		t.Error("order should not be re-written on duplicate event")
	}
	if orderFinal.Status != model.OrderStatusPaid {
		t.Errorf("order status should remain %s, got %s", model.OrderStatusPaid, orderFinal.Status)
	}

	// Report must not be re-written
	var reportFinal model.Report
	db.First(&reportFinal, 1)
	if reportFinal.UpdatedAt != reportAfter.UpdatedAt {
		t.Error("report should not be re-written on duplicate event")
	}
	if !reportFinal.Paid {
		t.Error("report should still be paid")
	}

	// Webhook event count = 1 (only first insert succeeded)
	var count int64
	db.Model(&model.ProcessedWebhookEvent{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 webhook event (no second insert), got %d", count)
	}
}

// ---------- TestFulfillPaidOrder_RollbackOnReportFail ----------

func TestFulfillPaidOrder_RollbackOnReportFail(t *testing.T) {
	repo, db := setupTestOrderRepo(t)
	createTestReport(t, db, 1)
	createTestOrder(t, db, 10, 1, model.OrderStatusCreated)

	// Drop the reports table to force the UPDATE to fail
	if err := db.Exec("DROP TABLE reports").Error; err != nil {
		t.Fatalf("failed to drop reports table: %v", err)
	}

	err := repo.FulfillPaidOrder("stripe", "evt_rollback", 10)
	if err == nil {
		t.Fatal("expected error when report update fails, got nil")
	}

	// Recover reports table to verify state
	// Re-migrate brings back table but not data
	// We only need to verify webhook_event and order counts
	var webhookCount int64
	db.Model(&model.ProcessedWebhookEvent{}).Count(&webhookCount)
	if webhookCount != 0 {
		t.Errorf("webhook_event should have 0 rows (rolled back), got %d", webhookCount)
	}

	// Order status should be unchanged (created, not paid)
	// We need to re-migrate orders too since sqlite in-memory might have issues
	// Actually, gorm in-memory sqlite keeps data across migrations if table exists.
	// The orders table was not dropped, so it still has the order.
	var order model.Order
	if err := db.First(&order, 10).Error; err != nil {
		t.Fatalf("failed to load order: %v", err)
	}
	if order.Status != model.OrderStatusCreated {
		t.Errorf("order status should be %s (rolled back), got %s", model.OrderStatusCreated, order.Status)
	}
}

// ---------- TestFulfillPaidOrder_AlreadyPaidIdempotent ----------

func TestFulfillPaidOrder_AlreadyPaidIdempotent(t *testing.T) {
	repo, db := setupTestOrderRepo(t)
	createTestReport(t, db, 1)
	createTestOrder(t, db, 10, 1, model.OrderStatusPaid) // already paid

	err := repo.FulfillPaidOrder("stripe", "evt_new_for_paid", 10)
	if err != nil {
		t.Fatalf("expected nil (idempotent) for already-paid order, got: %v", err)
	}

	// Webhook event should be recorded (new event_id)
	var count int64
	db.Model(&model.ProcessedWebhookEvent{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 webhook event recorded, got %d", count)
	}

	// Order should still be paid
	var order model.Order
	db.First(&order, 10)
	if order.Status != model.OrderStatusPaid {
		t.Errorf("order should still be %s, got %s", model.OrderStatusPaid, order.Status)
	}

	// Report should not be re-unlocked
	var report model.Report
	db.First(&report, 1)
	if report.Paid {
		t.Error("report should NOT be re-unlocked (was unpaid before this call too)")
	}
}

// ---------- TestFulfillPaidOrder_IllegalTransition ----------

func TestFulfillPaidOrder_IllegalTransition(t *testing.T) {
	repo, db := setupTestOrderRepo(t)
	createTestReport(t, db, 1)
	createTestOrder(t, db, 10, 1, model.OrderStatusFailed) // failed → paid is illegal

	err := repo.FulfillPaidOrder("stripe", "evt_illegal", 10)
	if err == nil {
		t.Fatal("expected error for illegal transition, got nil")
	}

	// Webhook event must NOT be recorded (transaction rolled back)
	var count int64
	db.Model(&model.ProcessedWebhookEvent{}).Count(&count)
	if count != 0 {
		t.Errorf("webhook_event should have 0 rows (rolled back), got %d", count)
	}

	// Order must still be "failed"
	var order model.Order
	db.First(&order, 10)
	if order.Status != model.OrderStatusFailed {
		t.Errorf("order status should be %s (rolled back), got %s", model.OrderStatusFailed, order.Status)
	}

	// Report must NOT be paid
	var report model.Report
	db.First(&report, 1)
	if report.Paid {
		t.Error("report should NOT be paid after rollback")
	}
}

// ---------- TestFulfillPaidOrder_OrderNotFound ----------

func TestFulfillPaidOrder_OrderNotFound(t *testing.T) {
	repo, _ := setupTestOrderRepo(t)

	err := repo.FulfillPaidOrder("stripe", "evt_ghost", 999)
	if err == nil {
		t.Fatal("expected error for non-existent order")
	}
	if err == ErrDuplicateEvent {
		t.Error("should not return ErrDuplicateEvent for missing order")
	}
}

// ---------- TestFulfillPaidOrder_NoReport ----------

func TestFulfillPaidOrder_NoReport(t *testing.T) {
	repo, db := setupTestOrderRepo(t)
	createTestOrder(t, db, 10, 0, model.OrderStatusCreated) // ReportID=0 (no report)

	err := repo.FulfillPaidOrder("stripe", "evt_no_report", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Order should be paid
	var order model.Order
	db.First(&order, 10)
	if order.Status != model.OrderStatusPaid {
		t.Errorf("order should be paid, got %s", order.Status)
	}

	// Webhook event should be recorded
	var count int64
	db.Model(&model.ProcessedWebhookEvent{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 webhook event, got %d", count)
	}
}
