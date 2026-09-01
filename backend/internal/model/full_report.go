package model

import "time"

const (
	FullReportStatusPending      = "pending"
	FullReportStatusPreflighting = "preflighting"
	FullReportStatusGenerating   = "generating"
	FullReportStatusAssembling   = "assembling"
	FullReportStatusRendering    = "rendering"
	FullReportStatusCompleted    = "completed"
	FullReportStatusFailed       = "failed"
	FullReportStatusDeleting     = "deleting"
)

const (
	FullReportChapterStatusPending    = "pending"
	FullReportChapterStatusRunning    = "running"
	FullReportChapterStatusSucceeded  = "succeeded"
	FullReportChapterStatusFailed     = "failed"
	FullReportAttemptStatusRunning    = "running"
	FullReportAttemptStatusSucceeded  = "succeeded"
	FullReportAttemptStatusRejected   = "rejected"
	FullReportAttemptStatusFailed     = "failed"
	FullReportValidationStatusPending = "pending"
	FullReportValidationStatusPassed  = "passed"
	FullReportValidationStatusFailed  = "failed"
)

func IsFullReportTerminalStatus(status string) bool {
	return status == FullReportStatusCompleted || status == FullReportStatusFailed
}

// FullReport contains only searchable report metadata. Large immutable data is
// stored in the one-to-one payload tables below.
type FullReport struct {
	ID                 uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	PublicID           string     `gorm:"type:char(26);not null;uniqueIndex" json:"public_id"`
	UserID             uint64     `gorm:"not null;index:idx_full_reports_user_created,priority:1" json:"user_id"`
	ProfileID          *uint64    `gorm:"index" json:"profile_id,omitempty"`
	ChartID            *uint64    `gorm:"index" json:"chart_id,omitempty"`
	OrderID            *uint64    `gorm:"index" json:"order_id,omitempty"`
	SourceReportID     *uint64    `gorm:"index" json:"source_report_id,omitempty"`
	Locale             string     `gorm:"type:varchar(8);not null" json:"locale"`
	PayMethod          string     `gorm:"type:varchar(16);not null" json:"pay_method"`
	Paid               bool       `gorm:"not null;default:false;index" json:"paid"`
	Status             string     `gorm:"type:varchar(24);not null;index:idx_full_reports_status_created,priority:1" json:"status"`
	CurrentStage       string     `gorm:"type:varchar(32);not null" json:"current_stage"`
	ChapterTotal       uint8      `gorm:"not null;default:10" json:"chapter_total"`
	ChapterSucceeded   uint8      `gorm:"not null;default:0" json:"chapter_succeeded"`
	ChapterFailed      uint8      `gorm:"not null;default:0" json:"chapter_failed"`
	ProviderChainKey   string     `gorm:"type:varchar(64);not null" json:"provider_chain_key"`
	ChapterConcurrency uint8      `gorm:"not null" json:"chapter_concurrency"`
	FactsHash          string     `gorm:"type:char(64);not null;index:idx_full_reports_facts_created,priority:1" json:"facts_hash"`
	ExecutionHash      string     `gorm:"type:char(64)" json:"execution_hash,omitempty"`
	ContentHash        string     `gorm:"type:char(64)" json:"content_hash,omitempty"`
	ErrorCode          string     `gorm:"type:varchar(64)" json:"error_code,omitempty"`
	ErrorSummary       string     `gorm:"type:varchar(512)" json:"error_summary,omitempty"`
	RetentionPolicy    string     `gorm:"type:varchar(32);not null" json:"retention_policy"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	ExpiresAt          time.Time  `gorm:"not null;index:idx_full_reports_expiry,priority:1" json:"expires_at"`
	CreatedAt          time.Time  `gorm:"not null;index:idx_full_reports_user_created,priority:2;index:idx_full_reports_status_created,priority:2;index:idx_full_reports_facts_created,priority:2" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"not null" json:"updated_at"`
}

func (FullReport) TableName() string { return "full_reports" }

type FullReportExecutionSnapshot struct {
	ID                      uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ReportID                uint64    `gorm:"not null;uniqueIndex" json:"report_id"`
	ChartHash               string    `gorm:"type:char(64);not null" json:"chart_hash"`
	FactsHash               string    `gorm:"type:char(64);not null" json:"facts_hash"`
	ExecutionHash           string    `gorm:"type:char(64);not null" json:"execution_hash"`
	InputSchemaVersion      string    `gorm:"type:varchar(64);not null" json:"input_schema_version"`
	ChartSchemaVersion      string    `gorm:"type:varchar(64);not null" json:"chart_schema_version"`
	FactsSchemaVersion      string    `gorm:"type:varchar(64);not null" json:"facts_schema_version"`
	RuleSetVersion          string    `gorm:"type:varchar(64);not null" json:"rule_set_version"`
	PromptVersion           string    `gorm:"type:varchar(64);not null" json:"prompt_version"`
	DictionaryVersion       string    `gorm:"type:varchar(64);not null" json:"dictionary_version"`
	RuntimePolicyVersion    string    `gorm:"type:varchar(64);not null" json:"runtime_policy_version"`
	LocationDatabaseVersion string    `gorm:"type:varchar(64);not null" json:"location_database_version"`
	TimezoneDatabaseVersion string    `gorm:"type:varchar(64);not null" json:"timezone_database_version"`
	SolarAlgorithmVersion   string    `gorm:"type:varchar(64);not null" json:"solar_algorithm_version"`
	LunarGoVersion          string    `gorm:"type:varchar(64);not null" json:"lunar_go_version"`
	FrozenAt                time.Time `gorm:"not null" json:"frozen_at"`
	CreatedAt               time.Time `gorm:"not null" json:"created_at"`
}

func (FullReportExecutionSnapshot) TableName() string { return "full_report_execution_snapshots" }

type FullReportExecutionPayload struct {
	ID                      uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	SnapshotID              uint64    `gorm:"not null;uniqueIndex" json:"snapshot_id"`
	InputSnapshot           JSONRaw   `gorm:"type:json;not null" json:"input_snapshot"`
	TimeCalculationSnapshot JSONRaw   `gorm:"type:json;not null" json:"time_calculation_snapshot"`
	ChartSnapshot           JSONRaw   `gorm:"type:json;not null" json:"chart_snapshot"`
	FactsSnapshot           JSONRaw   `gorm:"type:json;not null" json:"facts_snapshot"`
	PreflightResult         JSONRaw   `gorm:"type:json;not null" json:"preflight_result"`
	ChapterPlanSnapshot     JSONRaw   `gorm:"type:json;not null" json:"chapter_plan_snapshot"`
	RuntimeConfigSnapshot   JSONRaw   `gorm:"type:json;not null" json:"runtime_config_snapshot"`
	CreatedAt               time.Time `gorm:"not null" json:"created_at"`
}

func (FullReportExecutionPayload) TableName() string { return "full_report_execution_payloads" }

type FullReportChapter struct {
	ID                uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ReportID          uint64     `gorm:"not null;uniqueIndex:uk_full_report_chapter_no,priority:1;uniqueIndex:uk_full_report_chapter_key,priority:1" json:"report_id"`
	ChapterNo         uint8      `gorm:"not null;uniqueIndex:uk_full_report_chapter_no,priority:2" json:"chapter_no"`
	ChapterKey        string     `gorm:"type:varchar(64);not null;uniqueIndex:uk_full_report_chapter_key,priority:2" json:"chapter_key"`
	Title             string     `gorm:"type:varchar(128);not null" json:"title"`
	Status            string     `gorm:"type:varchar(24);not null;index" json:"status"`
	AttemptCount      uint16     `gorm:"not null;default:0" json:"attempt_count"`
	SelectedAttemptID *uint64    `gorm:"index" json:"selected_attempt_id,omitempty"`
	PromptHash        string     `gorm:"type:char(64);not null" json:"prompt_hash"`
	OutputHash        string     `gorm:"type:char(64)" json:"output_hash,omitempty"`
	SchemaValid       bool       `gorm:"not null;default:false" json:"schema_valid"`
	ValidationStatus  string     `gorm:"type:varchar(24);not null" json:"validation_status"`
	ErrorCode         string     `gorm:"type:varchar(64)" json:"error_code,omitempty"`
	ErrorSummary      string     `gorm:"type:varchar(512)" json:"error_summary,omitempty"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CreatedAt         time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"not null" json:"updated_at"`
}

func (FullReportChapter) TableName() string { return "full_report_chapters" }

type FullReportChapterPayload struct {
	ID                  uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChapterID           uint64    `gorm:"not null;uniqueIndex" json:"chapter_id"`
	SemanticDigest      string    `gorm:"type:mediumtext;not null" json:"semantic_digest"`
	LanguageInstruction string    `gorm:"type:mediumtext;not null" json:"language_instruction"`
	TerminologySnapshot JSONRaw   `gorm:"type:json;not null" json:"terminology_snapshot"`
	FinalPrompt         string    `gorm:"type:mediumtext;not null" json:"final_prompt"`
	OutputSchema        JSONRaw   `gorm:"type:json;not null" json:"output_schema"`
	FinalRawOutput      string    `gorm:"type:mediumtext" json:"final_raw_output,omitempty"`
	FinalParsedOutput   JSONRaw   `gorm:"type:json" json:"final_parsed_output,omitempty"`
	ValidationResult    JSONRaw   `gorm:"type:json" json:"validation_result,omitempty"`
	CreatedAt           time.Time `gorm:"not null" json:"created_at"`
}

func (FullReportChapterPayload) TableName() string { return "full_report_chapter_payloads" }

type FullReportAttempt struct {
	ID               uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ReportID         uint64     `gorm:"not null;index:idx_full_report_attempt_report_chapter,priority:1" json:"report_id"`
	ChapterID        uint64     `gorm:"not null;uniqueIndex:uk_full_report_attempt,priority:1;index:idx_full_report_attempt_report_chapter,priority:2" json:"chapter_id"`
	AttemptNo        uint16     `gorm:"not null;uniqueIndex:uk_full_report_attempt,priority:2" json:"attempt_no"`
	RouteNo          uint8      `gorm:"not null" json:"route_no"`
	Provider         string     `gorm:"type:varchar(64);not null" json:"provider"`
	Model            string     `gorm:"type:varchar(128);not null" json:"model"`
	Status           string     `gorm:"type:varchar(24);not null;index:idx_full_report_attempt_status_started,priority:1" json:"status"`
	SchemaValid      bool       `gorm:"not null;default:false" json:"schema_valid"`
	ValidationStatus string     `gorm:"type:varchar(24);not null" json:"validation_status"`
	ErrorCode        string     `gorm:"type:varchar(64)" json:"error_code,omitempty"`
	ErrorSummary     string     `gorm:"type:varchar(512)" json:"error_summary,omitempty"`
	PromptHash       string     `gorm:"type:char(64);not null" json:"prompt_hash"`
	OutputHash       string     `gorm:"type:char(64)" json:"output_hash,omitempty"`
	PromptTokens     int        `gorm:"not null;default:0" json:"prompt_tokens"`
	CompletionTokens int        `gorm:"not null;default:0" json:"completion_tokens"`
	DurationMS       int64      `gorm:"not null;default:0" json:"duration_ms"`
	TraceID          string     `gorm:"type:varchar(64);not null;index" json:"trace_id"`
	StartedAt        time.Time  `gorm:"not null;index:idx_full_report_attempt_status_started,priority:2" json:"started_at"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
	CreatedAt        time.Time  `gorm:"not null" json:"created_at"`
}

