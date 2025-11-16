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

// CommandConsumer обрабатывает команды из Kafka
type CommandConsumer struct {
	reader    *kafka.Reader
	handler   CommandHandler
	logger    *logrus.Logger
	serverKey string
}

// CommandHandler интерфейс для обработки команд
type CommandHandler interface {
	HandleCommand(ctx context.Context, command *protocol.Message) (*protocol.Message, error)
}

// CommandConsumerConfig конфигурация consumer
type CommandConsumerConfig struct {
	Brokers        []string
	GroupID        string
	ServerKey      string
	Topic          string
	MinBytes       int
	MaxBytes       int
	CommitInterval time.Duration
}

// NewCommandConsumer создает новый consumer для команд
func NewCommandConsumer(cfg CommandConsumerConfig, handler CommandHandler, logger *logrus.Logger) (*CommandConsumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers list is empty")
	}
	if handler == nil {
		return nil, fmt.Errorf("command handler is required")
	}
	if logger == nil {
		logger = logrus.New()
	}

	// Устанавливаем значения по умолчанию
	if cfg.MinBytes == 0 {
		cfg.MinBytes = 10e3 // 10KB
	}
	if cfg.MaxBytes == 0 {
		cfg.MaxBytes = 10e6 // 10MB
	}
	if cfg.CommitInterval == 0 {
		cfg.CommitInterval = time.Second
	}

	// Создаем reader для команд
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.GroupID,
		Topic:          cfg.Topic,
		MinBytes:       cfg.MinBytes,
		MaxBytes:       cfg.MaxBytes,
		CommitInterval: cfg.CommitInterval,
		StartOffset:    kafka.LastOffset,
	})

	logger.WithFields(logrus.Fields{
		"brokers":    cfg.Brokers,
		"group_id":   cfg.GroupID,
		"topic":      cfg.Topic,
		"server_key": cfg.ServerKey,
	}).Info("Kafka command consumer initialized")

	return &CommandConsumer{
		reader:    reader,
		handler:   handler,
		logger:    logger,
		serverKey: cfg.ServerKey,
	}, nil
}

// Start запускает consumer в отдельной goroutine
func (c *CommandConsumer) Start(ctx context.Context) error {
	c.logger.Info("Starting Kafka command consumer")

	go func() {
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("Command consumer stopped")
				return
			default:
				if err := c.consumeMessage(ctx); err != nil {
					c.logger.WithError(err).Error("Failed to consume message")
					time.Sleep(time.Second) // Prevent tight loop on errors
				}
			}
		}
	}()

	return nil
}

// consumeMessage обрабатывает одно сообщение
func (c *CommandConsumer) consumeMessage(ctx context.Context) error {
	// Устанавливаем таймаут для чтения
	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	msg, err := c.reader.ReadMessage(readCtx)
	if err != nil {
		if err == context.DeadlineExceeded {
			return nil // Нормально, просто нет сообщений
		}
		return fmt.Errorf("failed to read message: %w", err)
	}

	// Десериализуем команду
	var command protocol.Message
	if err := json.Unmarshal(msg.Value, &command); err != nil {
		c.logger.WithError(err).WithField("message", string(msg.Value)).Error("Failed to unmarshal command")
		return nil // Не возвращаем ошибку, чтобы продолжить обработку других сообщений
	}

	c.logger.WithFields(logrus.Fields{
		"command_id":   command.ID,
		"command_type": command.Type,
		"partition":    msg.Partition,
		"offset":       msg.Offset,
	}).Info("Received command from Kafka")

	// Обрабатываем команду
	response, err := c.handler.HandleCommand(ctx, &command)
	if err != nil {
		c.logger.WithError(err).WithField("command_id", command.ID).Error("Failed to handle command")

		// Отправляем error response
		errorResponse := &protocol.Message{
			ID:        command.ID,
			Type:      protocol.TypeErrorResponse,
			Timestamp: time.Now(),
			ServerID:  command.ServerID,
			ServerKey: command.ServerKey,
			Payload:   map[string]string{"error": err.Error()},
		}

		if sendErr := c.sendResponse(ctx, errorResponse); sendErr != nil {
			c.logger.WithError(sendErr).Error("Failed to send error response")
		}
		return nil
	}

	// Отправляем успешный response
	if err := c.sendResponse(ctx, response); err != nil {
		c.logger.WithError(err).Error("Failed to send response")
	}

	return nil
}

// sendResponse отправляет response в отдельный топик
func (c *CommandConsumer) sendResponse(ctx context.Context, response *protocol.Message) error {
	// Создаем writer для ответов
	responseTopic := "servereye.responses" // Единый топик для всех ответов

	writer := &kafka.Writer{
		Addr:     kafka.TCP(c.reader.Config().Brokers...),
		Topic:    responseTopic,
		Balancer: &kafka.Hash{},
		Async:    false,
	}

	defer writer.Close()

	// Сериализуем response
	value, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Создаем сообщение с правильным partitioning по serverKey
	msg := kafka.Message{
		Key:   []byte(response.ServerKey), // Используем serverKey для partitioning
		Value: value,
		Time:  response.Timestamp,
		Headers: []kafka.Header{
			{Key: "server_key", Value: []byte(response.ServerKey)},
			{Key: "command_id", Value: []byte(response.ID)},
			{Key: "response_type", Value: []byte(string(response.Type))},
		},
	}

	// Отправляем сообщение
	if err := writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"command_id":    response.ID,
		"response_type": response.Type,
		"server_key":    response.ServerKey,
		"topic":         responseTopic,
	}).Info("Response sent to Kafka")

	return nil
}

// Close закрывает consumer
func (c *CommandConsumer) Close() error {
	c.logger.Info("Closing Kafka command consumer")
	return c.reader.Close()
}

// Stats возвращает статистику consumer
func (c *CommandConsumer) Stats() kafka.ReaderStats {
	return c.reader.Stats()
}
