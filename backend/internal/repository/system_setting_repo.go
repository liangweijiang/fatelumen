package repository

import (
	"context"
	"strconv"
	"time"

	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const ReportChapterConcurrencyKey = "report.chapter_concurrency"

type SystemSettingRepo struct{ db *gorm.DB }

func NewSystemSettingRepo(db *gorm.DB) *SystemSettingRepo { return &SystemSettingRepo{db: db} }

func (r *SystemSettingRepo) ReportChapterConcurrency(ctx context.Context, fallback int) (int, error) {
	var row model.SystemSetting
	// `key` is reserved by MySQL, so keep the identifier quoted in explicit
	// predicates (GORM already quotes it for INSERT/ON CONFLICT clauses).
	err := r.db.WithContext(ctx).First(&row, "`key` = ?", ReportChapterConcurrencyKey).Error
	if err == gorm.ErrRecordNotFound {
		return fallback, nil
	}
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(row.Value)
	if err != nil || value < 1 || value > 10 {
		return fallback, nil
	}
	return value, nil
}

func (r *SystemSettingRepo) SaveReportChapterConcurrency(ctx context.Context, value int) error {
	row := model.SystemSetting{Key: ReportChapterConcurrencyKey, Value: strconv.Itoa(value), UpdatedAt: time.Now().UTC()}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"})}).Create(&row).Error
}
