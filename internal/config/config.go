package config

import (
	"log"
	"os"
)

type Config struct {
	BaseURL     string
	DatabaseURL string
}

func Load() *Config {
	cfg := &Config{
		BaseURL:     getEnv("BASE_URL", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}
