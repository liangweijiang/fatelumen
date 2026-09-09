package model

import "time"

// SystemSetting stores small operational values. Large execution contracts are
// still frozen in their domain tables when a report starts.
type SystemSetting struct {
	Key       string    `gorm:"type:varchar(64);primaryKey" json:"key"`
	Value     string    `gorm:"type:varchar(512);not null" json:"value"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

func (SystemSetting) TableName() string { return "system_settings" }
