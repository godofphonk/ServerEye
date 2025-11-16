package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// sendCommandViaKafka отправляет команду через Kafka
func (b *Bot) sendCommandViaKafka(ctx context.Context, serverKey string, command *protocol.Message, timeout time.Duration) (*protocol.Message, error) {
	if !b.useKafka || b.commandProducer == nil {
		return nil, fmt.Errorf("Kafka not enabled or producer not initialized")
	}

	// Получаем logger
	var logger *logrus.Logger
	if sl, ok := b.logger.(*StructuredLogger); ok {
		logger = sl.logger
	} else {
		logger = logrus.New()
	}

	logger.WithFields(logrus.Fields{
		"command_id":   command.ID,
		"command_type": command.Type,
		"server_key":   serverKey,
	}).Info("Sending command via Kafka")

	// Отправляем команду
	if err := b.commandProducer.SendCommand(ctx, serverKey, command); err != nil {
		logger.WithError(err).Error("Failed to send command via Kafka")
		return nil, fmt.Errorf("failed to send command via Kafka: %w", err)
	}

	// Ждем ответ
	response, err := b.waitForKafkaResponse(command.ID, timeout)
	if err != nil {
		logger.WithError(err).WithField("command_id", command.ID).Error("Failed to get response via Kafka")
		return nil, err
	}

	logger.WithFields(logrus.Fields{
		"command_id":    command.ID,
		"response_type": response.Type,
	}).Info("Response received via Kafka")

	return response, nil
}

// waitForKafkaResponse ожидает ответ через Kafka response consumer
func (b *Bot) waitForKafkaResponse(commandID string, timeout time.Duration) (*protocol.Message, error) {
	if b.responseConsumer == nil {
		return nil, fmt.Errorf("response consumer not initialized")
	}

	// Ждем ответ от response consumer
	response, err := b.responseConsumer.WaitForResponse(commandID, timeout)
	if err != nil {
		return nil, fmt.Errorf("timeout waiting for Kafka response: %w", err)
	}

	return response, nil
}

// sendCommandViaKafkaWithFallback отправляет команду через Kafka с fallback на Streams/PubSub
func (b *Bot) sendCommandViaKafkaWithFallback(ctx context.Context, serverKey string, command *protocol.Message, timeout time.Duration) (*protocol.Message, error) {
	// Пробуем Kafka если доступен
	if b.useKafka && b.commandProducer != nil {
		response, err := b.sendCommandViaKafka(ctx, serverKey, command, timeout)
		if err == nil {
			return response, nil
		}
		
		// Логируем ошибку Kafka и пробуем fallback
		b.logger.Error("Kafka command failed, falling back to Streams", err)
	}

	// Fallback на Streams
	return b.sendCommandViaStreams(ctx, serverKey, command, timeout)
}
