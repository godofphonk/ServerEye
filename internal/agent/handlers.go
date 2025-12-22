package agent

import (
	"context"
	"fmt"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
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

// TODO: Uncomment when unknown command handling is needed
// handleUnknownCommand обрабатывает неизвестную команду
// func (a *Agent) handleUnknownCommand(msg *protocol.Message) *protocol.Message {
// 	payload := protocol.ErrorPayload{
// 		ErrorCode:    protocol.ErrorInvalidCommand,
// 		ErrorMessage: fmt.Sprintf("Неизвестная команда: %s", msg.Type),
// 	}
//
// 	response := protocol.NewMessage(protocol.TypeErrorResponse, payload)
// 	response.ID = msg.ID
// 	return response
// }

// handleGetCPUTemp обрабатывает запрос температуры CPU
func (a *Agent) handleGetCPUTemp(msg *protocol.Message) *protocol.Message {
	temp, err := a.cpuMetrics.GetTemperature()
	if err != nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorCommandTimeout,
			ErrorMessage: fmt.Sprintf("Failed to get CPU temperature: %v", err),
		})
	}

	response := protocol.NewMessage(protocol.TypeCPUTempResponse, map[string]interface{}{
		"temperature": temp,
		"timestamp":   msg.Timestamp,
	})
	response.ID = msg.ID
	return response
}

// handleGetContainers обрабатывает запрос списка контейнеров
func (a *Agent) handleGetContainers(msg *protocol.Message) *protocol.Message {
	if a.dockerClient == nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorDockerUnavailable,
			ErrorMessage: "Docker client not available",
		})
	}

	containers, err := a.dockerClient.GetContainers(context.Background())
	if err != nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorDockerUnavailable,
			ErrorMessage: fmt.Sprintf("Failed to get containers: %v", err),
		})
	}

	response := protocol.NewMessage(protocol.TypeContainersResponse, *containers)
	response.ID = msg.ID
	return response
}

// handleStartContainer обрабатывает запрос запуска контейнера
func (a *Agent) handleStartContainer(msg *protocol.Message) *protocol.Message {
	if a.dockerClient == nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorDockerUnavailable,
			ErrorMessage: "Docker client not available",
		})
	}

	// Extract container ID from payload
	// Log payload type and structure before parsing
	a.logger.WithFields(logrus.Fields{
		"payload_type":  fmt.Sprintf("%T", msg.Payload),
		"payload_value": fmt.Sprintf("%+v", msg.Payload),
	}).Info("Container action payload received")

	payloadData, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Invalid payload format",
		})
	}

	// Debug logging to see actual payload structure
	a.logger.WithField("payload", fmt.Sprintf("%+v", payloadData)).Info("Received container action payload")

	containerID, ok := payloadData["container_id"].(string)
	if !ok {
		// Try container_name as fallback
		containerID, ok = payloadData["container_name"].(string)
		if !ok {
			return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
				ErrorCode:    protocol.ErrorInvalidCommand,
				ErrorMessage: "Container ID not provided",
			})
		}
	}

	// Start container
	response, err := a.dockerClient.StartContainer(context.Background(), containerID)
	if err != nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorDockerUnavailable,
			ErrorMessage: fmt.Sprintf("Failed to start container: %v", err),
		})
	}

	return protocol.NewMessage(protocol.TypeContainerActionResponse, response)
}

// handleStopContainer обрабатывает запрос остановки контейнера
func (a *Agent) handleStopContainer(msg *protocol.Message) *protocol.Message {
	if a.dockerClient == nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorDockerUnavailable,
			ErrorMessage: "Docker client not available",
		})
	}

	// Extract container ID from payload
	// Log payload type and structure before parsing
	a.logger.WithFields(logrus.Fields{
		"payload_type":  fmt.Sprintf("%T", msg.Payload),
		"payload_value": fmt.Sprintf("%+v", msg.Payload),
	}).Info("Container action payload received")

	payloadData, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Invalid payload format",
		})
	}

	// Debug logging to see actual payload structure
	a.logger.WithField("payload", fmt.Sprintf("%+v", payloadData)).Info("Received container action payload")

	containerID, ok := payloadData["container_id"].(string)
	if !ok {
		// Try container_name as fallback
		containerID, ok = payloadData["container_name"].(string)
		if !ok {
			return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
				ErrorCode:    protocol.ErrorInvalidCommand,
				ErrorMessage: "Container ID not provided",
			})
		}
	}

	// Stop container
	response, err := a.dockerClient.StopContainer(context.Background(), containerID)
	if err != nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorDockerUnavailable,
			ErrorMessage: fmt.Sprintf("Failed to stop container: %v", err),
		})
	}

	return protocol.NewMessage(protocol.TypeContainerActionResponse, response)
}

