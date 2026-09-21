package config

import (
	"fmt"
	"log"
	"os"
)

type Config struct {
	BaseURL string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	DatabaseURL string
}

func Load() *Config {
	cfg := &Config{
		BaseURL: getEnv("BASE_URL", ":8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "flexcrm"),
	}

	if cfg.DBPassword == "" {
		log.Fatal("DB_PASSWORD is required")
	}

	cfg.DatabaseURL = fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	return cfg
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}
