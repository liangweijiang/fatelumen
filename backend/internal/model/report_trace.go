package model

import "time"

// ReportFactSnapshot is the immutable deterministic input used by one report.
type ReportFactSnapshot struct {
	ID                      uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ReportID                uint64    `gorm:"not null;uniqueIndex" json:"report_id"`
	InputSnapshot           JSONRaw   `gorm:"type:json;not null" json:"input_snapshot"`
	TimeCalculationSnapshot JSONRaw   `gorm:"type:json;not null" json:"time_calculation_snapshot"`
	ChartSnapshot           JSONRaw   `gorm:"type:json;not null" json:"chart_snapshot"`
	FactsSnapshot           JSONRaw   `gorm:"type:json;not null" json:"facts_snapshot"`
	ChartHash               string    `gorm:"type:char(64);not null;index" json:"chart_hash"`
	FactsHash               string    `gorm:"type:char(64);not null;index" json:"facts_hash"`
	CreatedAt               time.Time `gorm:"not null" json:"created_at"`
}

func (ReportFactSnapshot) TableName() string { return "report_fact_snapshots" }

// ReportLLMCall is an append-only trace row. B2 exposes it read-only; the
// interpretation stage will append attempts without overwriting failures.
type ReportLLMCall struct {
	ID                 uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ReportID           uint64     `gorm:"not null;index;uniqueIndex:idx_report_llm_attempt" json:"report_id"`
	BatchNo            int        `gorm:"not null;uniqueIndex:idx_report_llm_attempt" json:"batch_no"`
	AttemptNo          int        `gorm:"not null;uniqueIndex:idx_report_llm_attempt" json:"attempt_no"`
	ChapterKeys        JSONRaw    `gorm:"type:json;not null" json:"chapter_keys"`
	Provider           string     `gorm:"type:varchar(64);not null" json:"provider"`
	Model              string     `gorm:"type:varchar(128);not null" json:"model"`
	SystemPrompt       string     `gorm:"type:mediumtext;not null" json:"system_prompt"`
	UserPrompt         string     `gorm:"type:mediumtext;not null" json:"user_prompt"`
	InputFacts         JSONRaw    `gorm:"type:json;not null" json:"input_facts"`
	RawOutput          string     `gorm:"type:mediumtext" json:"raw_output,omitempty"`
	ParsedOutput       JSONRaw    `gorm:"type:json" json:"parsed_output,omitempty"`
	SchemaValid        bool       `gorm:"not null;default:false" json:"schema_valid"`
	SchemaErrors       JSONRaw    `gorm:"type:json" json:"schema_errors,omitempty"`
	Parameters         JSONRaw    `gorm:"type:json;not null" json:"parameters"`
	UsageData          JSONRaw    `gorm:"type:json" json:"usage_data"`
	Status             string     `gorm:"type:varchar(32);not null;index" json:"status"`
	ErrorSummary       string     `gorm:"type:varchar(512)" json:"error_summary,omitempty"`
	StartedAt          time.Time  `gorm:"not null" json:"started_at"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	DurationMS         int64      `gorm:"not null;default:0" json:"duration_ms"`
	TraceID            string     `gorm:"type:varchar(64);index" json:"trace_id"`
	FactsSchemaVersion string     `gorm:"type:varchar(64);not null" json:"facts_schema_version"`
	RuleSetVersion     string     `gorm:"type:varchar(64);not null" json:"rule_set_version"`
	PromptVersion      string     `gorm:"type:varchar(64);not null" json:"prompt_version"`
}

func (ReportLLMCall) TableName() string { return "report_llm_calls" }
