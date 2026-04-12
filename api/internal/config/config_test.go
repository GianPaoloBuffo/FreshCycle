package config_test

import (
	"reflect"
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

func TestValidateRequiresOpenAIKeyWhenProviderIsOpenAI(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:                "8080",
		DatabaseURL:         "postgres://example",
		LabelParserProvider: "openai",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error when OPENAI_API_KEY is missing for the openai parser")
	}
}

func TestSplitCSVEnvTrimsValues(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFromMap(map[string]string{
		"SUPABASE_DB_URL":     "postgres://example",
		"API_ALLOWED_ORIGINS": " http://localhost:19006, https://app.example.vercel.app ,, https://freshcycle.example.com ",
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	expected := []string{
		"http://localhost:19006",
		"https://app.example.vercel.app",
		"https://freshcycle.example.com",
	}

	if !reflect.DeepEqual(cfg.AllowedOrigins, expected) {
		t.Fatalf("expected allowed origins %v, got %v", expected, cfg.AllowedOrigins)
	}
}
