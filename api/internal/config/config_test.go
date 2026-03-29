package config_test

import (
	"testing"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/config"
)

func TestValidateRequiresDatabaseURL(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:        "8080",
		DatabaseURL: "",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error when SUPABASE_DB_URL is missing")
	}
}

func TestAddressUsesPort(t *testing.T) {
	t.Parallel()

	cfg := config.Config{Port: "8080", DatabaseURL: "postgres://example"}

	if got := cfg.Address(); got != ":8080" {
		t.Fatalf("expected address :8080, got %s", got)
	}
}
