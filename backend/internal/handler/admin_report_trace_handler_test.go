package handler

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"fatelumen/backend/internal/repository"
)

func TestReportCursorRoundTrip(t *testing.T) {
	wantTime := time.Date(2026, time.September, 7, 5, 30, 12, 987654321, time.UTC)
	wantID := uint64(42)

	encoded := encodeReportCursor(wantTime, wantID)
	decoded, err := decodeReportCursor(encoded)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if !decoded.CreatedAt.Equal(wantTime) {
		t.Fatalf("created_at mismatch: got %s want %s", decoded.CreatedAt, wantTime)
	}
	if decoded.ID != wantID {
		t.Fatalf("id mismatch: got %d want %d", decoded.ID, wantID)
	}
}

func TestChapterTraceUsesStableLowercaseContract(t *testing.T) {
	raw, err := json.Marshal(repository.FullReportChapterWithPayload{})
	if err != nil {
		t.Fatal(err)
	}
	value := string(raw)
	if !strings.Contains(value, `"chapter"`) || !strings.Contains(value, `"payload"`) || strings.Contains(value, `"Chapter"`) {
		t.Fatalf("unexpected chapter trace json: %s", value)
	}
}

func TestReportCursorRejectsInvalidValues(t *testing.T) {
	for _, raw := range []string{"not-base64", "aW52YWxpZA", "MjAyNi0wOS0wN1QwNTozMDoxMloxfDA"} {
		if _, err := decodeReportCursor(raw); err == nil {
			t.Fatalf("expected invalid cursor %q to fail", raw)
		}
	}
}
