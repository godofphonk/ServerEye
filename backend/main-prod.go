package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/servereye/servereye/backend/internal/api"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Get Kafka brokers from environment or use default
	brokers := []string{"kafka:9093"}
	if brokersEnv := os.Getenv("KAFKA_BROKERS"); brokersEnv != "" {
		brokers = []string{brokersEnv}
	}

	// Simple config for production
	apiConfig := &api.Config{
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
			Brokers:     brokers,
			TopicPrefix: "metrics",
		},
		Auth: struct {
			APIKey string
		}{
			APIKey: os.Getenv("API_KEY"),
		},
	}

	// Initialize API server (without storage)
	apiServer, err := api.New(apiConfig, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize API server")
	}

	// Start API server
	go func() {
		if err := apiServer.Start(); err != nil {
			logger.WithError(err).Error("API server error")
		}
	}()

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
