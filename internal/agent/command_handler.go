package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// HandleCommand реализует интерфейс CommandHandler для HTTP API
func (a *Agent) HandleCommand(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	a.logger.WithFields(logrus.Fields{
		"command_id":   msg.ID,
		"command_type": msg.Type,
		"server_id":    a.config.Server.Name,
	}).Info("Handling command via HTTP API")

	// Создаем базовый response
	response := &protocol.Message{
		ID:        msg.ID,
		Timestamp: time.Now(),
		ServerID:  a.config.Server.Name,
		ServerKey: a.config.Server.SecretKey,
	}

	// Обрабатываем команду в зависимости от типа
	switch msg.Type {
	case protocol.TypeGetCPUTemp:
		return a.handleTemperatureCommand(ctx, msg, response)
	case protocol.TypeGetMemoryInfo:
		return a.handleMemoryCommand(ctx, msg, response)
	case protocol.TypeGetSystemInfo:
		return a.handleMemoryCommand(ctx, msg, response)
	case protocol.TypeGetDiskInfo:
		return a.handleDiskCommand(ctx, msg, response)
	case protocol.TypeGetUptime:
		return a.handleUptimeCommand(ctx, msg, response)
	case protocol.TypeGetProcesses:
		return a.handleProcessesCommand(ctx, msg, response)
	case protocol.TypeGetContainers:
		return a.handleContainersCommand(ctx, msg, response)
	default:
		response.Type = protocol.TypeErrorResponse
		response.Payload = map[string]string{
			"error": fmt.Sprintf("неизвестный тип команды: %s", msg.Type),
		}
		return response, nil
	}
}

// handleTemperatureCommand обрабатывает запрос температуры
func (a *Agent) handleTemperatureCommand(ctx context.Context, msg *protocol.Message, response *protocol.Message) (*protocol.Message, error) {
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
func (a *Agent) handleMemoryCommand(ctx context.Context, msg *protocol.Message, response *protocol.Message) (*protocol.Message, error) {
	memInfo, err := a.systemMonitor.GetMemoryInfo()
	if err != nil {
		response.Type = protocol.TypeErrorResponse
		response.Payload = map[string]string{
			"error": fmt.Sprintf("не удалось получить информацию о памяти: %v", err),
		}
		return response, nil
	}

	response.Type = protocol.TypeMemoryInfoResponse
	response.Payload = memInfo

	return response, nil
}

// handleDiskCommand обрабатывает запрос диска
func (a *Agent) handleDiskCommand(ctx context.Context, msg *protocol.Message, response *protocol.Message) (*protocol.Message, error) {
	diskInfo, err := a.systemMonitor.GetDiskInfo()
	if err != nil {
		response.Type = protocol.TypeErrorResponse
		response.Payload = map[string]string{
			"error": fmt.Sprintf("не удалось получить информацию о диске: %v", err),
		}
		return response, nil
	}

	response.Type = protocol.TypeDiskInfoResponse
	response.Payload = map[string]interface{}{
		"disks": diskInfo.Disks,
	}

	return response, nil
}

// handleUptimeCommand обрабатывает запрос uptime
func (a *Agent) handleUptimeCommand(ctx context.Context, msg *protocol.Message, response *protocol.Message) (*protocol.Message, error) {
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
		"uptime":    uptime.Uptime,
		"boot_time": uptime.BootTime,
		"formatted": formatUptime(int64(uptime.Uptime)),
	}

	return response, nil
}

// handleProcessesCommand обрабатывает запрос процессов
func (a *Agent) handleProcessesCommand(ctx context.Context, msg *protocol.Message, response *protocol.Message) (*protocol.Message, error) {
	// TODO: Реализовать получение списка процессов в SystemMonitor
	// Команда процессов временно отключена
	response.Type = protocol.TypeErrorResponse
	response.Payload = map[string]interface{}{
		"error": "processes command not yet implemented",
	}
	return response, nil
}

// handleContainersCommand обрабатывает запрос контейнеров
func (a *Agent) handleContainersCommand(ctx context.Context, msg *protocol.Message, response *protocol.Message) (*protocol.Message, error) {
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
		"total":      len(containers.Containers),
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
