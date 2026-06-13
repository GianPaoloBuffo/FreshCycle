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

func TestLoadDefaultsOCRAndGeminiConfig(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFromMap(map[string]string{
		"SUPABASE_DB_URL": "postgres://example",
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.OCRTesseractPath != "tesseract" {
		t.Fatalf("expected default tesseract path, got %q", cfg.OCRTesseractPath)
	}
	if cfg.OCRLanguages != "eng+spa" {
		t.Fatalf("expected default OCR languages, got %q", cfg.OCRLanguages)
	}
	if cfg.LabelParserFallbackProvider != "gemini" {
		t.Fatalf("expected default fallback provider, got %q", cfg.LabelParserFallbackProvider)
	}
	if cfg.GeminiModel != "gemini-3.1-flash-lite" {
		t.Fatalf("expected default Gemini model, got %q", cfg.GeminiModel)
	}
	if cfg.GeminiBaseURL != "https://generativelanguage.googleapis.com/v1beta" {
		t.Fatalf("expected default Gemini base URL, got %q", cfg.GeminiBaseURL)
	}
	if !cfg.ScanCacheEnabled {
		t.Fatal("expected scan cache to default enabled")
	}
	if cfg.ScanCacheMaxEntries != 512 {
		t.Fatalf("expected default scan cache max entries, got %d", cfg.ScanCacheMaxEntries)
	}
	if !cfg.SymbolDetectorEnabled {
		t.Fatal("expected symbol detector to default enabled")
	}
	if !cfg.ScanTelemetryEnabled {
		t.Fatal("expected scan telemetry to default enabled")
	}
	if cfg.ScanTelemetryEnvironment != "production" {
		t.Fatalf("expected production scan telemetry environment, got %q", cfg.ScanTelemetryEnvironment)
	}
}

func TestLoadScanPipelineConfigOverrides(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFromMap(map[string]string{
		"SUPABASE_DB_URL":            "postgres://example",
		"SCAN_CACHE_ENABLED":         "false",
		"SCAN_CACHE_TTL":             "2h",
		"SCAN_CACHE_MAX_ENTRIES":     "128",
		"SYMBOL_DETECTOR_ENABLED":    "false",
		"SCAN_TELEMETRY_ENABLED":     "false",
		"SCAN_TELEMETRY_ENVIRONMENT": "debug",
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.ScanCacheEnabled {
		t.Fatal("expected scan cache override to disable cache")
	}
	if cfg.ScanCacheTTL.String() != "2h0m0s" {
		t.Fatalf("expected scan cache TTL override, got %s", cfg.ScanCacheTTL)
	}
	if cfg.ScanCacheMaxEntries != 128 {
		t.Fatalf("expected scan cache max entries override, got %d", cfg.ScanCacheMaxEntries)
	}
	if cfg.SymbolDetectorEnabled {
		t.Fatal("expected symbol detector override to disable detector")
	}
	if cfg.ScanTelemetryEnabled {
		t.Fatal("expected scan telemetry override to disable telemetry")
	}
	if cfg.ScanTelemetryEnvironment != "debug" {
		t.Fatalf("expected debug scan telemetry environment, got %q", cfg.ScanTelemetryEnvironment)
	}
}

func TestValidateRejectsUnsupportedScanTelemetryEnvironment(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:                     "8080",
		DatabaseURL:              "postgres://example",
		ScanTelemetryEnvironment: "staging",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid scan telemetry environment to fail validation")
	}
}

func TestValidateAllowsOCRWithoutGeminiKey(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:                "8080",
		DatabaseURL:         "postgres://example",
		LabelParserProvider: "ocr",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected OCR config without Gemini key to validate, got %v", err)
	}
}

func TestValidateRequiresGeminiKeyWhenProviderIsHybrid(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:                        "8080",
		DatabaseURL:                 "postgres://example",
		LabelParserProvider:         "hybrid",
		LabelParserFallbackProvider: "gemini",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error when GEMINI_API_KEY is missing for the hybrid parser")
	}
}

func TestValidateRejectsUnsupportedHybridFallbackProvider(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:                        "8080",
		DatabaseURL:                 "postgres://example",
		LabelParserProvider:         "hybrid",
		LabelParserFallbackProvider: "openai",
		GeminiAPIKey:                "gemini-key",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for unsupported hybrid fallback provider")
	}
}

func TestValidateAllowsHybridWithGeminiKey(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:                        "8080",
		DatabaseURL:                 "postgres://example",
		LabelParserProvider:         "hybrid",
		LabelParserFallbackProvider: "gemini",
		GeminiAPIKey:                "gemini-key",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected hybrid config to validate, got %v", err)
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
