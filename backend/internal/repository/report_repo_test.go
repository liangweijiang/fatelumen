package repository

import (
	"testing"
	"time"

	"fatelumen/backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestReportRepo(t *testing.T) (*ReportRepo, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Report{},
		&model.CreditLedger{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return NewReportRepo(db), db
}

func seedUnlockUser(t *testing.T, db *gorm.DB, id uint64, credits int) {
	t.Helper()
	u := &model.User{
		ID:        id,
		Email:     "unlock@test.com",
		Credits:   credits,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
}

func seedUnlockReport(t *testing.T, db *gorm.DB, id, userID uint64, paid bool) {
	t.Helper()
	r := &model.Report{
		ID:        id,
		UserID:    userID,
		ProfileID: 1,
		Status:    "done",
		Paid:      paid,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create report: %v", err)
	}
}

// ---------- TestUnlockReportWithCredits_Success ----------

func TestUnlockReportWithCredits_Success(t *testing.T) {
	repo, db := setupTestReportRepo(t)
	seedUnlockUser(t, db, 1, 100)
	seedUnlockReport(t, db, 10, 1, false)

	if err := repo.UnlockReportWithCredits(1, 10, 30); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 积分扣减 100 - 30 = 70
	var u model.User
	db.First(&u, 1)
	if u.Credits != 70 {
		t.Errorf("user credits: want 70, got %d", u.Credits)
	}

	// 报告解锁 + pay_method=credit
	var r model.Report
	db.First(&r, 10)
	if !r.Paid {
		t.Error("report should be paid")
	}
	if r.PayMethod != "credit" {
		t.Errorf("report pay_method: want credit, got %q", r.PayMethod)
	}

	// 负向流水一条
	var ledgers []model.CreditLedger
	db.Where("user_id = ?", 1).Find(&ledgers)
	if len(ledgers) != 1 {
		t.Fatalf("expected 1 ledger row, got %d", len(ledgers))
	}
	l := ledgers[0]
	if l.Delta != -30 {
		t.Errorf("ledger delta: want -30, got %d", l.Delta)
	}
	if l.BalanceAfter != 70 {
		t.Errorf("ledger balance_after: want 70, got %d", l.BalanceAfter)
	}
	if l.Reason != "unlock_report" {
		t.Errorf("ledger reason: want unlock_report, got %q", l.Reason)
	}
	if l.RefID == nil || *l.RefID != 10 {
		t.Errorf("ledger ref_id: want 10, got %v", l.RefID)
	}
}

// ---------- TestUnlockReportWithCredits_Insufficient ----------

func TestUnlockReportWithCredits_Insufficient(t *testing.T) {
	repo, db := setupTestReportRepo(t)
	seedUnlockUser(t, db, 1, 10)
	seedUnlockReport(t, db, 10, 1, false)

	err := repo.UnlockReportWithCredits(1, 10, 30)
	if err != ErrInsufficientCredits {
		t.Fatalf("expected ErrInsufficientCredits, got: %v", err)
	}

	// 积分不变
	var u model.User
	db.First(&u, 1)
	if u.Credits != 10 {
		t.Errorf("user credits: want 10 (unchanged), got %d", u.Credits)
	}

	// 报告未解锁
	var r model.Report
	db.First(&r, 10)
	if r.Paid {
		t.Error("report should NOT be paid")
	}

	// 无流水
	var count int64
	db.Model(&model.CreditLedger{}).Where("user_id = ?", 1).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 ledger rows, got %d", count)
	}
}

// ---------- TestUnlockReportWithCredits_Idempotent ----------

func TestUnlockReportWithCredits_Idempotent(t *testing.T) {
	repo, db := setupTestReportRepo(t)
	seedUnlockUser(t, db, 1, 100)
	seedUnlockReport(t, db, 10, 1, true) // 已解锁

	if err := repo.UnlockReportWithCredits(1, 10, 30); err != nil {
		t.Fatalf("expected nil (idempotent) for already-paid report, got: %v", err)
	}

	// 余额不再扣减
	var u model.User
	db.First(&u, 1)
	if u.Credits != 100 {
		t.Errorf("user credits: want 100 (unchanged), got %d", u.Credits)
	}

	// 无新流水
	var count int64
	db.Model(&model.CreditLedger{}).Where("user_id = ?", 1).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 ledger rows on idempotent unlock, got %d", count)
	}
}
