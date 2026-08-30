package model

import "time"

// CalculationArchive is an administrator-owned reusable deterministic input.
// Results are append-only CalculationVersion rows so a recalculation never
// destroys the evidence used by an earlier prompt or report.
type CalculationArchive struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string    `gorm:"type:varchar(128);not null" json:"name"`
	Status           string    `gorm:"type:varchar(24);not null;default:ready;index" json:"status"`
	Input            JSONRaw   `gorm:"type:json;not null" json:"input"`
	LatestVersionID  *uint64   `gorm:"index" json:"latest_version_id,omitempty"`
	LatestVersionNo  int       `gorm:"not null;default:0" json:"latest_version_no"`
	LastError        string    `gorm:"type:text" json:"last_error,omitempty"`
	CreatedByAdminID uint64    `gorm:"not null;index" json:"created_by_admin_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (CalculationArchive) TableName() string { return "calculation_archives" }

type CalculationVersion struct {
	ID                      uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ArchiveID               uint64    `gorm:"not null;uniqueIndex:idx_calculation_archive_version;index" json:"archive_id"`
	VersionNo               int       `gorm:"not null;uniqueIndex:idx_calculation_archive_version" json:"version_no"`
	InputSnapshot           JSONRaw   `gorm:"type:json;not null" json:"input_snapshot"`
	TimeCalculationSnapshot JSONRaw   `gorm:"type:json;not null" json:"time_calculation_snapshot"`
	ChartSnapshot           JSONRaw   `gorm:"type:json;not null" json:"chart_snapshot"`
	FactsSnapshot           JSONRaw   `gorm:"type:json;not null" json:"facts_snapshot"`
	ChartHash               string    `gorm:"type:char(64);not null;index" json:"chart_hash"`
	FactsHash               string    `gorm:"type:char(64);not null;index" json:"facts_hash"`
	CreatedAt               time.Time `json:"created_at"`
}

func (CalculationVersion) TableName() string { return "calculation_versions" }
