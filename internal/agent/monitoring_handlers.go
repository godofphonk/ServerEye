package agent

import (
	"fmt"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// handleGetCPUTemp обрабатывает команду получения температуры CPU
func (a *Agent) handleGetCPUTemp(msg *protocol.Message) *protocol.Message {
	temp, err := a.cpuMetrics.GetTemperature()
	if err != nil {
		a.logger.WithError(err).Error("Не удалось получить температуру CPU")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorSensorNotFound,
			ErrorMessage: err.Error(),
		})
	}

	payload := protocol.CPUTempPayload{
		Temperature: temp,
		Unit:        "celsius",
		Sensor:      a.cpuMetrics.GetSensorInfo(),
	}

	response := protocol.NewMessage(protocol.TypeCPUTempResponse, payload)
	response.ID = msg.ID
	return response
}

// handleGetMemoryInfo обрабатывает команду получения информации о памяти
func (a *Agent) handleGetMemoryInfo(msg *protocol.Message) *protocol.Message {
	a.logger.Debug("Обработка команды получения информации о памяти")

	memInfo, err := a.systemMonitor.GetMemoryInfo()
	if err != nil {
		a.logger.WithError(err).Error("Ошибка получения информации о памяти")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    "MEMORY_INFO_ERROR",
			ErrorMessage: fmt.Sprintf("Ошибка получения информации о памяти: %v", err),
		})
	}

	a.logger.WithFields(logrus.Fields{
		"total_gb":     float64(memInfo.Total) / 1024 / 1024 / 1024,
		"used_percent": memInfo.UsedPercent,
	}).Info("Информация о памяти получена")

	response := protocol.NewMessage(protocol.TypeMemoryInfoResponse, memInfo)
	response.ID = msg.ID
	return response
}

// handleGetDiskInfo обрабатывает команду получения информации о дисках
func (a *Agent) handleGetDiskInfo(msg *protocol.Message) *protocol.Message {
	a.logger.Debug("Обработка команды получения информации о дисках")

	diskInfo, err := a.systemMonitor.GetDiskInfo()
	if err != nil {
		a.logger.WithError(err).Error("Ошибка получения информации о дисках")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    "DISK_INFO_ERROR",
			ErrorMessage: fmt.Sprintf("Ошибка получения информации о дисках: %v", err),
		})
	}

	a.logger.WithField("disks_count", len(diskInfo.Disks)).Info("Информация о дисках получена")
	response := protocol.NewMessage(protocol.TypeDiskInfoResponse, diskInfo)
	response.ID = msg.ID
	return response
}

// handleGetUptime обрабатывает команду получения времени работы системы
func (a *Agent) handleGetUptime(msg *protocol.Message) *protocol.Message {
	a.logger.Debug("Обработка команды получения времени работы")

	uptimeInfo, err := a.systemMonitor.GetUptime()
	if err != nil {
		a.logger.WithError(err).Error("Ошибка получения времени работы")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    "UPTIME_ERROR",
			ErrorMessage: fmt.Sprintf("Ошибка получения времени работы: %v", err),
		})
	}

	a.logger.WithField("uptime", uptimeInfo.Formatted).Info("Время работы получено")
	response := protocol.NewMessage(protocol.TypeUptimeResponse, uptimeInfo)
	response.ID = msg.ID
	return response
}

// handleGetProcesses обрабатывает команду получения списка процессов
func (a *Agent) handleGetProcesses(msg *protocol.Message) *protocol.Message {
	a.logger.Debug("Обработка команды получения списка процессов")

	processes, err := a.systemMonitor.GetTopProcesses(10)
	if err != nil {
		a.logger.WithError(err).Error("Ошибка получения списка процессов")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    "PROCESSES_ERROR",
			ErrorMessage: fmt.Sprintf("Ошибка получения списка процессов: %v", err),
		})
	}

	a.logger.WithField("processes_count", len(processes.Processes)).Info("Список процессов получен")
	response := protocol.NewMessage(protocol.TypeProcessesResponse, processes)
	response.ID = msg.ID
	return response
}
