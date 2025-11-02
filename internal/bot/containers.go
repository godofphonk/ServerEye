package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/servereye/servereye/pkg/protocol"
)

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
	// Создаем команду сначала чтобы получить ID
	message := protocol.NewMessage(messageType, payload)

	// Подписываемся на УНИКАЛЬНЫЙ канал ответов с ID команды
	responseChannel := fmt.Sprintf("resp:%s:%s", serverKey, message.ID)
	subscription, err := b.redisClient.Subscribe(b.ctx, responseChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response channel: %w", err)
	}
	defer subscription.Close()

	b.logger.Info("Operation completed")

	// Небольшая задержка для стабилизации подписки (уменьшена с 1s до 100ms)
	time.Sleep(100 * time.Millisecond)

	// Отправляем команду
	commandChannel := fmt.Sprintf("cmd:%s", serverKey)

	messageData, err := message.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	if err := b.redisClient.Publish(b.ctx, commandChannel, messageData); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	b.logger.Info("Команда отправлена агенту")

	// Ожидаем ответ (увеличенный timeout для медленных операций + Docker pull time)
	timeout := 60 * time.Second
	if payload.Action == "stop" || payload.Action == "restart" || payload.Action == "remove" {
		timeout = 90 * time.Second // Stop, restart и remove могут быть очень долгими
	}
	ctx, cancel := context.WithTimeout(b.ctx, timeout)
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

	var actionEmoji, actionText string
	switch response.Action {
	case "start":
		actionEmoji = "▶️"
		actionText = "started"
	case "stop":
		actionEmoji = "⏹️"
		actionText = "stopped"
	case "restart":
		actionEmoji = "🔄"
		actionText = "restarted"
	case "remove":
		actionEmoji = "🗑️"
		actionText = "deleted"
	default:
		actionEmoji = "⚙️"
		actionText = response.Action + "ed"
	}

	result := fmt.Sprintf("✅ %s Container **%s** successfully %s!",
		actionEmoji, response.ContainerName, actionText)

	if response.NewState != "" && response.Action != "remove" {
		var stateEmoji string
		switch response.NewState {
		case "running":
			stateEmoji = "🟢"
		case "exited":
			stateEmoji = "🔴"
		default:
			stateEmoji = "🟡"
		}
		result += fmt.Sprintf("\n\n%s **Status:** %s", stateEmoji, response.NewState)
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
	// Защищаем только критичные контейнеры ServerEye инфраструктуры
	if action != "remove" && action != "start" {
		// Exact match для защищенных контейнеров
		protectedContainers := []string{
			"servereye-docker-servereye-bot-1",
			"servereye-docker-redis-1",
			"servereye-docker-postgres-1",
		}

		containerLower := strings.ToLower(containerID)
		for _, protected := range protectedContainers {
			if containerLower == strings.ToLower(protected) {
				return fmt.Errorf("Container '%s' is critical infrastructure and cannot be stopped/restarted", containerID)
			}
		}
	}

	return nil
}

// createContainerFromTemplate creates a container from predefined template
func (b *Bot) createContainerFromTemplate(userID int64, serverKey, template string) string {
	b.logger.Info("Creating container from template")

	// Get template configuration
	payload, err := b.getTemplateConfig(template)
	if err != nil {
		return fmt.Sprintf("❌ Unknown template: %s", template)
	}

	// Get user's servers
	servers, err := b.getUserServers(userID)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return "❌ Error getting your servers"
	}

	if len(servers) == 0 {
		return "❌ No servers found"
	}

	// Use first server
	serverKey = servers[0]

	// Send create container command
	response, err := b.sendCreateContainerCommand(serverKey, payload)
	if err != nil {
		b.logger.Error("Error occurred", err)
		return fmt.Sprintf("❌ Failed to create container: %v", err)
	}

	// Format response
	if response.Success {
		return fmt.Sprintf("✅ Container **%s** created successfully!\n\n📷 Image: `%s`\n🔄 Status: %s",
			response.ContainerName, payload.Image, response.Message)
	}
	return fmt.Sprintf("❌ Failed to create container: %s", response.Message)
}