func (FullReportAttempt) TableName() string { return "full_report_attempts" }

type FullReportAttemptPayload struct {
	ID                uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AttemptID         uint64    `gorm:"not null;uniqueIndex" json:"attempt_id"`
	RequestParameters JSONRaw   `gorm:"type:json;not null" json:"request_parameters"`
	RequestPrompt     string    `gorm:"type:mediumtext;not null" json:"request_prompt"`
	RawOutput         string    `gorm:"type:mediumtext" json:"raw_output,omitempty"`
	ParsedOutput      JSONRaw   `gorm:"type:json" json:"parsed_output,omitempty"`
	SchemaErrors      JSONRaw   `gorm:"type:json" json:"schema_errors,omitempty"`
	ValidationResult  JSONRaw   `gorm:"type:json" json:"validation_result,omitempty"`
	CreatedAt         time.Time `gorm:"not null" json:"created_at"`
}

func (FullReportAttemptPayload) TableName() string { return "full_report_attempt_payloads" }

type FullReportResult struct {
	ID            uint64        `gorm:"primaryKey;autoIncrement" json:"id"`
	ReportID      uint64        `gorm:"not null;uniqueIndex" json:"report_id"`
	Locale        string        `gorm:"type:varchar(8);not null" json:"locale"`
	Content       ReportContent `gorm:"column:content_json;type:json;not null" json:"content"`
	ContentHash   string        `gorm:"type:char(64);not null" json:"content_hash"`
	RenderVersion string        `gorm:"type:varchar(64);not null" json:"render_version"`
	PDFStorageKey string        `gorm:"type:varchar(512)" json:"pdf_storage_key,omitempty"`
	PDFURL        string        `gorm:"type:varchar(512)" json:"pdf_url,omitempty"`
	PDFHash       string        `gorm:"type:char(64)" json:"pdf_hash,omitempty"`
	CreatedAt     time.Time     `gorm:"not null" json:"created_at"`
}

func (FullReportResult) TableName() string { return "full_report_results" }
