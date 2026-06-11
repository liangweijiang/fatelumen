package model

import (
	"testing"
)

func TestCanTransit_Valid(t *testing.T) {
	tests := []struct {
		from string
		to   string
	}{
		{OrderStatusCreated, OrderStatusPending},
		{OrderStatusCreated, OrderStatusPaid},
		{OrderStatusPending, OrderStatusPaid},
		{OrderStatusPending, OrderStatusFailed},
		{OrderStatusFailed, OrderStatusPending},
		{OrderStatusPaid, OrderStatusRefunded},
	}
	for _, tc := range tests {
		if !CanTransit(tc.from, tc.to) {
			t.Errorf("CanTransit(%s, %s) should be true", tc.from, tc.to)
		}
	}
}

func TestCanTransit_Invalid(t *testing.T) {
	tests := []struct {
		from string
		to   string
	}{
		{OrderStatusCreated, OrderStatusFailed},
		{OrderStatusCreated, OrderStatusRefunded},
		{OrderStatusCreated, OrderStatusCreated},
		{OrderStatusPending, OrderStatusCreated},
		{OrderStatusPending, OrderStatusPending},
		{OrderStatusPending, OrderStatusRefunded},
		{OrderStatusFailed, OrderStatusPaid},
		{OrderStatusFailed, OrderStatusFailed},
		{OrderStatusFailed, OrderStatusRefunded},
		{OrderStatusPaid, OrderStatusCreated},
		{OrderStatusPaid, OrderStatusPending},
		{OrderStatusPaid, OrderStatusFailed},
		{OrderStatusPaid, OrderStatusPaid},
		{OrderStatusRefunded, OrderStatusCreated},
		{OrderStatusRefunded, OrderStatusPending},
		{OrderStatusRefunded, OrderStatusPaid},
		{OrderStatusRefunded, OrderStatusFailed},
		{OrderStatusRefunded, OrderStatusRefunded},
		{"", OrderStatusPending},
		{OrderStatusCreated, ""},
		{"unknown", "anything"},
	}
	for _, tc := range tests {
		if CanTransit(tc.from, tc.to) {
			t.Errorf("CanTransit(%s, %s) should be false", tc.from, tc.to)
		}
	}
}

func TestOrder_Transit_Success(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{"created to pending", OrderStatusCreated, OrderStatusPending},
		{"created to paid", OrderStatusCreated, OrderStatusPaid},
		{"pending to paid", OrderStatusPending, OrderStatusPaid},
		{"pending to failed", OrderStatusPending, OrderStatusFailed},
		{"failed to pending (retry)", OrderStatusFailed, OrderStatusPending},
		{"paid to refunded", OrderStatusPaid, OrderStatusRefunded},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := &Order{Status: tc.from}
			if err := o.Transit(tc.to); err != nil {
				t.Fatalf("expected success, got: %v", err)
			}
			if o.Status != tc.to {
				t.Fatalf("expected status %s, got %s", tc.to, o.Status)
			}
		})
	}
}

func TestOrder_Transit_Illegal_StatusUnchanged(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{"created to refunded", OrderStatusCreated, OrderStatusRefunded},
		{"paid to created", OrderStatusPaid, OrderStatusCreated},
		{"paid to pending", OrderStatusPaid, OrderStatusPending},
		{"paid to paid", OrderStatusPaid, OrderStatusPaid},
		{"refunded to created", OrderStatusRefunded, OrderStatusCreated},
		{"refunded to pending", OrderStatusRefunded, OrderStatusPending},
		{"refunded to paid", OrderStatusRefunded, OrderStatusPaid},
		{"refunded to failed", OrderStatusRefunded, OrderStatusFailed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := &Order{Status: tc.from}
			err := o.Transit(tc.to)
			if err == nil {
				t.Fatal("expected error for illegal transition")
			}
			if o.Status != tc.from {
				t.Fatalf("status should not change on illegal transition, expected %s, got %s", tc.from, o.Status)
			}
		})
	}
}
