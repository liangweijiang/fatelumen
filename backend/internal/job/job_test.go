package job

import (
	"testing"
)

func TestCanTransit_Valid(t *testing.T) {
	tests := []struct {
		from Status
		to   Status
	}{
		{StatusPending, StatusProcessing},
		{StatusProcessing, StatusDone},
		{StatusProcessing, StatusFailed},
		{StatusFailed, StatusPending}, // retry
	}
	for _, tc := range tests {
		if !CanTransit(tc.from, tc.to) {
			t.Errorf("CanTransit(%s, %s) should be true", tc.from, tc.to)
		}
	}
}

func TestCanTransit_Invalid(t *testing.T) {
	tests := []struct {
		from Status
		to   Status
	}{
		{StatusPending, StatusDone},
		{StatusPending, StatusFailed},
		{StatusProcessing, StatusPending},
		{StatusDone, StatusPending},
		{StatusDone, StatusProcessing},
		{StatusDone, StatusFailed},
		{StatusFailed, StatusDone},
		{StatusFailed, StatusProcessing},
		{"", StatusPending},
		{StatusPending, ""},
		{"unknown", "anything"},
	}
	for _, tc := range tests {
		if CanTransit(tc.from, tc.to) {
			t.Errorf("CanTransit(%s, %s) should be false", tc.from, tc.to)
		}
	}
}

func TestJob_Transit_Success(t *testing.T) {
	j := &Job{Status: StatusPending}
	if err := j.Transit(StatusProcessing); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if j.Status != StatusProcessing {
		t.Fatalf("expected processing, got %s", j.Status)
	}
}

func TestJob_Transit_Illegal(t *testing.T) {
	j := &Job{Status: StatusDone}
	if err := j.Transit(StatusProcessing); err == nil {
		t.Fatal("expected error for illegal transition")
	}
}

func TestJob_TransitToFailed_WithAttempt(t *testing.T) {
	j := &Job{Status: StatusProcessing, Attempts: 1}

	if err := j.TransitToFailed(); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if j.Attempts != 2 {
		t.Fatalf("expected attempts 2, got %d", j.Attempts)
	}
	if j.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", j.Status)
	}
}

func TestJob_TransitToRetry(t *testing.T) {
	j := &Job{Status: StatusFailed, Attempts: 2}

	if err := j.TransitToRetry(); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if j.Status != StatusPending {
		t.Fatalf("expected pending after retry, got %s", j.Status)
	}
	if j.Attempts != 2 {
		t.Fatalf("attempts should not change on retry transit, got %d", j.Attempts)
	}
}

func TestNewJobID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NewJobID()
		if ids[id] {
			t.Fatalf("duplicate job ID: %s", id)
		}
		ids[id] = true
	}
}

func TestNewJobID_Length(t *testing.T) {
	id := NewJobID()
	if len(id) != 16 {
		t.Fatalf("expected 16 hex chars, got %d: %s", len(id), id)
	}
}
