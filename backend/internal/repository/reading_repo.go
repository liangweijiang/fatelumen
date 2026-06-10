package repository

import (
	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

// ReadingRepo 简单测算数据访问层。
type ReadingRepo struct {
	db *gorm.DB
}

func NewReadingRepo(db *gorm.DB) *ReadingRepo {
	return &ReadingRepo{db: db}
}

// Create 创建测算记录。
func (r *ReadingRepo) Create(reading *model.Reading) error {
	return r.db.Create(reading).Error
}

// GetByID 按 ID 查找测算记录，带 userID 归属校验防越权。
func (r *ReadingRepo) GetByID(id, userID uint64) (*model.Reading, error) {
	var reading model.Reading
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&reading).Error
	if err != nil {
		return nil, err
	}
	return &reading, nil
}

// ListByUser 列出用户所有测算记录。
func (r *ReadingRepo) ListByUser(userID uint64, limit, offset int) ([]model.Reading, error) {
	var readings []model.Reading
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&readings).Error
	return readings, err
}

// UpdateImageURL 更新测算记录的图片 URL。
func (r *ReadingRepo) UpdateImageURL(id uint64, imageURL string) error {
	return r.db.Model(&model.Reading{}).Where("id = ?", id).Update("image_url", imageURL).Error
}
