package config

import (
	"os"
	"strconv"

	"scrapeNPM/internal/db"
)

type Config struct {
	DB db.Config
}

func Load() Config {
	return Config{
		DB: db.Config{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "ctuser"),
			Password: getEnv("DB_PASSWORD", "password"),
			Database: getEnv("DB_NAME", "scrapeNPM"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return fallback
}
