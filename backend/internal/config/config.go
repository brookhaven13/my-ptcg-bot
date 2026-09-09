package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GeminiAPIKey string
	Port         string
	DatabaseURL  string
}

func Load() (*Config, error) {
	_ = godotenv.Load("../.env")

	cfg := &Config{
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
		Port:         os.Getenv("PORT"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = "postgres://postgres:postgres@localhost:5432/ptcg?sslmode=disable"
	}

	if cfg.GeminiAPIKey == "" {
		fmt.Println("Warning: GEMINI_API_KEY not set, AI features will be unavailable")
	}

	return cfg, nil
}
