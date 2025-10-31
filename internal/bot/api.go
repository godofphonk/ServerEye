package bot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/servereye/servereye/pkg/redis"
)

// getCPUTemperature requests CPU temperature from agent
func (b *Bot) getCPUTemperature(serverKey string) (float64, error) {
	// Create command message
	cmd := protocol.NewMessage(protocol.TypeGetCPUTemp, nil)
	data, err := cmd.ToJSON()
	if err != nil {
		return 0, fmt.Errorf("failed to serialize command: %v", err)
	}

	// Subscribe to response channel first
	respChannel := redis.GetResponseChannel(serverKey)
	b.logger.Info("Подписались на канал Redis")

	subscription, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return 0, fmt.Errorf("failed to subscribe to response: %v", err)
	}
	defer subscription.Close()

	// Small delay to ensure subscription is active
	time.Sleep(100 * time.Millisecond)

	// Send command to agent
	cmdChannel := redis.GetCommandChannel(serverKey)
	if err := b.redisClient.Publish(b.ctx, cmdChannel, data); err != nil {
		return 0, fmt.Errorf("failed to send command: %v", err)
	}

	b.logger.Info("Команда отправлена агенту")

	// Wait for response with timeout
	timeout := time.After(10 * time.Second)
	for {
		select {
		case respData := <-subscription.Channel():
			b.logger.Debug("Получен ответ от агента")

			resp, err := protocol.FromJSON(respData)
			if err != nil {
				b.logger.Error("Failed to parse response", err)
				continue
			}

			// Check if this response is for our command
			if resp.ID != cmd.ID {
				b.logger.Debug("Response ID mismatch, waiting for correct response")
				continue
			}

			if resp.Type == protocol.TypeErrorResponse {
				return 0, fmt.Errorf("agent error: %v", resp.Payload)
			}

			if resp.Type == protocol.TypeCPUTempResponse {
				// Parse temperature from payload
				if payload, ok := resp.Payload.(map[string]interface{}); ok {
					if temp, ok := payload["temperature"].(float64); ok {
						b.logger.Info("Получена температура CPU")
						return temp, nil
					}
				}
				return 0, fmt.Errorf("invalid temperature data in response")
			}

			return 0, fmt.Errorf("unexpected response type: %s", resp.Type)

		case <-timeout:
			return 0, fmt.Errorf("timeout waiting for response")
		}
	}
}

// getContainers requests Docker containers list from agent
func (b *Bot) getContainers(serverKey string) (*protocol.ContainersPayload, error) {
	// Create command message
	cmd := protocol.NewMessage(protocol.TypeGetContainers, nil)
	data, err := cmd.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %v", err)
	}

	// Subscribe to response channel first
	respChannel := redis.GetResponseChannel(serverKey)
	b.logger.Info("Подписались на канал Redis")
	
	subscription, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response: %v", err)
	}

	// Small delay to ensure subscription is active
	time.Sleep(100 * time.Millisecond)

	// Send command to agent
	cmdChannel := redis.GetCommandChannel(serverKey)
	if err := b.redisClient.Publish(b.ctx, cmdChannel, data); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	b.logger.Info("Команда отправлена агенту")

	// Wait for response with timeout
	timeout := time.After(10 * time.Second)
	for {
		select {
		case respData := <-subscription.Channel():
			b.logger.Debug("Получен ответ от агента")
			
			resp, err := protocol.FromJSON(respData)
			if err != nil {
				b.logger.Error("Failed to parse response", err)
				continue
			}

			// Check if this response is for our command
			if resp.ID != cmd.ID {
				b.logger.Debug("Response ID mismatch, waiting for correct response")
				continue
			}

			if resp.Type == protocol.TypeErrorResponse {
				return nil, fmt.Errorf("agent error: %v", resp.Payload)
			}

			if resp.Type == protocol.TypeContainersResponse {
				// Parse containers from payload
				if payload, ok := resp.Payload.(map[string]interface{}); ok {
					containersData, _ := json.Marshal(payload)
					var containers protocol.ContainersPayload
					if err := json.Unmarshal(containersData, &containers); err == nil {
						b.logger.Info("Получен список контейнеров")
						return &containers, nil
					}
				}
				return nil, fmt.Errorf("invalid containers data in response")
			}

			return nil, fmt.Errorf("unexpected response type: %s", resp.Type)

		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for response")
		}
	}
}

