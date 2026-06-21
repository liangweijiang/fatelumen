package payment

import (
	"context"
	"strings"
	"testing"
)

func TestMockProvider_CreateCheckout(t *testing.T) {
	m := &mockProvider{secret: "test-secret", baseURL: "http://localhost:8080"}

	res, err := m.CreateCheckout(context.Background(), CheckoutInput{OrderID: 42})
	if err != nil {
		t.Fatalf("CreateCheckout returned error: %v", err)
	}
	if res.SessionID != "mock_42" {
		t.Errorf("expected session_id mock_42, got %q", res.SessionID)
	}
	if !strings.Contains(res.CheckoutURL, "/api/v1/dev/pay/42") {
		t.Errorf("expected checkout_url to contain /api/v1/dev/pay/42, got %q", res.CheckoutURL)
	}
}

func TestMockProvider_BuildAndVerify(t *testing.T) {
	m := &mockProvider{secret: "test-secret", baseURL: "http://localhost:8080"}

	payload, sig, err := m.BuildCompletedEvent(7)
	if err != nil {
		t.Fatalf("BuildCompletedEvent returned error: %v", err)
	}

	we, err := m.VerifyAndParse(payload, sig)
	if err != nil {
		t.Fatalf("VerifyAndParse returned error: %v", err)
	}
	if we.OrderID != 7 {
		t.Errorf("expected order_id 7, got %d", we.OrderID)
	}
	if we.Type != "checkout.session.completed" {
		t.Errorf("expected type checkout.session.completed, got %q", we.Type)
	}
	if we.SessionID != "mock_7" {
		t.Errorf("expected session_id mock_7, got %q", we.SessionID)
	}
	if we.Provider != "mock" {
		t.Errorf("expected provider mock, got %q", we.Provider)
	}
}

func TestMockProvider_VerifyAndParse_TamperedSignature(t *testing.T) {
	m := &mockProvider{secret: "test-secret", baseURL: "http://localhost:8080"}

	payload, _, err := m.BuildCompletedEvent(7)
	if err != nil {
		t.Fatalf("BuildCompletedEvent returned error: %v", err)
	}

	if _, err := m.VerifyAndParse(payload, "deadbeef"); err == nil {
		t.Fatal("expected error for tampered signature, got nil")
	}
}
