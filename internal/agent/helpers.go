package agent

import (
	"fmt"
	"time"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/servereye/servereye/pkg/redis"
)

// handlePing обрабатывает ping команду
func (a *Agent) handlePing(msg *protocol.Message) *protocol.Message {
	payload := protocol.PongPayload{
		Status: "healthy",
		Uptime: "unknown",
	}

	response := protocol.NewMessage(protocol.TypePong, payload)
	response.ID = msg.ID
	return response
}

// handleUnknownCommand обрабатывает неизвестную команду
func (a *Agent) handleUnknownCommand(msg *protocol.Message) *protocol.Message {
	payload := protocol.ErrorPayload{
		ErrorCode:    protocol.ErrorInvalidCommand,
		ErrorMessage: fmt.Sprintf("Неизвестная команда: %s", msg.Type),
	}

	response := protocol.NewMessage(protocol.TypeErrorResponse, payload)
	response.ID = msg.ID
	return response
}

// sendResponse отправляет ответ в канал ответов (legacy Pub/Sub)
func (a *Agent) sendResponse(msg *protocol.Message) error {
	data, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("не удалось сериализовать ответ: %w", err)
	}

	respChannel := redis.GetResponseChannel(a.config.Server.SecretKey)
	return a.redisClient.Publish(a.ctx, respChannel, data)
}

// sendResponseToCommand отправляет ответ в Stream или Pub/Sub
func (a *Agent) sendResponseToCommand(msg *protocol.Message, commandID string) error {
	data, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("не удалось сериализовать ответ: %w", err)
	}

	// Use Streams if available
	if a.useStreams && a.streamsClient != nil {
		respStream := fmt.Sprintf("stream:resp:%s", a.config.Server.SecretKey)

		values := map[string]string{
			"type":       string(msg.Type),
			"id":         msg.ID,
			"command_id": commandID,
			"payload":    string(data),
			"timestamp":  time.Now().Format(time.RFC3339),
		}

		if _, err := a.streamsClient.AddMessage(a.ctx, respStream, values); err != nil {
			a.logger.WithError(err).Error("Failed to send via Streams")
			return err
		}

		a.logger.WithField("stream", respStream).Debug("Response sent via Streams")
		return nil
	}

	// Fallback to Pub/Sub
	respChannel := fmt.Sprintf("resp:%s:%s", a.config.Server.SecretKey, commandID)
	a.logger.WithField("response_channel", respChannel).Debug("Отправка ответа в уникальный канал")
	return a.redisClient.Publish(a.ctx, respChannel, data)
}
