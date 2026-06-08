package model

import "time"

// DailyQuota 每日免费额度（若不用 Redis）。
type DailyQuota struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"not null;uniqueIndex:uk_user_date" json:"user_id"`
	QuotaDate time.Time `gorm:"type:date;not null;uniqueIndex:uk_user_date" json:"quota_date"`
	UsedCount int       `gorm:"not null;default:0" json:"used_count"`
}

func (DailyQuota) TableName() string { return "daily_quota" }
