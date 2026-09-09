package service

import (
	"encoding/json"
	"testing"

	"fatelumen/backend/internal/facts"
	"fatelumen/backend/internal/model"
	hashutil "fatelumen/backend/internal/pkg/hash"
)

func TestVerifyFullReportIntegrityMatchedAndTampered(t *testing.T) {
	frozenFacts := model.InterpretationFacts{}
	factsHash, err := facts.CalculateHash(frozenFacts)
	if err != nil {
		t.Fatal(err)
	}
	plans := []map[string]any{{"no": 1, "key": "destiny_depth", "prompt_hash": "prompt"}}
	runtime := FullReportRuntimeConfig{ChapterConcurrency: 3}
	executionHash, err := hashutil.CanonicalJSONSHA256(map[string]any{"facts_hash": factsHash, "plans": plans, "runtime": runtime})
	if err != nil {
		t.Fatal(err)
	}
	content := model.ReportContent{}
	contentHash, err := hashutil.CanonicalJSONSHA256(content)
	if err != nil {
		t.Fatal(err)
	}
	factsRaw, _ := json.Marshal(frozenFacts)
	plansRaw, _ := json.Marshal(plans)
	runtimeRaw, _ := json.Marshal(runtime)
	report := &model.FullReport{FactsHash: factsHash, ExecutionHash: executionHash, ContentHash: contentHash}
	trace := &FullReportIntegrityTrace{FactsHash: factsHash, ExecutionHash: executionHash, FactsSnapshot: factsRaw, ChapterPlanSnapshot: plansRaw, RuntimeConfigSnapshot: runtimeRaw}
	result := &model.FullReportResult{Content: content, ContentHash: contentHash}

	verified, err := VerifyFullReportIntegrity(report, trace, result)
	if err != nil {
		t.Fatal(err)
	}
	if !verified.Passed || verified.Facts.Status != IntegrityStatusMatched || verified.Execution.Status != IntegrityStatusMatched || verified.Content.Status != IntegrityStatusMatched {
		t.Fatalf("expected all hashes to match: %+v", verified)
	}

	report.ContentHash = "tampered"
	verified, err = VerifyFullReportIntegrity(report, trace, result)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Passed || verified.Content.Status != IntegrityStatusMismatched {
		t.Fatalf("expected tampered content hash to fail: %+v", verified)
	}
}

func TestVerifyFullReportIntegrityMissingSnapshots(t *testing.T) {
	verified, err := VerifyFullReportIntegrity(&model.FullReport{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Passed || verified.Facts.Status != IntegrityStatusMissing || verified.Execution.Status != IntegrityStatusMissing || verified.Content.Status != IntegrityStatusMissing {
		t.Fatalf("expected missing statuses: %+v", verified)
	}
}
