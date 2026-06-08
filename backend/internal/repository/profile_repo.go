package repository

import (
	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

// ProfileRepo 出生档案数据访问层。
type ProfileRepo struct {
	db *gorm.DB
}

func NewProfileRepo(db *gorm.DB) *ProfileRepo {
	return &ProfileRepo{db: db}
}

// Create 创建出生档案。
func (r *ProfileRepo) Create(profile *model.BirthProfile) error {
	return r.db.Create(profile).Error
}

// ListByUserID 列出用户所有档案。
func (r *ProfileRepo) ListByUserID(userID uint64) ([]model.BirthProfile, error) {
	var profiles []model.BirthProfile
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&profiles).Error
	return profiles, err
}

// FindByID 按 ID 查找档案。
func (r *ProfileRepo) FindByID(id uint64) (*model.BirthProfile, error) {
	var profile model.BirthProfile
	err := r.db.First(&profile, id).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// Delete 删除档案。
func (r *ProfileRepo) Delete(id uint64) error {
	return r.db.Delete(&model.BirthProfile{}, id).Error
}
