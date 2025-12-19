package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/servereye/servereye/backend/internal/api"
	"github.com/servereye/servereye/backend/internal/config"
	"github.com/servereye/servereye/backend/internal/consumer"
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

	// Initialize API server
	apiServer, err := api.New(&api.Config{
		Server: struct {
			Host string
			Port string
		}{
			Host: "0.0.0.0",
			Port: "8080",
		},
		Kafka: struct {
			Brokers     []string
			TopicPrefix string
		}{
			Brokers:     cfg.Kafka.Brokers,
			TopicPrefix: cfg.Kafka.TopicPrefix,
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

	// Initialize Kafka consumer with WebSocket server
	kafkaConsumer, err := consumer.New(cfg, storage, apiServer.GetWebSocketServer(), logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Kafka consumer")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumer in goroutine
	go func() {
		if err := kafkaConsumer.Start(ctx); err != nil {
			logger.WithError(err).Error("Kafka consumer error")
			cancel()
		}
	}()

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

	if err := kafkaConsumer.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Error shutting down consumer")
	}

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Error shutting down API server")
	}

	logger.Info("Shutdown complete")
}
