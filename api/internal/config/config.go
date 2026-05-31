package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const defaultPort = "8080"
const defaultOpenAIModel = "gpt-4.1-mini"
const defaultOCRTesseractPath = "tesseract"
const defaultOCRLanguages = "eng+spa"
const defaultLabelParserFallbackProvider = "gemini"
const defaultGeminiModel = "gemini-3.1-flash-lite"
const defaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"

type Config struct {
	Port                        string
	DatabaseURL                 string
	LabelParserProvider         string
	LabelParserFallbackProvider string
	OpenAIAPIKey                string
	OpenAIModel                 string
	OpenAIBaseURL               string
	OCRTesseractPath            string
	OCRLanguages                string
	GeminiAPIKey                string
	GeminiModel                 string
	GeminiBaseURL               string
	AllowedOrigins              []string
	SupabaseProjectURL          string
	SupabaseSecretKey           string
}

func Load() (Config, error) {
	return LoadFromMap(nil)
}

func LoadFromMap(values map[string]string) (Config, error) {
	lookup := func(key string) string {
		if values != nil {
			if value, ok := values[key]; ok {
				return value
			}
		}

		return os.Getenv(key)
	}

	cfg := Config{
		Port:                        getEnvFromLookup(lookup, "API_PORT", defaultPort),
		DatabaseURL:                 lookup("SUPABASE_DB_URL"),
		LabelParserProvider:         getEnvFromLookup(lookup, "LABEL_PARSER_PROVIDER", "stub"),
		LabelParserFallbackProvider: getEnvFromLookup(lookup, "LABEL_PARSER_FALLBACK_PROVIDER", defaultLabelParserFallbackProvider),
		OpenAIAPIKey:                lookup("OPENAI_API_KEY"),
		OpenAIModel:                 getEnvFromLookup(lookup, "OPENAI_MODEL", defaultOpenAIModel),
		OpenAIBaseURL:               getEnvFromLookup(lookup, "OPENAI_BASE_URL", ""),
		OCRTesseractPath:            getEnvFromLookup(lookup, "OCR_TESSERACT_PATH", defaultOCRTesseractPath),
		OCRLanguages:                getEnvFromLookup(lookup, "OCR_LANGUAGES", defaultOCRLanguages),
		GeminiAPIKey:                lookup("GEMINI_API_KEY"),
		GeminiModel:                 getEnvFromLookup(lookup, "GEMINI_MODEL", defaultGeminiModel),
		GeminiBaseURL:               getEnvFromLookup(lookup, "GEMINI_BASE_URL", defaultGeminiBaseURL),
		AllowedOrigins:              splitCSVEnv(getEnvFromLookup(lookup, "API_ALLOWED_ORIGINS", defaultAllowedOrigins)),
		SupabaseProjectURL:          getEnvFromLookup(lookup, "SUPABASE_URL", ""),
		SupabaseSecretKey:           getEnvFromLookup(lookup, "SUPABASE_SECRET_KEY", ""),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Address() string {
	return ":" + c.Port
}

func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("SUPABASE_DB_URL is required")
	}

	if c.Port == "" {
		return errors.New("API_PORT cannot be empty")
	}

	provider := strings.ToLower(strings.TrimSpace(c.LabelParserProvider))
	fallbackProvider := strings.ToLower(strings.TrimSpace(c.LabelParserFallbackProvider))

	if provider == "openai" && strings.TrimSpace(c.OpenAIAPIKey) == "" {
		return errors.New("OPENAI_API_KEY is required when LABEL_PARSER_PROVIDER=openai")
	}

	if provider == "hybrid" {
		if fallbackProvider != "gemini" {
			return errors.New("LABEL_PARSER_FALLBACK_PROVIDER must be gemini when LABEL_PARSER_PROVIDER=hybrid")
		}

		if strings.TrimSpace(c.GeminiAPIKey) == "" {
			return errors.New("GEMINI_API_KEY is required when LABEL_PARSER_PROVIDER=hybrid")
		}
	}

	return nil
}

func (c Config) RedactedDatabaseURL() string {
	if c.DatabaseURL == "" {
		return ""
	}

	return fmt.Sprintf("%s://%s", "postgres", "[configured]")
}

func getEnvFromLookup(lookup func(string) string, key string, fallback string) string {
	value := lookup(key)
	if value == "" {
		return fallback
	}

	return value
}

func splitCSVEnv(value string) []string {
	rawValues := strings.Split(value, ",")
	values := make([]string, 0, len(rawValues))
	for _, item := range rawValues {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		values = append(values, trimmed)
	}

	return values
}

const defaultAllowedOrigins = "http://localhost:19006,http://127.0.0.1:19006,http://localhost:8081,https://*.vercel.app"
