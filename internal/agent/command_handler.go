package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// HandleCommand реализует интерфейс CommandHandler для Kafka
func (a *Agent) HandleCommand(ctx context.Context, command *protocol.Message) (*protocol.Message, error) {
	a.logger.WithFields(logrus.Fields{
		"command_id":   command.ID,
		"command_type": command.Type,
		"server_id":    command.ServerID,
	}).Info("Handling command via Kafka")

	// Создаем базовый response
	response := &protocol.Message{
		ID:        command.ID,
		Timestamp: time.Now(),
		ServerID:  a.config.Server.Name,
		ServerKey: a.config.Server.SecretKey,
	}

	// Обрабатываем команду в зависимости от типа
	switch command.Type {
	case protocol.TypeGetCPUTemp:
		return a.handleTemperatureCommand(ctx, command, response)
	case protocol.TypeGetSystemInfo:
		return a.handleMemoryCommand(ctx, command, response)
	case protocol.TypeGetDiskInfo:
		return a.handleDiskCommand(ctx, command, response)
	case protocol.TypeGetUptime:
		return a.handleUptimeCommand(ctx, command, response)
	case protocol.TypeGetProcesses:
		return a.handleProcessesCommand(ctx, command, response)
	case protocol.TypeGetContainers:
		return a.handleContainersCommand(ctx, command, response)
	default:
		response.Type = protocol.TypeErrorResponse
		response.Payload = map[string]string{
			"error": fmt.Sprintf("неизвестный тип команды: %s", command.Type),
		}
		return response, nil
	}
}

// handleTemperatureCommand обрабатывает запрос температуры
func (a *Agent) handleTemperatureCommand(ctx context.Context, command *protocol.Message, response *protocol.Message) (*protocol.Message, error) {
	temp, err := a.cpuMetrics.GetTemperature()
	if err != nil {
		response.Type = protocol.TypeErrorResponse
		response.Payload = map[string]string{
			"error": fmt.Sprintf("не удалось получить температуру: %v", err),
		}
		return response, nil
	}

	response.Type = protocol.TypeCPUTempResponse
	response.Payload = map[string]interface{}{
		"temperature": temp,
		"timestamp":   time.Now().Unix(),
	}

	return response, nil
}

// handleMemoryCommand обрабатывает запрос памяти
func (a *Agent) handleMemoryCommand(ctx context.Context, command *protocol.Message, response *protocol.Message) (*protocol.Message, error) {
	memInfo, err := a.systemMonitor.GetMemoryInfo()
	if err != nil {
		response.Type = protocol.TypeErrorResponse
		response.Payload = map[string]string{
			"error": fmt.Sprintf("не удалось получить информацию о памяти: %v", err),
		}
		return response, nil
	}

	response.Type = protocol.TypeSystemInfoResponse
	response.Payload = memInfo

	return response, nil
}

// handleDiskCommand обрабатывает запрос диска
func (a *Agent) handleDiskCommand(ctx context.Context, command *protocol.Message, response *protocol.Message) (*protocol.Message, error) {
	diskInfo, err := a.systemMonitor.GetDiskInfo()
	if err != nil {
		response.Type = protocol.TypeErrorResponse
		response.Payload = map[string]string{
			"error": fmt.Sprintf("не удалось получить информацию о диске: %v", err),
		}
		return response, nil
	}

	response.Type = protocol.TypeDiskInfoResponse
	response.Payload = diskInfo

	return response, nil
}

// handleUptimeCommand обрабатывает запрос uptime
func (a *Agent) handleUptimeCommand(ctx context.Context, command *protocol.Message, response *protocol.Message) (*protocol.Message, error) {
	uptime, err := a.systemMonitor.GetUptime()
	if err != nil {
		response.Type = protocol.TypeErrorResponse
		response.Payload = map[string]string{
			"error": fmt.Sprintf("не удалось получить uptime: %v", err),
		}
		return response, nil
	}

	response.Type = protocol.TypeUptimeResponse
	response.Payload = map[string]interface{}{
		"uptime_seconds":   uptime.Uptime,
		"uptime_formatted": formatUptime(int64(uptime.Uptime)),
		"timestamp":        time.Now().Unix(),
	}

	return response, nil
}

// handleProcessesCommand обрабатывает запрос процессов
func (a *Agent) handleProcessesCommand(ctx context.Context, command *protocol.Message, response *protocol.Message) (*protocol.Message, error) {
	// TODO: Реализовать получение списка процессов в SystemMonitor
	// Команда процессов временно отключена для Kafka migration
	response.Type = protocol.TypeErrorResponse
	response.Payload = map[string]string{
		"error": "processes command not yet implemented in Kafka migration",
	}
	return response, nil
}

// handleContainersCommand обрабатывает запрос контейнеров
func (a *Agent) handleContainersCommand(ctx context.Context, command *protocol.Message, response *protocol.Message) (*protocol.Message, error) {
	containers, err := a.dockerClient.GetContainers(ctx)
	if err != nil {
		response.Type = protocol.TypeErrorResponse
		response.Payload = map[string]string{
			"error": fmt.Sprintf("не удалось получить список контейнеров: %v", err),
		}
		return response, nil
	}

	response.Type = protocol.TypeContainersResponse
	response.Payload = map[string]interface{}{
		"containers": containers.Containers,
		"count":      len(containers.Containers),
		"timestamp":  time.Now().Unix(),
	}

	return response, nil
}

// formatUptime форматирует uptime в человекочитаемый вид
func formatUptime(seconds int64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dд %dч %dм", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dч %dм", hours, minutes)
	} else {
		return fmt.Sprintf("%dм", minutes)
	}
}