// getTemplateConfig returns container configuration for a template
func (b *Bot) getTemplateConfig(template string) (*protocol.CreateContainerPayload, error) {
	// Use "0" for host port to let Docker choose a random available port
	templates := map[string]*protocol.CreateContainerPayload{
		"nginx": {
			Image: "nginx:latest",
			Name:  fmt.Sprintf("nginx-web-%d", time.Now().Unix()),
			Ports: map[string]string{"80/tcp": "0"}, // Random port
		},
		"postgres": {
			Image: "postgres:15",
			Name:  fmt.Sprintf("postgres-db-%d", time.Now().Unix()),
			Ports: map[string]string{"5432/tcp": "0"}, // Random port
			Environment: map[string]string{
				"POSTGRES_PASSWORD": "changeme123",
				"POSTGRES_DB":       "myapp",
			},
		},
		"redis": {
			Image: "redis:alpine",
			Name:  fmt.Sprintf("redis-cache-%d", time.Now().Unix()),
			Ports: map[string]string{"6379/tcp": "0"}, // Random port
		},
		"mongo": {
			Image: "mongo:latest",
			Name:  fmt.Sprintf("mongodb-%d", time.Now().Unix()),
			Ports: map[string]string{"27017/tcp": "0"}, // Random port
			Environment: map[string]string{
				"MONGO_INITDB_ROOT_USERNAME": "admin",
				"MONGO_INITDB_ROOT_PASSWORD": "changeme123",
			},
		},
		"rabbitmq": {
			Image: "rabbitmq:3-management",
			Name:  fmt.Sprintf("rabbitmq-%d", time.Now().Unix()),
			Ports: map[string]string{
				"5672/tcp":  "0", // Random port
				"15672/tcp": "0", // Random port
			},
		},
		"mysql": {
			Image: "mysql:8",
			Name:  fmt.Sprintf("mysql-db-%d", time.Now().Unix()),
			Ports: map[string]string{"3306/tcp": "0"}, // Random port
			Environment: map[string]string{
				"MYSQL_ROOT_PASSWORD": "changeme123",
				"MYSQL_DATABASE":      "myapp",
			},
		},
	}

	config, ok := templates[template]
	if !ok {
		return nil, fmt.Errorf("template not found")
	}

	return config, nil
}

// sendCreateContainerCommand sends create container command to agent
func (b *Bot) sendCreateContainerCommand(serverKey string, payload *protocol.CreateContainerPayload) (*protocol.ContainerActionResponse, error) {
	// Создаем команду сначала чтобы получить ID
	message := protocol.NewMessage(protocol.TypeCreateContainer, payload)

	// Подписываемся на УНИКАЛЬНЫЙ канал ответов с ID команды
	responseChannel := fmt.Sprintf("resp:%s:%s", serverKey, message.ID)
	subscription, err := b.redisClient.Subscribe(b.ctx, responseChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}
	defer subscription.Close()

	// Небольшая задержка для стабилизации подписки (уменьшена до 100ms)
	time.Sleep(100 * time.Millisecond)

	// Send command
	commandChannel := fmt.Sprintf("cmd:%s", serverKey)

	messageData, err := message.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize: %w", err)
	}

	if err := b.redisClient.Publish(b.ctx, commandChannel, messageData); err != nil {
		return nil, fmt.Errorf("failed to send: %w", err)
	}

	b.logger.Info("Create command sent to agent")

	// Wait for response (increased timeout for Docker pull + container creation)
	ctx, cancel := context.WithTimeout(b.ctx, 120*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for response")
		case respData := <-subscription.Channel():
			resp, err := protocol.FromJSON(respData)
			if err != nil {
				continue
			}

			// Parse response
			if resp.Type == protocol.TypeContainerActionResponse {
				payloadData, _ := json.Marshal(resp.Payload)
				var actionResponse protocol.ContainerActionResponse
				if err := json.Unmarshal(payloadData, &actionResponse); err == nil {
					return &actionResponse, nil
				}
			}

			if resp.Type == protocol.TypeErrorResponse {
				payloadData, _ := json.Marshal(resp.Payload)
				var errorPayload protocol.ErrorPayload
				if err := json.Unmarshal(payloadData, &errorPayload); err == nil {
					return nil, fmt.Errorf("%s", errorPayload.ErrorMessage)
				}
			}
		}
	}
}
