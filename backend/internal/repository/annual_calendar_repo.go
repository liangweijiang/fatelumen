package repository

import (
	"context"

	"fatelumen/backend/internal/bazi/annualcalendar"
	"fatelumen/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AnnualCalendarRepo struct{ db *gorm.DB }

func NewAnnualCalendarRepo(db *gorm.DB) *AnnualCalendarRepo { return &AnnualCalendarRepo{db: db} }

func (r *AnnualCalendarRepo) Seed(ctx context.Context, startYear, count int) error {
	rows, err := annualcalendar.Generate(startYear, count)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(rows, 100).Error
}

func (r *AnnualCalendarRepo) Range(ctx context.Context, startYear, count int) ([]model.AnnualCalendarYear, error) {
	var rows []model.AnnualCalendarYear
	err := r.db.WithContext(ctx).Where("year >= ? AND year < ?", startYear, startYear+count).Order("year ASC").Find(&rows).Error
	return rows, err
}

func (r *AnnualCalendarRepo) List(ctx context.Context, startYear, endYear, page, size int) ([]model.AnnualCalendarYear, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.AnnualCalendarYear{})
	if startYear > 0 {
		query = query.Where("year >= ?", startYear)
	}
	if endYear > 0 {
		query = query.Where("year <= ?", endYear)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.AnnualCalendarYear
	err := query.Order("year ASC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}
