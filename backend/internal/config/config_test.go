package config

import (
	"testing"
)

func TestIsAdminEmail_Hit(t *testing.T) {
	cfg := &Config{AdminEmails: []string{"admin@example.com", "root@test.com"}}

	if !cfg.IsAdminEmail("admin@example.com") {
		t.Error("expected hit for exact match")
	}
	if !cfg.IsAdminEmail("root@test.com") {
		t.Error("expected hit for second admin")
	}
}

func TestIsAdminEmail_Miss(t *testing.T) {
	cfg := &Config{AdminEmails: []string{"admin@example.com"}}

	if cfg.IsAdminEmail("user@example.com") {
		t.Error("expected miss for non-admin")
	}
	if cfg.IsAdminEmail("") {
		t.Error("expected miss for empty")
	}
}

func TestIsAdminEmail_CaseInsensitive(t *testing.T) {
	cfg := &Config{AdminEmails: []string{"Admin@Example.Com"}}

	if !cfg.IsAdminEmail("admin@example.com") {
		t.Error("expected case insensitive match (lower)")
	}
	if !cfg.IsAdminEmail("ADMIN@EXAMPLE.COM") {
		t.Error("expected case insensitive match (upper)")
	}
}

func TestIsAdminEmail_Whitespace(t *testing.T) {
	// Config values are already trimmed by splitEnv on load
	cfg := &Config{AdminEmails: []string{"admin@example.com"}}

	if !cfg.IsAdminEmail(" admin@example.com ") {
		t.Error("expected match with input having spaces")
	}
}

func TestIsAdminEmail_EmptyList(t *testing.T) {
	cfg := &Config{}

	if cfg.IsAdminEmail("admin@example.com") {
		t.Error("expected miss for empty admin list")
	}
}
