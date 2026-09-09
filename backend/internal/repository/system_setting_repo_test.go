package repository

import (
	"context"
	"testing"

	"fatelumen/backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSystemSettingReportConcurrencyFallbackAndSave(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:system-settings?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatal(err)
	}
	repo := NewSystemSettingRepo(db)
	value, err := repo.ReportChapterConcurrency(context.Background(), 3)
	if err != nil || value != 3 {
		t.Fatalf("fallback value=%d err=%v", value, err)
	}
	if err := repo.SaveReportChapterConcurrency(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	value, err = repo.ReportChapterConcurrency(context.Background(), 3)
	if err != nil || value != 7 {
		t.Fatalf("saved value=%d err=%v", value, err)
	}
	if err := repo.SaveReportChapterConcurrency(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	value, err = repo.ReportChapterConcurrency(context.Background(), 3)
	if err != nil || value != 2 {
		t.Fatalf("updated value=%d err=%v", value, err)
	}
}
