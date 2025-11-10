package bot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/servereye/servereye/pkg/redis"
)

// getCPUTemperature requests CPU temperature from agent via Streams
func (b *Bot) getCPUTemperature(serverKey string) (float64, error) {
	type tempResponse struct {
		Temperature float64 `json:"temperature"`
	}

	result, err := sendCommandAndParse[tempResponse](
		b,
		serverKey,
		protocol.TypeGetCPUTemp,
		nil,
		protocol.TypeCPUTempResponse,
		10*time.Second,
	)
	if err != nil {
		return 0, err
	}

	return result.Temperature, nil
}

// getContainers requests Docker containers list from agent
func (b *Bot) getContainers(serverKey string) (*protocol.ContainersPayload, error) {
	// Try Streams first if available
	if b.streamsClient != nil {
		containers, err := b.getContainersViaStreams(serverKey)
		if err == nil {
			return containers, nil
		}
		b.logger.Error("Streams failed, using Pub/Sub", err)
	}

	// Fallback to Pub/Sub
	return b.getContainersViaPubSub(serverKey)
}

// getContainersViaPubSub is the old Pub/Sub implementation
func (b *Bot) getContainersViaPubSub(serverKey string) (*protocol.ContainersPayload, error) {
	// Create command message first to get ID
	cmd := protocol.NewMessage(protocol.TypeGetContainers, nil)

	// Subscribe to UNIQUE response channel with command ID
	respChannel := fmt.Sprintf("resp:%s:%s", serverKey, cmd.ID)
	b.logger.Info("Подписались на канал Redis")

	subscription, err := b.redisClient.Subscribe(b.ctx, respChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response: %v", err)
	}
	defer func() {
		if subscription != nil {
			if err := subscription.Close(); err != nil {
				b.logger.Error("Failed to close subscription", err)
			}
		}
	}()

	// Send command to agent
	cmdChannel := redis.GetCommandChannel(serverKey)
	data, err := cmd.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %v", err)
	}

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
	result.WriteString(fmt.Sprintf("🐳 Docker Containers (%d total):\n\n", containers.Total))

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

		result.WriteString(fmt.Sprintf("%s %s\n", statusEmoji, container.Name))
		result.WriteString(fmt.Sprintf("📷 Image: `%s`\n", container.Image))
		result.WriteString(fmt.Sprintf("🔄 Status: %s\n", container.Status))

		if len(container.Ports) > 0 {
			result.WriteString(fmt.Sprintf("🔌 Ports: %s\n", strings.Join(container.Ports, ", ")))
		}

		result.WriteString("\n")
	}

	return result.String()
}

// getMemoryInfo requests memory information from agent via Streams
func (b *Bot) getMemoryInfo(serverKey string) (*protocol.MemoryInfo, error) {
	return sendCommandAndParse[protocol.MemoryInfo](
		b,
		serverKey,
		protocol.TypeGetMemoryInfo,
		nil,
		protocol.TypeMemoryInfoResponse,
		10*time.Second,
	)
}

// getDiskInfo requests disk information from agent via Streams
func (b *Bot) getDiskInfo(serverKey string) (*protocol.DiskInfoPayload, error) {
	return sendCommandAndParse[protocol.DiskInfoPayload](
		b,
		serverKey,
		protocol.TypeGetDiskInfo,
		nil,
		protocol.TypeDiskInfoResponse,
		10*time.Second,
	)
}

// getUptime requests uptime information from agent via Streams
func (b *Bot) getUptime(serverKey string) (*protocol.UptimeInfo, error) {
	return sendCommandAndParse[protocol.UptimeInfo](
		b,
		serverKey,
		protocol.TypeGetUptime,
		nil,
		protocol.TypeUptimeResponse,
		10*time.Second,
	)
}

// getProcesses requests processes information from agent via Streams
func (b *Bot) getProcesses(serverKey string) (*protocol.ProcessesPayload, error) {
	return sendCommandAndParse[protocol.ProcessesPayload](
		b,
		serverKey,
		protocol.TypeGetProcesses,
		nil,
		protocol.TypeProcessesResponse,
		10*time.Second,
	)
}

// getNetworkInfo requests network information from agent via Streams
func (b *Bot) getNetworkInfo(serverKey string) (*protocol.NetworkInfo, error) {
	return sendCommandAndParse[protocol.NetworkInfo](
		b,
		serverKey,
		protocol.TypeGetNetworkInfo,
		nil,
		protocol.TypeNetworkInfoResponse,
		10*time.Second,
	)
}

// updateAgent requests agent to update itself
func (b *Bot) updateAgent(serverKey string, version string) (*protocol.UpdateAgentResponse, error) {
	payload := &protocol.UpdateAgentPayload{
		Version: version,
	}

	return sendCommandAndParse[protocol.UpdateAgentResponse](
		b,
		serverKey,
		protocol.TypeUpdateAgent,
		payload,
		protocol.TypeUpdateAgentResponse,
		30*time.Second,
	)
}
