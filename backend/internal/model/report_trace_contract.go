package model

import "time"

const (
	LLMCallStatusPending = "pending"
	LLMCallStatusDone    = "done"
	LLMCallStatusFailed  = "failed"
)

// ReportFactSnapshotContract is the decoded read-only admin representation.
type ReportFactSnapshotContract struct {
	ReportID                uint64                  `json:"report_id"`
	InputSnapshot           ReportInputSnapshot     `json:"input_snapshot"`
	TimeCalculationSnapshot TimeCalculationSnapshot `json:"time_calculation_snapshot"`
	ChartSnapshot           ChartSnapshot           `json:"chart_snapshot"`
	Facts                   InterpretationFacts     `json:"facts"`
	ChartHash               string                  `json:"chart_hash"`
	FactsHash               string                  `json:"facts_hash"`
	CreatedAt               time.Time               `json:"created_at"`
}

// LLMCallParameters contains non-secret provider options used for one call.
type LLMCallParameters struct {
	Temperature float32 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

// LLMUsage records provider usage without storing credentials.
type LLMUsage struct {
	PromptTokens        int   `json:"prompt_tokens,omitempty"`
	CompletionTokens    int   `json:"completion_tokens,omitempty"`
	TotalTokens         int   `json:"total_tokens,omitempty"`
	EstimatedCostMicros int64 `json:"estimated_cost_micros,omitempty"`
}

// ReportLLMCallContract records one attempt. Retries append new records and
// never overwrite failed attempts.
type ReportLLMCallContract struct {
	ID                   uint64            `json:"id"`
	ReportID             uint64            `json:"report_id"`
	BatchNo              int               `json:"batch_no"`
	AttemptNo            int               `json:"attempt_no"`
	ChapterKeys          []string          `json:"chapter_keys"`
	Provider             string            `json:"provider"`
	Model                string            `json:"model"`
	SystemPrompt         string            `json:"system_prompt"`
	UserPrompt           string            `json:"user_prompt"`
	InputFacts           map[string]any    `json:"input_facts"`
	RawOutput            string            `json:"raw_output,omitempty"`
	ParsedOutput         map[string]any    `json:"parsed_output,omitempty"`
	SchemaValid          bool              `json:"schema_valid"`
	SchemaErrors         []string          `json:"schema_errors,omitempty"`
	Parameters           LLMCallParameters `json:"parameters"`
	Usage                LLMUsage          `json:"usage"`
	Status               string            `json:"status"`
	ErrorSummary         string            `json:"error_summary,omitempty"`
	StartedAt            time.Time         `json:"started_at"`
	FinishedAt           *time.Time        `json:"finished_at,omitempty"`
	DurationMilliseconds int64             `json:"duration_milliseconds,omitempty"`
	TraceID              string            `json:"trace_id"`
	FactsSchemaVersion   string            `json:"facts_schema_version"`
	RuleSetVersion       string            `json:"rule_set_version"`
	PromptVersion        string            `json:"prompt_version"`
}

// AdminReportFactsResponse is the read-only admin API contract.
type AdminReportFactsResponse struct {
	ReportID uint64                     `json:"report_id"`
	Snapshot ReportFactSnapshotContract `json:"snapshot"`
}

// AdminReportLLMCallsResponse is the paged admin API contract.
type AdminReportLLMCallsResponse struct {
	ReportID uint64                  `json:"report_id"`
	Items    []ReportLLMCallContract `json:"items"`
	Total    int64                   `json:"total"`
}
