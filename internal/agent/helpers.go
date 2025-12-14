package agent

import (
	"fmt"

	"github.com/servereye/servereye/pkg/protocol"
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

// sendResponse отправляет ответ через Kafka
func (a *Agent) sendResponse(msg *protocol.Message) error {
	// В Kafka-only архитектуре ответы отправляются через commandConsumer
	if a.useKafka && a.commandConsumer != nil {
		return a.commandConsumer.SendResponse(a.ctx, msg.ID, msg)
	}

	return fmt.Errorf("Kafka не доступен для отправки ответа")
}

// sendResponseToCommand отправляет ответ через Kafka
func (a *Agent) sendResponseToCommand(msg *protocol.Message, commandID string) error {
	// В Kafka-only архитектуре ответы отправляются через commandConsumer
	if a.useKafka && a.commandConsumer != nil {
		return a.commandConsumer.SendResponse(a.ctx, commandID, msg)
	}

	return fmt.Errorf("Kafka не доступен для отправки ответа")
}