// formatContainers formats containers list for display
func (b *Bot) formatContainers(containers *protocol.ContainersPayload) string {
	if containers.Total == 0 {
		return "📦 No Docker containers found on the server."
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("🐳 **Docker Containers (%d total):**\n\n", containers.Total))

	for i, container := range containers.Containers {
		if i >= 10 { // Limit to 10 containers to avoid message length issues
			result.WriteString(fmt.Sprintf("... and %d more containers\n", containers.Total-10))
			break
		}

		// Status emoji
		statusEmoji := "🔴" // Red for stopped
		if strings.Contains(strings.ToLower(container.State), "running") {
			statusEmoji = "🟢" // Green for running
		} else if strings.Contains(strings.ToLower(container.State), "paused") {
			statusEmoji = "🟡" // Yellow for paused
		}

		result.WriteString(fmt.Sprintf("%s **%s**\n", statusEmoji, container.Name))
		result.WriteString(fmt.Sprintf("📷 Image: `%s`\n", container.Image))
		result.WriteString(fmt.Sprintf("🔄 Status: %s\n", container.Status))
		
		if len(container.Ports) > 0 {
			result.WriteString(fmt.Sprintf("🔌 Ports: %s\n", strings.Join(container.Ports, ", ")))
		}
		
		result.WriteString("\n")
	}

	return result.String()
}

// getMemoryInfo requests memory information from agent
func (b *Bot) getMemoryInfo(serverKey string) (*protocol.MemoryInfo, error) {
	cmd := protocol.NewMessage(protocol.TypeGetMemoryInfo, nil)
	data, err := cmd.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %v", err)
	}

	respChannel := redis.GetResponseChannel(serverKey)
	subscription, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response: %v", err)
	}
	defer subscription.Close()

	time.Sleep(100 * time.Millisecond)

	cmdChannel := redis.GetCommandChannel(serverKey)
	if err := b.redisClient.Publish(b.ctx, cmdChannel, data); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	timeout := time.After(10 * time.Second)
	for {
		select {
		case respData := <-subscription.Channel():
			resp, err := protocol.FromJSON(respData)
			if err != nil {
				continue
			}

			if resp.ID != cmd.ID {
				continue
			}

			if resp.Type == protocol.TypeErrorResponse {
				return nil, fmt.Errorf("agent error: %v", resp.Payload)
			}

			if resp.Type == protocol.TypeMemoryInfoResponse {
				if payload, ok := resp.Payload.(map[string]interface{}); ok {
					memData, _ := json.Marshal(payload)
					var memInfo protocol.MemoryInfo
					if err := json.Unmarshal(memData, &memInfo); err == nil {
						return &memInfo, nil
					}
				}
				return nil, fmt.Errorf("invalid memory data in response")
			}

			return nil, fmt.Errorf("unexpected response type: %s", resp.Type)

		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for response")
		}
	}
}

