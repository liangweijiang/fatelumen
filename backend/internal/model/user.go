package model

import "time"

// User 用户（Google OAuth 登录）。
type User struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	GoogleSub      string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"google_sub"`
	Email          string    `gorm:"type:varchar(255);not null;index" json:"email"`
	Name           string    `gorm:"type:varchar(128)" json:"name"`
	AvatarURL      string    `gorm:"type:varchar(512)" json:"avatar_url"`
	Credits        int       `gorm:"not null;default:0" json:"credits"`
	Locale         string    `gorm:"type:varchar(8);not null;default:'en'" json:"locale"`
	CurrentTokenID string    `gorm:"type:varchar(64)" json:"-"`
	CreatedAt      time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt      time.Time `gorm:"not null" json:"updated_at"`
}

func (User) TableName() string { return "users" }
