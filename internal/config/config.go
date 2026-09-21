package config

import "os"

type Config struct {
	BaseURL string
}

func Load() Config {
	baseURL := os.Getenv("BASE_URL")

	if baseURL == "" {
		baseURL = ":8080"
	}

	return Config{
		BaseURL: baseURL,
	}
}
