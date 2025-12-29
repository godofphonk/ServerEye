package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/godofphonk/ServerEye/backend/internal/api"
	"github.com/godofphonk/ServerEye/backend/internal/config"
	"github.com/godofphonk/ServerEye/backend/internal/storage"
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

	// Initialize keys storage
	var keysStore *storage.KeysStorage
	if cfg.KeysDatabaseURL != "" && cfg.KeysDatabaseURL != "skip" {
		var err error
		keysStore, err = storage.NewKeysStorage(cfg.KeysDatabaseURL, logger)
		if err != nil {
			logger.WithError(err).Fatal("Failed to initialize keys storage")
		}
		defer keysStore.Close()
	} else {
		logger.Info("Keys storage disabled")
	}

	// Initialize API server (HTTP-only mode)
	apiServer, err := api.New(&api.Config{
		Server: struct {
			Host string
			Port string
		}{
			Host: cfg.Server.Host,
			Port: cfg.Server.Port,
		},
		Auth: struct {
			APIKey string
		}{
			APIKey: cfg.Auth.APIKey,
		},
		Kafka: struct {
			Brokers     []string
			TopicPrefix string
			Enabled     bool
		}{
			Brokers:     cfg.Kafka.Brokers,
			TopicPrefix: cfg.Kafka.TopicPrefix,
			Enabled:     cfg.Kafka.Enabled,
		},
	}, logger, keysStore)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize API server")
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
