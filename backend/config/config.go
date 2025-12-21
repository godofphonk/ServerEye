package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

type Config struct {
	// Server
	Host string
	Port int

	// Database
	DatabaseURL string

	// Metrics
	MetricsTopic string

	// Security
	JWTSecret     string
	WebhookSecret string

	// Telegram
	TelegramBotToken string

	// Web
	WebURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		Host: getEnv("HOST", "0.0.0.0"),
		Port: getEnvInt("PORT", 8080),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/servereye?sslmode=disable"),

		MetricsTopic: getEnv("METRICS_TOPIC", "metrics"),

		JWTSecret:     getEnv("JWT_SECRET", "change-me-in-production"),
		WebhookSecret: getEnv("WEBHOOK_SECRET", "change-me-in-production"),

		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		WebURL:           getEnv("WEB_URL", "http://localhost:3000"),
	}

	// Validate required fields
	if cfg.JWTSecret == "change-me-in-production" {
		logrus.Warn("Using default JWT secret - please set JWT_SECRET in production")
	}

	if cfg.TelegramBotToken == "" {
		logrus.Warn("TELEGRAM_BOT_TOKEN not set - bot features will be disabled")
	}

	return cfg, nil
}

func (c *Config) GetAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		return []string{value}
	}
	return defaultValue
}
