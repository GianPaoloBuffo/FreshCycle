package config

import (
	"errors"
	"fmt"
	"os"
)

const defaultPort = "8080"

type Config struct {
	Port        string
	DatabaseURL string
}

func Load() (Config, error) {
	cfg := Config{
		Port:        getEnv("API_PORT", defaultPort),
		DatabaseURL: os.Getenv("SUPABASE_DB_URL"),
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

	return nil
}

func (c Config) RedactedDatabaseURL() string {
	if c.DatabaseURL == "" {
		return ""
	}

	return fmt.Sprintf("%s://%s", "postgres", "[configured]")
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
