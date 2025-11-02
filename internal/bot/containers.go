package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/servereye/servereye/pkg/protocol"
)

// handleStartContainer handles the /start_container command
func (b *Bot) handleStartContainer(message *tgbotapi.Message) string {
	b.logger.Info("Operation completed")

	// Парсим команду
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		return "❌ Usage: /start_container <container_id_or_name>\n\nExample: /start_container nginx"
	}

	containerID := parts[1]
	return b.handleContainerAction(message.From.ID, containerID, "start")
}

// handleStopContainer handles the /stop_container command
func (b *Bot) handleStopContainer(message *tgbotapi.Message) string {
	b.logger.Info("Operation completed")

	// Парсим команду
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		return "❌ Usage: /stop_container <container_id_or_name>\n\nExample: /stop_container nginx"
	}

	containerID := parts[1]
	return b.handleContainerAction(message.From.ID, containerID, "stop")
}

// handleRestartContainer handles the /restart_container command
func (b *Bot) handleRestartContainer(message *tgbotapi.Message) string {
	b.logger.Info("Operation completed")

	// Парсим команду
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		return "❌ Usage: /restart_container <container_id_or_name>\n\nExample: /restart_container nginx"
	}

	containerID := parts[1]
	return b.handleContainerAction(message.From.ID, containerID, "restart")
}

// handleContainerAction handles container management actions
func (b *Bot) handleContainerAction(userID int64, containerID, action string) string {
	// Валидация входных данных
	if err := b.validateContainerAction(containerID, action); err != nil {
		return fmt.Sprintf("❌ %s", err.Error())
	}

	// Получаем серверы пользователя
	servers, err := b.getUserServers(userID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return "❌ Error getting your servers. Please try again."
	}

	if len(servers) == 0 {
		return "❌ You don't have any connected servers. Use /add to connect a server first."
	}

	b.logger.Info("Найдено серверов пользователя")

	// Пока работаем только с первым сервером
	serverKey := servers[0]
	b.logger.Info("Operation completed")

	// Определяем тип команды
	var messageType protocol.MessageType
	switch action {
	case "start":
		messageType = protocol.TypeStartContainer
	case "stop":
		messageType = protocol.TypeStopContainer
	case "restart":
		messageType = protocol.TypeRestartContainer
	case "remove":
		messageType = protocol.TypeRemoveContainer
	default:
		return fmt.Sprintf("❌ Invalid action: %s", action)
	}

	// Создаем payload
	payload := protocol.ContainerActionPayload{
		ContainerID:   containerID,
		ContainerName: containerID, // Может быть именем или ID
		Action:        action,
	}

	response, err := b.sendContainerAction(serverKey, messageType, payload)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return fmt.Sprintf("❌ Failed to %s container: %v", action, err)
	}

	b.logger.Info("Operation completed")
	return b.formatContainerActionResponse(response)
}

// sendContainerAction sends container action command to agent
func (b *Bot) sendContainerAction(serverKey string, messageType protocol.MessageType, payload protocol.ContainerActionPayload) (*protocol.ContainerActionResponse, error) {
	// Подписываемся на канал ответов
	responseChannel := fmt.Sprintf("resp:%s", serverKey)
	subscription, err := b.redisClient.Subscribe(b.ctx, responseChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response channel: %w", err)
	}
	defer subscription.Close()

	b.logger.Info("Operation completed")

	// Отправляем команду
	message := protocol.NewMessage(messageType, payload)
	commandChannel := fmt.Sprintf("cmd:%s", serverKey)

	messageData, err := message.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	if err := b.redisClient.Publish(b.ctx, commandChannel, messageData); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	b.logger.Info("Команда отправлена агенту")

	// Ожидаем ответ
	ctx, cancel := context.WithTimeout(b.ctx, 30*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for agent response")
		case msgBytes := <-subscription.Channel():
			var response protocol.Message
			if err := json.Unmarshal(msgBytes, &response); err != nil {
				b.logger.Error("Error occurred", err)
				continue
			}

			if response.ID != message.ID {
				continue // Не наш ответ
			}

			if response.Type == protocol.TypeErrorResponse {
				// Парсим ошибку
				if errorData, ok := response.Payload.(map[string]interface{}); ok {
					errorMsg := "unknown error"
					if msg, exists := errorData["error_message"]; exists {
						errorMsg = fmt.Sprintf("%v", msg)
					}
					return nil, fmt.Errorf("agent error: %s", errorMsg)
				}
				return nil, fmt.Errorf("agent returned error")
			}

			if response.Type == protocol.TypeContainerActionResponse {
				// Парсим ответ
				if payload, ok := response.Payload.(map[string]interface{}); ok {
					actionData, _ := json.Marshal(payload)
					var actionResponse protocol.ContainerActionResponse
					if err := json.Unmarshal(actionData, &actionResponse); err == nil {
						b.logger.Info("Operation completed")
						return &actionResponse, nil
					}
				}
				return nil, fmt.Errorf("invalid container action response format")
			}
		}
	}
}

// formatContainerActionResponse formats container action response for display
func (b *Bot) formatContainerActionResponse(response *protocol.ContainerActionResponse) string {
	if !response.Success {
		return fmt.Sprintf("❌ Failed to %s container **%s**:\n%s",
			response.Action, response.ContainerName, response.Message)
	}

	var actionEmoji string
	switch response.Action {
	case "start":
		actionEmoji = "▶️"
	case "stop":
		actionEmoji = "⏹️"
	case "restart":
		actionEmoji = "🔄"
	default:
		actionEmoji = "⚙️"
	}

	result := fmt.Sprintf("%s Successfully **%sed** container **%s**",
		actionEmoji, response.Action, response.ContainerName)

	if response.NewState != "" {
		var stateEmoji string
		switch response.NewState {
		case "running":
			stateEmoji = "🟢"
		case "exited":
			stateEmoji = "🔴"
		default:
			stateEmoji = "🟡"
		}
		result += fmt.Sprintf("\n%s New state: %s", stateEmoji, response.NewState)
	}

	return result
}

// validateContainerAction validates container action parameters
func (b *Bot) validateContainerAction(containerID, action string) error {
	// Проверяем длину ID контейнера
	if len(containerID) < 3 {
		return fmt.Errorf("Container ID/name too short (minimum 3 characters)")
	}

	if len(containerID) > 64 {
		return fmt.Errorf("Container ID/name too long (maximum 64 characters)")
	}

	// Проверяем символы в ID/имени
	for _, char := range containerID {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.') {
			return fmt.Errorf("Container ID/name contains invalid characters. Only alphanumeric, hyphens, underscores and dots allowed")
		}
	}

	// Проверяем действие
	validActions := map[string]bool{
		"start":   true,
		"stop":    true,
		"restart": true,
		"remove":  true,
	}

	if !validActions[action] {
		return fmt.Errorf("Invalid action '%s'. Allowed: start, stop, restart, remove", action)
	}

	// Проверяем черный список контейнеров для stop/restart (не для remove)
	if action != "remove" {
		blacklist := []string{
			"servereye-bot",
			"redis",
			"postgres",
			"postgresql",
			"database",
			"db",
		}

		containerLower := strings.ToLower(containerID)
		for _, blocked := range blacklist {
			if strings.Contains(containerLower, blocked) {
				return fmt.Errorf("Container '%s' is protected and cannot be stopped/restarted", containerID)
			}
		}
	}

	return nil
}
