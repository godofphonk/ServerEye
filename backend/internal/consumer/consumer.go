package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
	"github.com/servereye/servereye/backend/internal/api"
	"github.com/servereye/servereye/backend/internal/config"
	"github.com/servereye/servereye/backend/internal/storage"
	"github.com/servereye/servereye/pkg/publisher"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Kafka struct {
		Brokers     []string
		TopicPrefix string
	}
}

type Consumer struct {
	config   *config.Config
	storage  *storage.Storage
	logger   *logrus.Logger
	wsServer *api.WebSocketServer
}

func New(cfg *config.Config, storage *storage.Storage, wsServer *api.WebSocketServer, logger *logrus.Logger) (*Consumer, error) {
	return &Consumer{
		config:   cfg,
		storage:  storage,
		logger:   logger,
		wsServer: wsServer,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting Kafka consumer for metrics broadcast")

	// Create Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     c.config.Kafka.Brokers,
		Topic:       c.config.Kafka.TopicPrefix + ".metrics",
		GroupID:     "servereye-backend",
		Partition:   0,                // Explicit partition to match producer
		MinBytes:    1,                // 1 byte to avoid blocking
		MaxBytes:    10e6,             // 10MB
		StartOffset: kafka.LastOffset, // Read only new messages
	})

	defer reader.Close()

	c.logger.WithFields(logrus.Fields{
		"brokers": c.config.Kafka.Brokers,
		"topic":   c.config.Kafka.TopicPrefix + ".metrics",
		"groupID": "servereye-backend",
	}).Info("Kafka reader created, starting to read messages")

	for {
		c.logger.Info("Waiting for Kafka message...")
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			c.logger.WithError(err).Error("Failed to read Kafka message")
			continue
		}

		c.logger.WithFields(logrus.Fields{
			"topic":     msg.Topic,
			"partition": msg.Partition,
			"offset":    msg.Offset,
			"size":      len(msg.Value),
		}).Info("Received Kafka message")

		if err := c.processMessage(msg); err != nil {
			c.logger.WithError(err).Error("Failed to process message")
		}

		// Auto-commit handled by consumer group
	}
}

func (c *Consumer) processMessage(msg kafka.Message) error {
	var metric publisher.Metric
	if err := json.Unmarshal(msg.Value, &metric); err != nil {
		return fmt.Errorf("failed to unmarshal metric: %w", err)
	}

	// Broadcast to WebSocket clients
	if c.wsServer != nil {
		c.wsServer.BroadcastMetric(metric)
		c.logger.WithFields(logrus.Fields{
			"server_id":   metric.ServerID,
			"metric_type": metric.Type,
		}).Info("Metric broadcasted to WebSocket clients")
	}

	return nil
}

func (c *Consumer) Shutdown(ctx context.Context) error {
	c.logger.Info("Shutting down Kafka consumer")
	return nil
}
