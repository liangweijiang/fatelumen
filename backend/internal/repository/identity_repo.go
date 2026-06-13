package repository

import (
	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

type IdentityRepo struct {
	db *gorm.DB
}

func NewIdentityRepo(db *gorm.DB) *IdentityRepo {
	return &IdentityRepo{db: db}
}

func (r *IdentityRepo) FindByProvider(provider, externalID string) (*model.UserIdentity, error) {
	var id model.UserIdentity
	err := r.db.Where("provider = ? AND external_id = ?", provider, externalID).First(&id).Error
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *IdentityRepo) Create(identity *model.UserIdentity) error {
	return r.db.Create(identity).Error
}

func (r *IdentityRepo) ListByUser(userID uint64) ([]model.UserIdentity, error) {
	var ids []model.UserIdentity
	err := r.db.Where("user_id = ?", userID).Find(&ids).Error
	return ids, err
}
