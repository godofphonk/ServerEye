package consumer

import (
	"context"

	"github.com/servereye/servereye/backend/internal/storage"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Kafka struct {
		Brokers     []string
		TopicPrefix string
	}
}

type Consumer struct {
	config  *Config
	storage *storage.Storage
	logger  *logrus.Logger
}

func New(cfg interface{}, storage *storage.Storage, logger *logrus.Logger) (*Consumer, error) {
	return &Consumer{
		storage: storage,
		logger:  logger,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting Kafka consumer (placeholder)")
	// TODO: Implement actual Kafka consumer for commands
	<-ctx.Done()
	return nil
}

func (c *Consumer) Shutdown(ctx context.Context) error {
	c.logger.Info("Shutting down Kafka consumer")
	return nil
}
