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

func TestLLMCallContractKeepsAttemptAndVersions(t *testing.T) {
	call := ReportLLMCallContract{
		ReportID:           8,
		BatchNo:            2,
		AttemptNo:          3,
		FactsSchemaVersion: FactsSchemaVersion,
		RuleSetVersion:     "rules-v1",
		PromptVersion:      "full-v1",
	}
	if call.AttemptNo != 3 || call.FactsSchemaVersion == "" || call.RuleSetVersion == "" || call.PromptVersion == "" {
		t.Fatal("LLM call trace must pin attempt and all contract versions")
	}
}
