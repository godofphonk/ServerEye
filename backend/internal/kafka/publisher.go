package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Publisher интерфейс для публикации метрик в Kafka
type Publisher interface {
	PublishMetric(ctx context.Context, metric *MetricMessage) error
	Close() error
}

// MetricMessage структура метрики для Kafka
type MetricMessage struct {
	ServerID   string            `json:"server_id"`
	ServerKey  string            `json:"server_key"`
	ServerName string            `json:"server_name"`
	Type       string            `json:"type"`
	Timestamp  time.Time         `json:"timestamp"`
	Value      interface{}       `json:"value"`
	Tags       map[string]string `json:"tags"`
	Version    string            `json:"version"`
}

// kafkaPublisher реализация Publisher для Kafka
type kafkaPublisher struct {
	writer      *kafka.Writer
	topicPrefix string
}

// NewPublisher создает новый Kafka publisher
func NewPublisher(brokers []string, topicPrefix string) (Publisher, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers list is empty")
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        true, // Асинхронная отправка для производительности
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		Compression:  kafka.Snappy,
	}

	logrus.WithFields(logrus.Fields{
		"brokers": brokers,
		"prefix":  topicPrefix,
	}).Info("Kafka publisher initialized")

	return &kafkaPublisher{
		writer:      writer,
		topicPrefix: topicPrefix,
	}, nil
}

// PublishMetric публикует метрику в Kafka
func (p *kafkaPublisher) PublishMetric(ctx context.Context, metric *MetricMessage) error {
	data, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	topic := fmt.Sprintf("%s.%s", p.topicPrefix, metric.Type)

	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(metric.ServerKey),
		Value: data,
		Headers: []kafka.Header{
			{Key: "server_id", Value: []byte(metric.ServerID)},
			{Key: "metric_type", Value: []byte(metric.Type)},
			{Key: "timestamp", Value: []byte(metric.Timestamp.Format(time.RFC3339))},
		},
	}

	// Асинхронная отправка
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error":       err,
			"server_id":   metric.ServerID,
			"metric_type": metric.Type,
			"topic":       topic,
		}).Error("Failed to publish metric to Kafka")
		return fmt.Errorf("failed to publish metric: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"server_id":   metric.ServerID,
		"metric_type": metric.Type,
		"topic":       topic,
	}).Debug("Metric published to Kafka")

	return nil
}

// Close закрывает Kafka writer
func (p *kafkaPublisher) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
