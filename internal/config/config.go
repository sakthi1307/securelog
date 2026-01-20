package config

import (
	"os"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	APIKey      string

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func Load() Config {
	return Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://securelog:securelog@localhost:5432/securelog?sslmode=disable"),
		APIKey:      getenv("API_KEY", "dev-key"),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

