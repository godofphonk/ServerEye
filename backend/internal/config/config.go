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
	DatabaseURL     string `env:"DATABASE_URL" envDefault:"postgres://user:password@localhost/servereye?sslmode=disable"`
	KeysDatabaseURL string `env:"KEYS_DATABASE_URL" envDefault:"postgres://servereye_keys:KMRb0xHxWCH%2FQa28YskBl62xI%2FBfkwi%2FZPiHMrZueEc%3D@localhost:5433/PgRegisteredKeys?sslmode=disable"`

	Auth struct {
		APIKey        string `env:"API_KEY" envDefault:""`
		JWTSecret     string `env:"JWT_SECRET" envDefault:"change-me-in-production"`
		WebhookSecret string `env:"WEBHOOK_SECRET" envDefault:"change-me-in-production"`
	}

	Kafka struct {
		Brokers     []string `env:"KAFKA_BROKERS" envDefault:"localhost:9092"`
		TopicPrefix string   `env:"KAFKA_TOPIC_PREFIX" envDefault:"metrics"`
		Enabled     bool     `env:"KAFKA_ENABLED" envDefault:"false"`
	}
}

func Load() (*Config, error) {
	cfg := &Config{}

	// For simplicity, using direct env var reading
	cfg.Server.Host = getEnv("SERVER_HOST", "0.0.0.0")
	cfg.Server.Port = getEnv("SERVER_PORT", "8080")
	cfg.DatabaseURL = getEnv("DATABASE_URL", "postgres://user:password@localhost/servereye?sslmode=disable")
	cfg.KeysDatabaseURL = getEnv("KEYS_DATABASE_URL", "postgres://servereye_keys:KMRb0xHxWCH%2FQa28YskBl62xI%2FBfkwi%2FZPiHMrZueEc%3D@localhost:5433/PgRegisteredKeys?sslmode=disable")
	cfg.Auth.APIKey = getEnv("API_KEY", "")
	cfg.Auth.JWTSecret = getEnv("JWT_SECRET", "change-me-in-production")
	cfg.Auth.WebhookSecret = getEnv("WEBHOOK_SECRET", "change-me-in-production")

	// Kafka configuration
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	if kafkaBrokers != "" {
		cfg.Kafka.Brokers = []string{kafkaBrokers}
	}
	cfg.Kafka.TopicPrefix = getEnv("KAFKA_TOPIC_PREFIX", "metrics")
	cfg.Kafka.Enabled = getEnv("KAFKA_ENABLED", "false") == "true"

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

func (c *Config) GetAddr() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}
