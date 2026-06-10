package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ---------- Quick Reading JSON ----------

// QuickContent 简单测算 LLM 返回 JSON。
type QuickContent struct {
	SummaryLine string   `json:"summary_line"`
	Personality string   `json:"personality"`
	Strengths   []string `json:"strengths"`
	Weaknesses  []string `json:"weaknesses"`
	ElementNote string   `json:"element_note"`
}

// Value 实现 driver.Valuer。
func (c QuickContent) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan 实现 sql.Scanner。
func (c *QuickContent) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, c)
}

// ---------- Reading Model ----------

// Reading 简单测算记录（出图）。
type Reading struct {
	ID        uint64       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64       `gorm:"not null;index" json:"user_id"`
	ProfileID uint64       `gorm:"not null" json:"profile_id"`
	ChartID   uint64       `gorm:"not null" json:"chart_id"`
	Locale    string       `gorm:"type:varchar(8);not null;default:'en'" json:"locale"`
	Content   QuickContent `gorm:"type:json" json:"content"`
	ImageURL  string       `gorm:"type:varchar(512)" json:"image_url"`
	Status    string       `gorm:"type:varchar(16);not null;default:'done'" json:"status"`
	CreatedAt time.Time    `gorm:"not null" json:"created_at"`
}

func (Reading) TableName() string { return "readings" }
