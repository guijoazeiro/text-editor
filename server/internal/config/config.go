package config

import (
	"os"
	"strings"
)

type Config struct {
	DatabaseURL    string
	Port           string
	Environment    string
	JWTSecret      string
	AllowedOrigins []string
}

func New() *Config {
	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/docs_db?sslmode=disable"),
		Port:           getEnv("PORT", "8080"),
		Environment:    getEnv("ENVIRONMENT", "development"),
		JWTSecret:      getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),
		AllowedOrigins: parseList(getEnv("ALLOWED_ORIGINS", "http://localhost:3000")),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseList(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
