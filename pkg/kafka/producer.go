package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/servereye/servereye/pkg/publisher"
	"github.com/sirupsen/logrus"
)

// Config конфигурация Kafka producer
type Config struct {
	Brokers       []string      // Список Kafka брокеров
	TopicPrefix   string        // Префикс для топиков (например, "metrics")
	Compression   string        // Тип сжатия: "none", "gzip", "snappy", "lz4", "zstd"
	MaxAttempts   int           // Максимальное количество попыток отправки
	BatchSize     int           // Размер батча сообщений
	BatchTimeout  time.Duration // Таймаут для батча
	RequiredAcks  int           // -1 = all, 0 = none, 1 = leader
	WriteTimeout  time.Duration // Таймаут записи
	EnableIdempot bool          // Включить идемпотентность
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		Brokers:       []string{"localhost:9092"},
		TopicPrefix:   "metrics",
		Compression:   "snappy",
		MaxAttempts:   3,
		BatchSize:     100,
		BatchTimeout:  1 * time.Second,
		RequiredAcks:  1,
		WriteTimeout:  10 * time.Second,
		EnableIdempot: true,
	}
}

// Producer Kafka producer для отправки метрик
type Producer struct {
	writer *kafka.Writer
	config Config
	logger *logrus.Logger
}

// NewProducer создает новый Kafka producer
func NewProducer(cfg Config, logger *logrus.Logger) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers list is empty")
	}

	if logger == nil {
		logger = logrus.New()
	}

	// Преобразуем строку compression в тип
	var compression kafka.Compression
	switch cfg.Compression {
	case "gzip":
		compression = kafka.Gzip
	case "snappy":
		compression = kafka.Snappy
	case "lz4":
		compression = kafka.Lz4
	case "zstd":
		compression = kafka.Zstd
	default:
		compression = kafka.Compression(0) // None
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.Hash{}, // Распределение по server_id
		Compression:  compression,
		MaxAttempts:  cfg.MaxAttempts,
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		ReadTimeout:  cfg.WriteTimeout,
		WriteTimeout: cfg.WriteTimeout,
		RequiredAcks: kafka.RequiredAcks(cfg.RequiredAcks),
		Async:        false, // Синхронная отправка для гарантии
	}

	if cfg.EnableIdempot {
		writer.AllowAutoTopicCreation = true
	}

	logger.WithFields(logrus.Fields{
		"brokers":     cfg.Brokers,
		"compression": cfg.Compression,
		"batch_size":  cfg.BatchSize,
	}).Info("Kafka producer initialized")

	return &Producer{
		writer: writer,
		config: cfg,
		logger: logger,
	}, nil
}

// Publish отправляет метрику в Kafka
func (p *Producer) Publish(ctx context.Context, metric *publisher.Metric) error {
	// Определяем топик на основе типа метрики
	topic := p.getTopicName(metric.Type)

	// Сериализуем метрику в JSON
	value, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	// Создаем Kafka message
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(metric.ServerID), // Ключ для партиционирования
		Value: value,
		Time:  metric.Timestamp,
		Headers: []kafka.Header{
			{Key: "server_id", Value: []byte(metric.ServerID)},
			{Key: "server_key", Value: []byte(metric.ServerKey)},
			{Key: "metric_type", Value: []byte(metric.Type)},
			{Key: "version", Value: []byte(metric.Version)},
		},
	}

	// Отправляем сообщение
	start := time.Now()
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		p.logger.WithFields(logrus.Fields{
			"topic":       topic,
			"server_id":   metric.ServerID,
			"metric_type": metric.Type,
			"error":       err,
		}).Error("Failed to publish metric to Kafka")
		return fmt.Errorf("kafka write failed: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"topic":       topic,
		"server_id":   metric.ServerID,
		"metric_type": metric.Type,
		"duration":    time.Since(start),
	}).Debug("Metric published to Kafka")

	return nil
}

// PublishBatch отправляет пакет метрик
func (p *Producer) PublishBatch(ctx context.Context, metrics []*publisher.Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	messages := make([]kafka.Message, 0, len(metrics))

	for _, metric := range metrics {
		topic := p.getTopicName(metric.Type)
		value, err := json.Marshal(metric)
		if err != nil {
			p.logger.WithError(err).Warn("Failed to marshal metric in batch")
			continue
		}

		messages = append(messages, kafka.Message{
			Topic: topic,
			Key:   []byte(metric.ServerID),
			Value: value,
			Time:  metric.Timestamp,
			Headers: []kafka.Header{
				{Key: "server_id", Value: []byte(metric.ServerID)},
				{Key: "server_key", Value: []byte(metric.ServerKey)},
				{Key: "metric_type", Value: []byte(metric.Type)},
			},
		})
	}

	if len(messages) == 0 {
		return fmt.Errorf("no valid messages to publish")
	}

	start := time.Now()
	err := p.writer.WriteMessages(ctx, messages...)
	if err != nil {
		p.logger.WithFields(logrus.Fields{
			"count": len(messages),
			"error": err,
		}).Error("Failed to publish batch to Kafka")
		return fmt.Errorf("kafka batch write failed: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"count":    len(messages),
		"duration": time.Since(start),
	}).Debug("Batch published to Kafka")

	return nil
}

// Close закрывает Kafka producer
func (p *Producer) Close() error {
	p.logger.Info("Closing Kafka producer")
	return p.writer.Close()
}

// Name возвращает имя publisher
func (p *Producer) Name() string {
	return "kafka"
}

// getTopicName формирует имя топика
func (p *Producer) getTopicName(metricType string) string {
	return fmt.Sprintf("%s.%s", p.config.TopicPrefix, metricType)
}

// Stats возвращает статистику Kafka producer
func (p *Producer) Stats() kafka.WriterStats {
	return p.writer.Stats()
}
