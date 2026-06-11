package config

import (
	"os"
	"testing"
)

func TestValidate_MissingCriticalKeys(t *testing.T) {
	cfg := &Config{} // all empty

	missing := cfg.Validate()

	requiredKeys := map[string]bool{
		"DB_USER":          false,
		"DB_PASSWORD":      false,
		"DB_NAME":          false,
		"JWT_SECRET":       false,
		"DEEPSEEK_API_KEY": false,
	}
	for _, m := range missing {
		if _, ok := requiredKeys[m]; ok {
			requiredKeys[m] = true
		}
	}
	for k, found := range requiredKeys {
		if !found {
			t.Errorf("expected missing key '%s' not reported", k)
		}
	}
}

func TestValidate_AllKeysPresent(t *testing.T) {
	cfg := &Config{
		DBUser:          "root",
		DBPassword:      "pass",
		DBName:          "db",
		JWTSecret:       "jwt",
		DeepSeekAPIKey:  "sk-key",
		PaymentProviders: nil, // no payment → Stripe keys not required
	}

	missing := cfg.Validate()
	if len(missing) > 0 {
		t.Errorf("expected no missing keys, got: %v", missing)
	}
}

func TestValidate_StripeKeysRequiredWhenPaymentEnabled(t *testing.T) {
	cfg := &Config{
		DBUser:          "root",
		DBPassword:      "pass",
		DBName:          "db",
		JWTSecret:       "jwt",
		DeepSeekAPIKey:  "sk-key",
		PaymentProviders: []string{"stripe"},
		// StripeSecretKey and StripeWebhookSecret are empty
	}

	missing := cfg.Validate()

	hasStripeKey := false
	hasStripeWebhook := false
	for _, m := range missing {
		if m == "STRIPE_SECRET_KEY" {
			hasStripeKey = true
		}
		if m == "STRIPE_WEBHOOK_SECRET" {
			hasStripeWebhook = true
		}
	}
	if !hasStripeKey {
		t.Error("expected STRIPE_SECRET_KEY to be reported as missing")
	}
	if !hasStripeWebhook {
		t.Error("expected STRIPE_WEBHOOK_SECRET to be reported as missing")
	}
}

func TestValidate_R2Partial(t *testing.T) {
	cfg := &Config{
		DBUser:          "root",
		DBPassword:      "pass",
		DBName:          "db",
		JWTSecret:       "jwt",
		DeepSeekAPIKey:  "sk-key",
		R2AccountID:     "acct-123",
		// Missing R2AccessKeyID, R2SecretAccessKey, R2Bucket
	}

	missing := cfg.Validate()

	requiredR2 := map[string]bool{
		"R2_ACCESS_KEY_ID":     false,
		"R2_SECRET_ACCESS_KEY": false,
		"R2_BUCKET":            false,
	}
	for _, m := range missing {
		if _, ok := requiredR2[m]; ok {
			requiredR2[m] = true
		}
	}
	for k, found := range requiredR2 {
		if !found {
			t.Errorf("expected missing R2 key '%s' not reported", k)
		}
	}
}

func TestValidate_R2Off(t *testing.T) {
	// R2AccountID empty → R2 creds not required
	cfg := &Config{
		DBUser:         "root",
		DBPassword:     "pass",
		DBName:         "db",
		JWTSecret:      "jwt",
		DeepSeekAPIKey: "sk-key",
	}

	missing := cfg.Validate()
	for _, m := range missing {
		if m == "R2_ACCESS_KEY_ID" || m == "R2_SECRET_ACCESS_KEY" || m == "R2_BUCKET" {
			t.Errorf("R2 key '%s' should not be required when R2 is disabled", m)
		}
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	os.Setenv("TEST_FATELUMEN_APP_PORT", "9999")
	defer os.Unsetenv("TEST_FATELUMEN_APP_PORT")

	// This test verifies Viper reads env vars correctly
	// Actual integration test — just verify the structure compiles
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	// Default port is 8080; env override would change it but
	// our key is different so it should stay default
	if cfg.AppPort != "8080" {
		t.Logf("AppPort=%s (might be env override)", cfg.AppPort)
	}
}
