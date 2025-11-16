package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// ResponseConsumer обрабатывает ответы из Kafka
type ResponseConsumer struct {
	reader    *kafka.Reader
	logger    *logrus.Logger
	serverKey string
	waiters   sync.Map // map[string]chan *protocol.Message
	ctx       context.Context
	cancel    context.CancelFunc
}

// ResponseConsumerConfig конфигурация consumer
type ResponseConsumerConfig struct {
	Brokers        []string
	GroupID        string
	ServerKey      string
	Topic          string
	MinBytes       int
	MaxBytes       int
	CommitInterval time.Duration
}

// NewResponseConsumer создает новый consumer для ответов
func NewResponseConsumer(cfg ResponseConsumerConfig, logger *logrus.Logger) (*ResponseConsumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers list is empty")
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

	// Создаем reader для ответов
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.GroupID,
		Topic:          cfg.Topic,
		MinBytes:       cfg.MinBytes,
		MaxBytes:       cfg.MaxBytes,
		CommitInterval: cfg.CommitInterval,
		StartOffset:    kafka.LastOffset,
	})

	ctx, cancel := context.WithCancel(context.Background())

	logger.WithFields(logrus.Fields{
		"brokers":    cfg.Brokers,
		"group_id":   cfg.GroupID,
		"topic":      cfg.Topic,
		"server_key": cfg.ServerKey,
	}).Info("Kafka response consumer initialized")

	return &ResponseConsumer{
		reader:    reader,
		logger:    logger,
		serverKey: cfg.ServerKey,
		waiters:   sync.Map{},
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// Start запускает consumer в отдельной goroutine
func (c *ResponseConsumer) Start() error {
	c.logger.Info("Starting Kafka response consumer")

	go func() {
		for {
			select {
			case <-c.ctx.Done():
				c.logger.Info("Response consumer stopped")
				return
			default:
				if err := c.consumeMessage(); err != nil {
					c.logger.WithError(err).Error("Failed to consume message")
					time.Sleep(time.Second) // Prevent tight loop on errors
				}
			}
		}
	}()

	return nil
}

// consumeMessage обрабатывает одно сообщение
func (c *ResponseConsumer) consumeMessage() error {
	// Устанавливаем таймаут для чтения
	readCtx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	msg, err := c.reader.ReadMessage(readCtx)
	if err != nil {
		if err == context.DeadlineExceeded {
			return nil // Нормально, просто нет сообщений
		}
		return fmt.Errorf("failed to read message: %w", err)
	}

	// Извлекаем command ID из headers или ключа
	var commandID string
	for _, header := range msg.Headers {
		if header.Key == "command_id" {
			commandID = string(header.Value)
			break
		}
	}

	if commandID == "" {
		c.logger.Warn("Received message without command_id header")
		return nil
	}

	// Десериализуем ответ
	var response protocol.Message
	if err := json.Unmarshal(msg.Value, &response); err != nil {
		c.logger.WithError(err).WithField("message", string(msg.Value)).Error("Failed to unmarshal response")
		return nil
	}

	c.logger.WithFields(logrus.Fields{
		"command_id":    commandID,
		"response_type": response.Type,
		"partition":     msg.Partition,
		"offset":        msg.Offset,
	}).Debug("Received response from Kafka")

	// Ищем ожидающий response channel
	if waiter, exists := c.waiters.Load(commandID); exists {
		if waiterChan, ok := waiter.(chan *protocol.Message); ok {
			select {
			case waiterChan <- &response:
				c.logger.WithField("command_id", commandID).Debug("Response delivered to waiter")
			default:
				c.logger.WithField("command_id", commandID).Warn("Waiter channel full, dropping response")
			}
		} else {
			c.logger.WithField("command_id", commandID).Error("Invalid waiter type in sync.Map")
		}

		// Удаляем waiter после доставки
		c.waiters.Delete(commandID)
	} else {
		c.logger.WithField("command_id", commandID).Warn("No waiter found for response")
	}

	return nil
}

// WaitForResponse ожидает ответ для конкретной команды
func (c *ResponseConsumer) WaitForResponse(commandID string, timeout time.Duration) (*protocol.Message, error) {
	// Создаем канал для ответа
	responseChan := make(chan *protocol.Message, 1)

	// Регистрируем waiter
	c.waiters.Store(commandID, responseChan)

	// Удаляем waiter после завершения
	defer c.waiters.Delete(commandID)

	// Ждем ответ
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()

	select {
	case response := <-responseChan:
		return response, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

// Close закрывает consumer
func (c *ResponseConsumer) Close() error {
	c.logger.Info("Closing Kafka response consumer")
	c.cancel()
	return c.reader.Close()
}

// Stats возвращает статистику consumer
func (c *ResponseConsumer) Stats() kafka.ReaderStats {
	return c.reader.Stats()
}
