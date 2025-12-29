package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

type Config struct {
	// Server
	Host string
	Port int

	// Database
	DatabaseURL     string
	KeysDatabaseURL string

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

		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/servereye?sslmode=disable"),
		KeysDatabaseURL: getEnv("KEYS_DATABASE_URL", "postgres://servereye_keys:KMRb0xHxWCH%2FQa28YskBl62xI%2FBfkwi%2FZPiHMrZueEc%3D@localhost:5433/PgRegisteredKeys?sslmode=disable"),

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

// Consumer configuration
type ConsumerConfig struct {
	Brokers           []string
	GroupID           string
	Topic             string
	BatchSize         int
	BatchTimeout      time.Duration
	CommitInterval    time.Duration
	RebalanceTimeout  time.Duration
	StartOffset       int64 // -2 = earliest, -1 = latest, 0+ = specific
	MaxProcessingTime time.Duration
	EnableAutoCommit  bool
	SessionTimeout    time.Duration
	HeartbeatInterval time.Duration
	MaxPollRecords    int
}

func NewConsumerConfig(cfg *Config) ConsumerConfig {
	return ConsumerConfig{
		Brokers:           cfg.KafkaBrokers,
		GroupID:           cfg.KafkaGroupID,
		Topic:             cfg.MetricsTopic,
		BatchSize:         100,
		BatchTimeout:      1 * time.Second,
		CommitInterval:    1 * time.Second,
		RebalanceTimeout:  30 * time.Second,
		StartOffset:       -2, // Start from earliest
		MaxProcessingTime: 30 * time.Second,
		EnableAutoCommit:  false,
		SessionTimeout:    30 * time.Second,
		HeartbeatInterval: 3 * time.Second,
		MaxPollRecords:    500,
	}
}
