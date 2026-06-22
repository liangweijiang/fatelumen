package repository

import (
	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

// UserRepo 用户数据访问层。
type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// FindByGoogleSub 按 Google sub 查找用户。
func (r *UserRepo) FindByGoogleSub(googleSub string) (*model.User, error) {
	var user model.User
	err := r.db.Where("google_sub = ?", googleSub).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID 按 ID 查找用户。
func (r *UserRepo) FindByID(id uint64) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpsertByGoogleSub 按 google_sub 创建或更新用户。
func (r *UserRepo) UpsertByGoogleSub(user *model.User) (*model.User, error) {
	existing, err := r.FindByGoogleSub(user.GoogleSub)
	if err == nil {
		existing.Email = user.Email
		existing.Name = user.Name
		existing.AvatarURL = user.AvatarURL
		if existing.Locale == "" {
			existing.Locale = "en"
		}
		if err := r.db.Save(existing).Error; err != nil {
			return nil, err
		}
		return existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if user.Locale == "" {
		user.Locale = "en"
	}
	if err := r.db.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateCurrentToken 更新当前会话令牌。
func (r *UserRepo) UpdateCurrentToken(userID uint64, tokenID string) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).
		Update("current_token_id", tokenID).Error
}

// ClearCurrentToken 清除当前会话令牌（登出）。
func (r *UserRepo) ClearCurrentToken(userID uint64) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).
		Update("current_token_id", "").Error
}

// UpdateFields 更新指定字段。
func (r *UserRepo) UpdateFields(userID uint64, updates map[string]interface{}) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}

// GetCurrentToken 获取用户当前有效 token_id（用于单设备检查）。
func (r *UserRepo) GetCurrentToken(userID uint64) (string, error) {
	var user model.User
	err := r.db.Select("current_token_id").First(&user, userID).Error
	if err != nil {
		return "", err
	}
	return user.CurrentTokenID, nil
}

// ListUsers 分页搜索用户（按 email/name 模糊匹配）。
func (r *UserRepo) ListUsers(keyword string, limit, offset int) ([]model.User, int64, error) {
	query := r.db.Model(&model.User{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("email LIKE ? OR name LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []model.User
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&users).Error
	return users, total, err
}

// GetUserByID 按 ID 查找用户（全字段，admin 侧用）。
func (r *UserRepo) GetUserByID(id uint64) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// SetUserActive 启用/禁用用户。
func (r *UserRepo) SetUserActive(id uint64, active bool) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("active", active).Error
}

func (r *UserRepo) SetUserUnlimited(id uint64, unlimited bool) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("unlimited", unlimited).Error
}

// FindByEmail 按 email 查找用户（邮箱密码登录用）。
func (r *UserRepo) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser 创建一个新用户（邮箱注册用），并写入一条 email identity。
// 在同一事务里完成，保证用户与身份一致。
func (r *UserRepo) CreateUser(user *model.User) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if user.Locale == "" {
			user.Locale = "en"
		}
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		identity := &model.UserIdentity{
			UserID:     user.ID,
			Provider:   model.ProviderEmail,
			ExternalID: user.Email,
		}
		return tx.Create(identity).Error
	})
}

// EmailsByIDs 按用户 id 批量查邮箱，返回 id->email 映射（admin 列表填充用，避免 N+1）。
func (r *UserRepo) EmailsByIDs(ids []uint64) (map[uint64]string, error) {
	out := make(map[uint64]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	type row struct {
		ID    uint64
		Email string
	}
	var rows []row
	if err := r.db.Model(&model.User{}).Select("id", "email").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, x := range rows {
		out[x.ID] = x.Email
	}
	return out, nil
}