// handleRestartContainer обрабатывает запрос перезапуска контейнера
func (a *Agent) handleRestartContainer(msg *protocol.Message) *protocol.Message {
	if a.dockerClient == nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorDockerUnavailable,
			ErrorMessage: "Docker client not available",
		})
	}

	// Extract container ID from payload
	// Log payload type and structure before parsing
	a.logger.WithFields(logrus.Fields{
		"payload_type":  fmt.Sprintf("%T", msg.Payload),
		"payload_value": fmt.Sprintf("%+v", msg.Payload),
	}).Info("Container action payload received")

	payloadData, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Invalid payload format",
		})
	}

	// Debug logging to see actual payload structure
	a.logger.WithField("payload", fmt.Sprintf("%+v", payloadData)).Info("Received container action payload")

	containerID, ok := payloadData["container_id"].(string)
	if !ok {
		// Try container_name as fallback
		containerID, ok = payloadData["container_name"].(string)
		if !ok {
			return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
				ErrorCode:    protocol.ErrorInvalidCommand,
				ErrorMessage: "Container ID not provided",
			})
		}
	}

	// Restart container
	response, err := a.dockerClient.RestartContainer(context.Background(), containerID)
	if err != nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorDockerUnavailable,
			ErrorMessage: fmt.Sprintf("Failed to restart container: %v", err),
		})
	}

	return protocol.NewMessage(protocol.TypeContainerActionResponse, response)
}

// handleRemoveContainer обрабатывает запрос удаления контейнера
func (a *Agent) handleRemoveContainer(msg *protocol.Message) *protocol.Message {
	if a.dockerClient == nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorDockerUnavailable,
			ErrorMessage: "Docker client not available",
		})
	}

	// Extract container ID from payload
	// Log payload type and structure before parsing
	a.logger.WithFields(logrus.Fields{
		"payload_type":  fmt.Sprintf("%T", msg.Payload),
		"payload_value": fmt.Sprintf("%+v", msg.Payload),
	}).Info("Container action payload received")

	payloadData, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Invalid payload format",
		})
	}

	// Debug logging to see actual payload structure
	a.logger.WithField("payload", fmt.Sprintf("%+v", payloadData)).Info("Received container action payload")

	containerID, ok := payloadData["container_id"].(string)
	if !ok {
		// Try container_name as fallback
		containerID, ok = payloadData["container_name"].(string)
		if !ok {
			return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
				ErrorCode:    protocol.ErrorInvalidCommand,
				ErrorMessage: "Container ID not provided",
			})
		}
	}

	// Remove container
	response, err := a.dockerClient.RemoveContainer(context.Background(), containerID)
	if err != nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorDockerUnavailable,
			ErrorMessage: fmt.Sprintf("Failed to remove container: %v", err),
		})
	}

	return protocol.NewMessage(protocol.TypeContainerActionResponse, response)
}

//
// func (a *Agent) handleCreateContainer(msg *protocol.Message) *protocol.Message {
// 	return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 		ErrorCode:    protocol.ErrorInvalidCommand,
// 		ErrorMessage: "Create container command not implemented",
// 	})
// }

// handleGetMemoryInfo обрабатывает запрос информации о памяти
func (a *Agent) handleGetMemoryInfo(msg *protocol.Message) *protocol.Message {
	if a.systemMonitor == nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorCommandTimeout,
			ErrorMessage: "System monitor not available",
		})
	}

	memInfo, err := a.systemMonitor.GetMemoryInfo()
	if err != nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorCommandTimeout,
			ErrorMessage: fmt.Sprintf("Failed to get memory info: %v", err),
		})
	}

	response := protocol.NewMessage(protocol.TypeMemoryInfoResponse, memInfo)
	response.ID = msg.ID
	return response
}

// handleGetDiskInfo обрабатывает запрос информации о диске
func (a *Agent) handleGetDiskInfo(msg *protocol.Message) *protocol.Message {
	if a.systemMonitor == nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorCommandTimeout,
			ErrorMessage: "System monitor not available",
		})
	}

	diskInfo, err := a.systemMonitor.GetDiskInfo()
	if err != nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorCommandTimeout,
			ErrorMessage: fmt.Sprintf("Failed to get disk info: %v", err),
		})
	}

	response := protocol.NewMessage(protocol.TypeDiskInfoResponse, map[string]interface{}{
		"disks": diskInfo.Disks,
	})
	response.ID = msg.ID
	return response
}

// handleGetUptime обрабатывает запрос uptime
func (a *Agent) handleGetUptime(msg *protocol.Message) *protocol.Message {
	if a.systemMonitor == nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorCommandTimeout,
			ErrorMessage: "System monitor not available",
		})
	}

	uptime, err := a.systemMonitor.GetUptime()
	if err != nil {
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorCommandTimeout,
			ErrorMessage: fmt.Sprintf("Failed to get uptime: %v", err),
		})
	}

	response := protocol.NewMessage(protocol.TypeUptimeResponse, map[string]interface{}{
		"uptime":    uptime.Uptime,
		"boot_time": uptime.BootTime,
	})
	response.ID = msg.ID
	return response
}

// handleGetProcesses обрабатывает запрос списка процессов
func (a *Agent) handleGetProcesses(msg *protocol.Message) *protocol.Message {
	return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
		ErrorCode:    protocol.ErrorInvalidCommand,
		ErrorMessage: "Get processes command not implemented",
	})
}

// handleGetNetworkInfo обрабатывает запрос информации о сети
func (a *Agent) handleGetNetworkInfo(msg *protocol.Message) *protocol.Message {
	return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
		ErrorCode:    protocol.ErrorInvalidCommand,
		ErrorMessage: "Get network info command not implemented",
	})
}

// handleUpdateAgent обрабатывает запрос обновления агента
func (a *Agent) handleUpdateAgent(msg *protocol.Message) *protocol.Message {
	return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
		ErrorCode:    protocol.ErrorInvalidCommand,
		ErrorMessage: "Update agent command not implemented",
	})
}
