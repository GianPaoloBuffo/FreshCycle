package config

import "os"

const defaultPort = "8080"

type Config struct {
	Port string
}

func Load() Config {
	return Config{
		Port: getEnv("API_PORT", defaultPort),
	}
}

func (c Config) Address() string {
	return ":" + c.Port
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
