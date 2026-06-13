package model

import "time"

const (
	ProviderGoogle = "google"
	ProviderWechat = "wechat"
	ProviderEmail  = "email"
)

type UserIdentity struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint64    `gorm:"not null;index" json:"user_id"`
	Provider   string    `gorm:"type:varchar(16);not null;uniqueIndex:uk_provider_extid" json:"provider"`
	ExternalID string    `gorm:"type:varchar(128);not null;uniqueIndex:uk_provider_extid" json:"external_id"`
	Meta       string    `gorm:"type:varchar(255)" json:"meta"`
	CreatedAt  time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt  time.Time `gorm:"not null" json:"updated_at"`
}

func (UserIdentity) TableName() string { return "user_identities" }