// getDiskInfo requests disk information from agent
func (b *Bot) getDiskInfo(serverKey string) (*protocol.DiskInfoPayload, error) {
	cmd := protocol.NewMessage(protocol.TypeGetDiskInfo, nil)
	data, err := cmd.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %v", err)
	}

	respChannel := redis.GetResponseChannel(serverKey)
	subscription, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response: %v", err)
	}
	defer subscription.Close()

	time.Sleep(100 * time.Millisecond)

	cmdChannel := redis.GetCommandChannel(serverKey)
	if err := b.redisClient.Publish(b.ctx, cmdChannel, data); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	timeout := time.After(10 * time.Second)
	for {
		select {
		case respData := <-subscription.Channel():
			resp, err := protocol.FromJSON(respData)
			if err != nil {
				continue
			}

			if resp.ID != cmd.ID {
				continue
			}

			if resp.Type == protocol.TypeErrorResponse {
				return nil, fmt.Errorf("agent error: %v", resp.Payload)
			}

			if resp.Type == protocol.TypeDiskInfoResponse {
				if payload, ok := resp.Payload.(map[string]interface{}); ok {
					diskData, _ := json.Marshal(payload)
					var diskInfo protocol.DiskInfoPayload
					if err := json.Unmarshal(diskData, &diskInfo); err == nil {
						return &diskInfo, nil
					}
				}
				return nil, fmt.Errorf("invalid disk data in response")
			}

			return nil, fmt.Errorf("unexpected response type: %s", resp.Type)

		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for response")
		}
	}
}

// getUptime requests uptime information from agent
func (b *Bot) getUptime(serverKey string) (*protocol.UptimeInfo, error) {
	cmd := protocol.NewMessage(protocol.TypeGetUptime, nil)
	data, err := cmd.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %v", err)
	}

	respChannel := redis.GetResponseChannel(serverKey)
	subscription, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response: %v", err)
	}
	defer subscription.Close()

	time.Sleep(100 * time.Millisecond)

	cmdChannel := redis.GetCommandChannel(serverKey)
	if err := b.redisClient.Publish(b.ctx, cmdChannel, data); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	timeout := time.After(10 * time.Second)
	for {
		select {
		case respData := <-subscription.Channel():
			resp, err := protocol.FromJSON(respData)
			if err != nil {
				continue
			}

			if resp.ID != cmd.ID {
				continue
			}

			if resp.Type == protocol.TypeErrorResponse {
				return nil, fmt.Errorf("agent error: %v", resp.Payload)
			}

			if resp.Type == protocol.TypeUptimeResponse {
				if payload, ok := resp.Payload.(map[string]interface{}); ok {
					uptimeData, _ := json.Marshal(payload)
					var uptimeInfo protocol.UptimeInfo
					if err := json.Unmarshal(uptimeData, &uptimeInfo); err == nil {
						return &uptimeInfo, nil
					}
				}
				return nil, fmt.Errorf("invalid uptime data in response")
			}

			return nil, fmt.Errorf("unexpected response type: %s", resp.Type)

		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for response")
		}
	}
}

// getProcesses requests processes information from agent
func (b *Bot) getProcesses(serverKey string) (*protocol.ProcessesPayload, error) {
	cmd := protocol.NewMessage(protocol.TypeGetProcesses, nil)
	data, err := cmd.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %v", err)
	}

	respChannel := redis.GetResponseChannel(serverKey)
	subscription, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response: %v", err)
	}
	defer subscription.Close()

	time.Sleep(100 * time.Millisecond)

	cmdChannel := redis.GetCommandChannel(serverKey)
	if err := b.redisClient.Publish(b.ctx, cmdChannel, data); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	timeout := time.After(10 * time.Second)
	for {
		select {
		case respData := <-subscription.Channel():
			resp, err := protocol.FromJSON(respData)
			if err != nil {
				continue
			}

			if resp.ID != cmd.ID {
				continue
			}

			if resp.Type == protocol.TypeErrorResponse {
				return nil, fmt.Errorf("agent error: %v", resp.Payload)
			}

			if resp.Type == protocol.TypeProcessesResponse {
				if payload, ok := resp.Payload.(map[string]interface{}); ok {
					processData, _ := json.Marshal(payload)
					var processes protocol.ProcessesPayload
					if err := json.Unmarshal(processData, &processes); err == nil {
						return &processes, nil
					}
				}
				return nil, fmt.Errorf("invalid processes data in response")
			}

			return nil, fmt.Errorf("unexpected response type: %s", resp.Type)

		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for response")
		}
	}
}
