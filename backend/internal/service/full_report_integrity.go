package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"fatelumen/backend/internal/facts"
	"fatelumen/backend/internal/model"
	hashutil "fatelumen/backend/internal/pkg/hash"
)

const (
	IntegrityStatusMatched    = "matched"
	IntegrityStatusMismatched = "mismatched"
	IntegrityStatusMissing    = "missing"
)

type FullReportHashCheck struct {
	Status     string `json:"status"`
	StoredHash string `json:"stored_hash,omitempty"`
	ActualHash string `json:"actual_hash,omitempty"`
}

type FullReportIntegrity struct {
	Passed    bool                `json:"passed"`
	Facts     FullReportHashCheck `json:"facts"`
	Execution FullReportHashCheck `json:"execution"`
	Content   FullReportHashCheck `json:"content"`
}

// VerifyFullReportIntegrity recalculates the three immutable report hashes.
// Missing snapshots are reported as data state, while malformed stored JSON is
// returned as an error because it prevents a trustworthy calculation.
func VerifyFullReportIntegrity(report *model.FullReport, trace *FullReportIntegrityTrace, result *model.FullReportResult) (*FullReportIntegrity, error) {
	checks := &FullReportIntegrity{
		Facts: missingHashCheck(report.FactsHash), Execution: missingHashCheck(report.ExecutionHash), Content: missingHashCheck(report.ContentHash),
	}
	if trace != nil {
		var frozenFacts model.InterpretationFacts
		if err := json.Unmarshal(trace.FactsSnapshot, &frozenFacts); err != nil {
			return nil, fmt.Errorf("decode facts snapshot: %w", err)
		}
		actual, err := facts.CalculateHash(frozenFacts)
		if err != nil {
			return nil, fmt.Errorf("hash facts snapshot: %w", err)
		}
		checks.Facts = compareHash(report.FactsHash, actual)
		if report.FactsHash != trace.FactsHash {
			checks.Facts.Status = IntegrityStatusMismatched
		}

		var plans []map[string]any
		if err := json.Unmarshal(trace.ChapterPlanSnapshot, &plans); err != nil {
			return nil, fmt.Errorf("decode chapter plan snapshot: %w", err)
		}
		var runtime FullReportRuntimeConfig
		if err := json.Unmarshal(trace.RuntimeConfigSnapshot, &runtime); err != nil {
			return nil, fmt.Errorf("decode runtime config snapshot: %w", err)
		}
		actual, err = hashutil.CanonicalJSONSHA256(map[string]any{"facts_hash": trace.FactsHash, "plans": plans, "runtime": runtime})
		if err != nil {
			return nil, fmt.Errorf("hash execution snapshot: %w", err)
		}
		checks.Execution = compareHash(report.ExecutionHash, actual)
		if report.ExecutionHash != trace.ExecutionHash {
			checks.Execution.Status = IntegrityStatusMismatched
		}
	}
	if result != nil {
		actual, err := hashutil.CanonicalJSONSHA256(result.Content)
		if err != nil {
			return nil, fmt.Errorf("hash report content: %w", err)
		}
		checks.Content = compareHash(report.ContentHash, actual)
		if report.ContentHash != result.ContentHash {
			checks.Content.Status = IntegrityStatusMismatched
		}
	}
	checks.Passed = checks.Facts.Status == IntegrityStatusMatched && checks.Execution.Status == IntegrityStatusMatched && checks.Content.Status == IntegrityStatusMatched
	return checks, nil
}

// repositoryIntegrityTrace is deliberately small: the handler maps only the
// frozen fields needed by verification, keeping repository concerns outside the
// pure comparison logic.
type FullReportIntegrityTrace struct {
	FactsHash             string
	ExecutionHash         string
	FactsSnapshot         model.JSONRaw
	ChapterPlanSnapshot   model.JSONRaw
	RuntimeConfigSnapshot model.JSONRaw
}

func NewIntegrityTrace(snapshot model.FullReportExecutionSnapshot, payload model.FullReportExecutionPayload) *FullReportIntegrityTrace {
	return &FullReportIntegrityTrace{FactsHash: snapshot.FactsHash, ExecutionHash: snapshot.ExecutionHash, FactsSnapshot: payload.FactsSnapshot, ChapterPlanSnapshot: payload.ChapterPlanSnapshot, RuntimeConfigSnapshot: payload.RuntimeConfigSnapshot}
}

func missingHashCheck(stored string) FullReportHashCheck {
	return FullReportHashCheck{Status: IntegrityStatusMissing, StoredHash: strings.TrimSpace(stored)}
}

func compareHash(stored, actual string) FullReportHashCheck {
	status := IntegrityStatusMismatched
	if stored != "" && stored == actual {
		status = IntegrityStatusMatched
	}
	return FullReportHashCheck{Status: status, StoredHash: stored, ActualHash: actual}
}
