package model

import "time"

// PromptChapterConfig stores the current administrator-selected fact allow-list.
// Calculation and report snapshots remain immutable and do not reference this row.
type PromptChapterConfig struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChapterKey       string    `gorm:"type:varchar(64);not null;uniqueIndex" json:"chapter_key"`
	FactKeys         JSONRaw   `gorm:"type:json;not null" json:"fact_keys"`
	UpdatedByAdminID uint64    `gorm:"not null;index" json:"updated_by_admin_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (PromptChapterConfig) TableName() string { return "prompt_chapter_configs" }
