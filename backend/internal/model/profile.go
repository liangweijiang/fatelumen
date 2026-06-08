package model

import "time"

// BirthProfile 出生信息档案（一个用户可存多个，如给家人测）。
type BirthProfile struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint64    `gorm:"not null;index" json:"user_id"`
	DisplayName  string    `gorm:"type:varchar(64)" json:"display_name"`
	Gender       int8      `gorm:"not null;comment:0=female 1=male" json:"gender"`
	CalendarType int8      `gorm:"not null;default:0;comment:0=solar 1=lunar" json:"calendar_type"`
	BirthYear    int16     `gorm:"not null" json:"birth_year"`
	BirthMonth   int8      `gorm:"not null" json:"birth_month"`
	BirthDay     int8      `gorm:"not null" json:"birth_day"`
	BirthHour    int8      `gorm:"not null;comment:0-23 unknown=-1" json:"birth_hour"`
	BirthMinute  int8      `gorm:"not null;default:0" json:"birth_minute"`
	IsLeapMonth  int8      `gorm:"not null;default:0" json:"is_leap_month"`
	BirthPlace   string    `gorm:"type:varchar(128)" json:"birth_place"`
	Timezone     string    `gorm:"type:varchar(48)" json:"timezone"`
	Longitude    float64   `gorm:"type:decimal(9,6)" json:"longitude"`
	CreatedAt    time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time `gorm:"not null" json:"updated_at"`
}

func (BirthProfile) TableName() string { return "birth_profiles" }
