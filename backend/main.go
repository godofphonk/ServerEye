package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/servereye/servereye/backend/internal/api"
	"github.com/servereye/servereye/backend/internal/config"
	"github.com/servereye/servereye/backend/internal/telegram"
	"github.com/servereye/servereye/backend/storage"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load config")
	}

	// Initialize storage
	storage, err := storage.New(cfg.DatabaseURL, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize storage")
	}
	defer storage.Close()

	// Initialize API server (HTTP-only mode)
	apiServer, err := api.New(&api.Config{
		Server: struct {
			Host string
			Port string
		}{
			Host: "0.0.0.0",
			Port: "8080",
		},
		Auth: struct {
			APIKey string
		}{
			APIKey: cfg.Auth.APIKey,
		},
	}, logger, storage)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize API server")
	}

	// Initialize Telegram bot if token is provided
	var telegramBot *telegram.Bot
	if cfg.TelegramBotToken != "" {
		telegramBot, err = telegram.NewBot(cfg.TelegramBotToken, logger)
		if err != nil {
			logger.WithError(err).Error("Failed to initialize Telegram bot")
		} else {
			logger.Info("Telegram bot initialized successfully")
		}
	} else {
		logger.Warn("TELEGRAM_BOT_TOKEN not set - Telegram bot disabled")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Telegram bot in goroutine if initialized
	if telegramBot != nil {
		go func() {
			if err := telegramBot.Start(ctx); err != nil {
				logger.WithError(err).Error("Telegram bot error")
			}
		}()
	}

	// Start API server (blocking)
	if err := apiServer.Start(); err != nil {
		logger.WithError(err).Fatal("Failed to start API server")
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutting down...")

	// Shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Error shutting down API server")
	}

	logger.Info("Shutdown complete")
}
