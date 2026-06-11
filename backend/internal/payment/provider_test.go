package payment

import (
	"testing"
)

func TestParseOrderID_Valid(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]string
		want     uint64
	}{
		{"normal", map[string]string{"order_id": "42"}, 42},
		{"zero", map[string]string{"order_id": "0"}, 0},
		{"large", map[string]string{"order_id": "18446744073709551615"}, 18446744073709551615},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseOrderID(tc.metadata)
			if got != tc.want {
				t.Errorf("ParseOrderID(%v) = %d, want %d", tc.metadata, got, tc.want)
			}
		})
	}
}

func TestParseOrderID_Missing(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]string
	}{
		{"nil", nil},
		{"empty", map[string]string{}},
		{"absent key", map[string]string{"other": "val"}},
		{"empty value", map[string]string{"order_id": ""}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseOrderID(tc.metadata)
			if got != 0 {
				t.Errorf("ParseOrderID(%v) = %d, want 0", tc.metadata, got)
			}
		})
	}
}

func TestParseOrderID_NonNumeric(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]string
	}{
		{"alphabetic", map[string]string{"order_id": "abc"}},
		{"alphanumeric", map[string]string{"order_id": "12ab"}},
		{"negative", map[string]string{"order_id": "-1"}},
		{"float", map[string]string{"order_id": "3.14"}},
		{"spaces", map[string]string{"order_id": " 42 "}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseOrderID(tc.metadata)
			if got != 0 {
				t.Errorf("ParseOrderID(%v) = %d, want 0", tc.metadata, got)
			}
		})
	}
}

func TestStripeProvider_VerifyAndParse_InvalidSignature(t *testing.T) {
	prov := NewStripeProvider("sk_test_fake", "whsec_fake_secret")
	payload := []byte(`{"type":"checkout.session.completed"}`)
	badSig := "t=9999999999,v1=0000000000000000000000000000000000000000000000000000000000000000"

	_, err := prov.VerifyAndParse(payload, badSig)
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
}

func TestStripeProvider_VerifyAndParse_EmptySignature(t *testing.T) {
	prov := NewStripeProvider("sk_test_fake", "whsec_fake_secret")
	payload := []byte(`{}`)

	_, err := prov.VerifyAndParse(payload, "")
	if err == nil {
		t.Fatal("expected error for empty signature, got nil")
	}
}
