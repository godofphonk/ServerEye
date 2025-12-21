package config

import (
	"fmt"
	"os"
)

type Config struct {
	Server struct {
		Host string `env:"SERVER_HOST" envDefault:"0.0.0.0"`
		Port string `env:"SERVER_PORT" envDefault:"8080"`
	}
	DatabaseURL string `env:"DATABASE_URL" envDefault:"postgres://user:password@localhost/servereye?sslmode=disable"`

	Kafka struct {
		Brokers     []string `env:"KAFKA_BROKERS" envDefault:"localhost:9092"`
		TopicPrefix string   `env:"KAFKA_TOPIC_PREFIX" envDefault:"metrics"`
	}

	Auth struct {
		APIKey string `env:"API_KEY" envDefault:""`
	}

	// Telegram
	TelegramBotToken string
}

func Load() (*Config, error) {
	cfg := &Config{}

	// For simplicity, using direct env var reading
	cfg.Server.Host = getEnv("SERVER_HOST", "0.0.0.0")
	cfg.Server.Port = getEnv("SERVER_PORT", "8080")
	cfg.DatabaseURL = getEnv("DATABASE_URL", "postgres://user:password@localhost/servereye?sslmode=disable")
	cfg.Kafka.Brokers = []string{getEnv("KAFKA_BROKERS", "localhost:9092")}
	cfg.Kafka.TopicPrefix = getEnv("KAFKA_TOPIC_PREFIX", "metrics")
	cfg.Auth.APIKey = getEnv("API_KEY", "")
	cfg.TelegramBotToken = getEnv("TELEGRAM_BOT_TOKEN", "")

	if cfg.Auth.APIKey == "" {
		return nil, fmt.Errorf("API_KEY environment variable is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
