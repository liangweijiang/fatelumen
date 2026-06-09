package repository

import (
	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

type ChartRepo struct {
	db *gorm.DB
}

func NewChartRepo(db *gorm.DB) *ChartRepo {
	return &ChartRepo{db: db}
}

func (r *ChartRepo) FindByHash(hash string) (*model.Chart, error) {
	var chart model.Chart
	err := r.db.Where("chart_hash = ?", hash).First(&chart).Error
	if err != nil {
		return nil, err
	}
	return &chart, nil
}

func (r *ChartRepo) Create(chart *model.Chart) error {
	return r.db.Create(chart).Error
}

func (r *ChartRepo) ListByProfileID(profileID uint64) ([]model.Chart, error) {
	var charts []model.Chart
	err := r.db.Where("profile_id = ?", profileID).Order("created_at DESC").Find(&charts).Error
	return charts, err
}

func (r *ChartRepo) FindByID(id uint64) (*model.Chart, error) {
	var chart model.Chart
	err := r.db.First(&chart, id).Error
	if err != nil {
		return nil, err
	}
	return &chart, nil
}

func (r *ChartRepo) FindByIDAndProfileID(id, profileID uint64) (*model.Chart, error) {
	var chart model.Chart
	err := r.db.Where("id = ? AND profile_id = ?", id, profileID).First(&chart).Error
	if err != nil {
		return nil, err
	}
	return &chart, nil
}
