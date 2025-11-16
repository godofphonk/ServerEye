package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// CommandProducer отправляет команды в Kafka
type CommandProducer struct {
	writer *kafka.Writer
	logger *logrus.Logger
}

// CommandProducerConfig конфигурация producer
type CommandProducerConfig struct {
	Brokers      []string
	Topic        string
	Compression  string
	BatchSize    int
	BatchTimeout time.Duration
}

// NewCommandProducer создает новый producer для команд
func NewCommandProducer(cfg CommandProducerConfig, logger *logrus.Logger) (*CommandProducer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers list is empty")
	}
	if logger == nil {
		logger = logrus.New()
	}

	// Устанавливаем значения по умолчанию
	if cfg.Topic == "" {
		cfg.Topic = "servereye.commands"
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 1 // Для команд отправляем сразу, не ждем батча
	}
	if cfg.BatchTimeout == 0 {
		cfg.BatchTimeout = 10 * time.Millisecond
	}

	// Создаем writer
	writer := kafka.WriterConfig{
		Brokers:      cfg.Brokers,
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		Async:        false, // Для команд используем синхронную отправку
	}

	kafkaWriter := kafka.NewWriter(writer)

	// Устанавливаем compression
	switch cfg.Compression {
	case "gzip":
		kafkaWriter.Compression = kafka.Gzip
	case "snappy":
		kafkaWriter.Compression = kafka.Snappy
	case "lz4":
		kafkaWriter.Compression = kafka.Lz4
	case "zstd":
		kafkaWriter.Compression = kafka.Zstd
	default:
		kafkaWriter.Compression = kafka.Compression(0) // None
	}

	logger.WithFields(logrus.Fields{
		"brokers":     cfg.Brokers,
		"topic":       cfg.Topic,
		"compression": cfg.Compression,
		"batch_size":  cfg.BatchSize,
	}).Info("Kafka command producer initialized")

	return &CommandProducer{
		writer: kafkaWriter,
		logger: logger,
	}, nil
}

// SendCommand отправляет команду в Kafka
func (p *CommandProducer) SendCommand(ctx context.Context, serverKey string, command *protocol.Message) error {
	// Сериализуем команду
	value, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	// Создаем сообщение
	msg := kafka.Message{
		Key:   []byte(serverKey), // Ключ для партиционирования по серверу
		Value: value,
		Time:  command.Timestamp,
		Headers: []kafka.Header{
			{Key: "server_key", Value: []byte(serverKey)},
			{Key: "command_id", Value: []byte(command.ID)},
			{Key: "command_type", Value: []byte(string(command.Type))},
			{Key: "server_id", Value: []byte(command.ServerID)},
		},
	}

	// Отправляем сообщение
	start := time.Now()
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		p.logger.WithFields(logrus.Fields{
			"command_id":   command.ID,
			"command_type": command.Type,
			"server_key":   serverKey,
			"error":        err,
		}).Error("Failed to send command to Kafka")
		return fmt.Errorf("kafka write failed: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"command_id":   command.ID,
		"command_type": command.Type,
		"server_key":   serverKey,
		"duration":     time.Since(start),
	}).Debug("Command sent to Kafka")

	return nil
}

// Close закрывает producer
func (p *CommandProducer) Close() error {
	p.logger.Info("Closing Kafka command producer")
	return p.writer.Close()
}

// Stats возвращает статистику producer
func (p *CommandProducer) Stats() kafka.WriterStats {
	return p.writer.Stats()
}
