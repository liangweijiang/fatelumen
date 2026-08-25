package repository

import (
	"fatelumen/backend/internal/model"
	"gorm.io/gorm"
)

type FreeChartRepo struct{ db *gorm.DB }

func NewFreeChartRepo(db *gorm.DB) *FreeChartRepo                   { return &FreeChartRepo{db: db} }
func (r *FreeChartRepo) Create(record *model.FreeChartRecord) error { return r.db.Create(record).Error }

func (r *FreeChartRepo) ListByUser(userID uint64, page, pageSize int) ([]model.FreeChartRecord, int64, error) {
	var records []model.FreeChartRecord
	var total int64
	query := r.db.Model(&model.FreeChartRecord{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error
	return records, total, err
}

func (r *FreeChartRepo) FindByIDAndUser(id, userID uint64) (*model.FreeChartRecord, error) {
	var record model.FreeChartRecord
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *FreeChartRepo) DeleteByIDAndUser(id, userID uint64) (int64, error) {
	result := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.FreeChartRecord{})
	return result.RowsAffected, result.Error
}

func (r *FreeChartRepo) BatchDeleteByUser(ids []uint64, userID uint64) (int64, error) {
	result := r.db.Where("user_id = ? AND id IN ?", userID, ids).Delete(&model.FreeChartRecord{})
	return result.RowsAffected, result.Error
}
