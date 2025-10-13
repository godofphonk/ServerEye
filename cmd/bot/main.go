package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/servereye/servereye/internal/bot"
	"github.com/servereye/servereye/internal/config"
	"github.com/sirupsen/logrus"
)

const (
	defaultConfigPath = "/etc/servereye/bot-config.yaml"
	defaultLogLevel   = "info"
)

func main() {
	var (
		configPath = flag.String("config", defaultConfigPath, "Path to configuration file")
		logLevel   = flag.String("log-level", defaultLogLevel, "Log level (debug, info, warn, error)")
		version    = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	// Show version
	if *version {
		fmt.Println("ServerEye Bot v1.0.0")
		return
	}

	// Setup logger
	logger := setupLogger(*logLevel)

	// Load configuration
	cfg, err := loadConfigFromEnv()
	if err != nil {
		// Try to load from file if env vars not available
		cfg, err = config.LoadBotConfig(*configPath)
		if err != nil {
			logger.WithError(err).Fatal("Failed to load configuration")
		}
	}

	// Create and start bot
	botInstance, err := bot.New(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create bot")
	}

	if err := botInstance.Start(); err != nil {
		logger.WithError(err).Fatal("Failed to start bot")
	}

	logger.Info("ServerEye Bot started successfully")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down bot...")
	if err := botInstance.Stop(); err != nil {
		logger.WithError(err).Error("Error during shutdown")
	}
}

// setupLogger configures and returns a logger instance
func setupLogger(level string) *logrus.Logger {
	logger := logrus.New()
	
	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Set formatter for production (JSON) or development (text)
	if os.Getenv("ENVIRONMENT") == "production" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	}

	return logger
}

// loadConfigFromEnv loads configuration from environment variables
func loadConfigFromEnv() (*config.BotConfig, error) {
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	redisURL := os.Getenv("REDIS_URL")
	databaseURL := os.Getenv("DATABASE_URL")

	if telegramToken == "" || redisURL == "" || databaseURL == "" {
		return nil, fmt.Errorf("missing required environment variables")
	}

	// Parse Redis URL (simple implementation)
	redisAddr := "localhost:6379"
	if redisURL != "" {
		// Extract address from redis://host:port format
		if len(redisURL) > 8 && redisURL[:8] == "redis://" {
			redisAddr = redisURL[8:]
		}
	}

	return &config.BotConfig{
		Telegram: config.TelegramConfig{
			Token: telegramToken,
		},
		Redis: config.RedisConfig{
			Address:  redisAddr,
			Password: "",
			DB:       0,
		},
		Database: config.DatabaseConfig{
			URL: databaseURL,
		},
		Logging: config.LoggingConfig{
			Level: "info",
		},
	}, nil
}
