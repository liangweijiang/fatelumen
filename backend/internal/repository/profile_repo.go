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
	// saved=false is meaningful for one-off report subjects. Select all fields
	// so GORM does not replace the explicit false zero value with the DB default.
	return r.db.Select("*").Create(profile).Error
}

// ListByUserID 列出用户所有档案。
func (r *ProfileRepo) ListByUserID(userID uint64) ([]model.BirthProfile, error) {
	var profiles []model.BirthProfile
	err := r.db.Where("user_id = ? AND saved = ?", userID, true).Order("created_at DESC").Find(&profiles).Error
	return profiles, err
}

func (r *ProfileRepo) Update(id uint64, updates map[string]interface{}) error {
	return r.db.Model(&model.BirthProfile{}).Where("id = ?", id).Updates(updates).Error
}

func (r *ProfileRepo) UpdateOwnedSaved(id, userID uint64, updates map[string]interface{}) error {
	return r.db.Model(&model.BirthProfile{}).
		Where("id = ? AND user_id = ? AND saved = ?", id, userID, true).
		Updates(updates).Error
}

func (r *ProfileRepo) FindSavedByIDAndUserID(id, userID uint64) (*model.BirthProfile, error) {
	var profile model.BirthProfile
	err := r.db.Where("id = ? AND user_id = ? AND saved = ?", id, userID, true).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
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

// FindByIDAndUserID returns both saved profiles and one-off report subjects,
// while enforcing ownership before a report job is created.
func (r *ProfileRepo) FindByIDAndUserID(id, userID uint64) (*model.BirthProfile, error) {
	var profile model.BirthProfile
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// Delete 删除档案。
func (r *ProfileRepo) Delete(id uint64) error {
	return r.db.Delete(&model.BirthProfile{}, id).Error
}
