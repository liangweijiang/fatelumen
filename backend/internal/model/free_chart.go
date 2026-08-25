package model

import "time"

// FreeChartRecord is an authenticated user's independent chart history.
// It never creates a report, consumes credits, or mutates a birth profile.
type FreeChartRecord struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint64    `gorm:"not null;index:idx_free_chart_user_created" json:"-"`
	Gender       int8      `gorm:"not null" json:"gender"`
	CalendarType int8      `gorm:"not null" json:"calendar_type"`
	BirthYear    int16     `gorm:"not null" json:"birth_year"`
	BirthMonth   int8      `gorm:"not null" json:"birth_month"`
	BirthDay     int8      `gorm:"not null" json:"birth_day"`
	BirthHour    int8      `gorm:"not null" json:"birth_hour"`
	BirthMinute  int8      `gorm:"not null" json:"birth_minute"`
	IsLeapMonth  bool      `gorm:"not null;default:false" json:"is_leap_month"`
	CountryCode  string    `gorm:"type:varchar(8)" json:"country_code,omitempty"`
	CountryName  string    `gorm:"type:varchar(96)" json:"country_name,omitempty"`
	RegionCode   string    `gorm:"type:varchar(32)" json:"region_code,omitempty"`
	RegionName   string    `gorm:"type:varchar(96)" json:"region_name,omitempty"`
	City         string    `gorm:"type:varchar(96)" json:"city,omitempty"`
	PlaceID      string    `gorm:"type:varchar(128)" json:"place_id,omitempty"`
	DisplayName  string    `gorm:"type:varchar(256)" json:"display_name,omitempty"`
	TimezoneID   string    `gorm:"type:varchar(64);not null" json:"timezone_id"`
	Latitude     float64   `gorm:"type:decimal(9,6);not null" json:"latitude"`
	Longitude    float64   `gorm:"type:decimal(9,6);not null" json:"longitude"`
	ChartHash    string    `gorm:"type:char(64);not null;index" json:"chart_hash"`
	ChartData    ChartData `gorm:"type:json;not null" json:"chart_data"`
	CreatedAt    time.Time `gorm:"not null;index:idx_free_chart_user_created,sort:desc" json:"created_at"`
	UpdatedAt    time.Time `gorm:"not null" json:"updated_at"`
}

func (FreeChartRecord) TableName() string { return "free_chart_records" }
