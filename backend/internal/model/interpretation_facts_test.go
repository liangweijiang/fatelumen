package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestInterpretationFactsContractHasNoCredentialFields(t *testing.T) {
	payload, err := json.Marshal(InterpretationFacts{})
	if err != nil {
		t.Fatalf("marshal facts contract: %v", err)
	}
	lower := strings.ToLower(string(payload))
	for _, forbidden := range []string{"password", "jwt", "api_key", "email", "phone", "token"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("facts contract contains forbidden privacy field %q", forbidden)
		}
	}
}

func TestFullReportTraceKeepsAttemptAndVersions(t *testing.T) {
	attempt := FullReportAttempt{ReportID: 8, ChapterID: 2, AttemptNo: 3, RouteNo: 1, Provider: "deepseek", Model: "deepseek-chat"}
	snapshot := FullReportExecutionSnapshot{FactsSchemaVersion: FactsSchemaVersion, RuleSetVersion: "rules-v1", PromptVersion: "full-v1"}
	if attempt.AttemptNo != 3 || attempt.RouteNo != 1 || attempt.Provider == "" || attempt.Model == "" {
		t.Fatal("full report attempt must pin attempt number and model route")
	}
	if snapshot.FactsSchemaVersion == "" || snapshot.RuleSetVersion == "" || snapshot.PromptVersion == "" {
		t.Fatal("full report execution snapshot must pin all contract versions")
	}
}
