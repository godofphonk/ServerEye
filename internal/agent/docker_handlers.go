package agent

import (
	"encoding/json"
	"fmt"

	"github.com/servereye/servereye/pkg/protocol"
)

// handleGetContainers обрабатывает команду получения списка Docker контейнеров
func (a *Agent) handleGetContainers(msg *protocol.Message) *protocol.Message {
	containers, err := a.dockerClient.GetContainers(a.ctx)
	if err != nil {
		a.logger.WithError(err).Error("Не удалось получить список Docker контейнеров")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorDockerUnavailable,
			ErrorMessage: fmt.Sprintf("Не удалось получить список контейнеров: %v", err),
		})
	}

	return protocol.NewMessage(protocol.TypeContainersResponse, containers)
}

// handleStartContainer обрабатывает команду запуска контейнера
func (a *Agent) handleStartContainer(msg *protocol.Message) *protocol.Message {
	a.logger.Info("Обработка команды start_container")

	var actionPayload protocol.ContainerActionPayload
	if err := parsePayload(msg.Payload, &actionPayload); err != nil {
		a.logger.WithError(err).Error("Не удалось распарсить payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	response, err := a.dockerClient.StartContainer(a.ctx, actionPayload.ContainerID)
	if err != nil {
		a.logger.WithError(err).Error("Ошибка при запуске контейнера")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorContainerAction,
			ErrorMessage: fmt.Sprintf("Ошибка при запуске контейнера: %v", err),
		})
	}

	response.ContainerName = actionPayload.ContainerName
	a.logger.WithField("container_id", actionPayload.ContainerID).Info("Контейнер успешно запущен")
	return protocol.NewMessage(protocol.TypeContainerActionResponse, response)
}

// handleStopContainer обрабатывает команду остановки контейнера
func (a *Agent) handleStopContainer(msg *protocol.Message) *protocol.Message {
	a.logger.Info("Обработка команды stop_container")

	var actionPayload protocol.ContainerActionPayload
	if err := parsePayload(msg.Payload, &actionPayload); err != nil {
		a.logger.WithError(err).Error("Не удалось распарсить payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	response, err := a.dockerClient.StopContainer(a.ctx, actionPayload.ContainerID)
	if err != nil {
		a.logger.WithError(err).Error("Ошибка при остановке контейнера")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorContainerAction,
			ErrorMessage: fmt.Sprintf("Ошибка при остановке контейнера: %v", err),
		})
	}

	response.ContainerName = actionPayload.ContainerName
	a.logger.WithField("container_id", actionPayload.ContainerID).Info("Контейнер успешно остановлен")
	return protocol.NewMessage(protocol.TypeContainerActionResponse, response)
}

// handleRestartContainer обрабатывает команду перезапуска контейнера
func (a *Agent) handleRestartContainer(msg *protocol.Message) *protocol.Message {
	a.logger.Info("Обработка команды restart_container")

	var actionPayload protocol.ContainerActionPayload
	if err := parsePayload(msg.Payload, &actionPayload); err != nil {
		a.logger.WithError(err).Error("Не удалось распарсить payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	response, err := a.dockerClient.RestartContainer(a.ctx, actionPayload.ContainerID)
	if err != nil {
		a.logger.WithError(err).Error("Ошибка при перезапуске контейнера")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorContainerAction,
			ErrorMessage: fmt.Sprintf("Ошибка при перезапуске контейнера: %v", err),
		})
	}

	response.ContainerName = actionPayload.ContainerName
	a.logger.WithField("container_id", actionPayload.ContainerID).Info("Контейнер успешно перезапущен")
	return protocol.NewMessage(protocol.TypeContainerActionResponse, response)
}

// handleRemoveContainer обрабатывает команду удаления контейнера
func (a *Agent) handleRemoveContainer(msg *protocol.Message) *protocol.Message {
	a.logger.Info("Обработка команды remove_container")

	var actionPayload protocol.ContainerActionPayload
	if err := parsePayload(msg.Payload, &actionPayload); err != nil {
		a.logger.WithError(err).Error("Не удалось распарсить payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	response, err := a.dockerClient.RemoveContainer(a.ctx, actionPayload.ContainerID)
	if err != nil {
		a.logger.WithError(err).Error("Ошибка при удалении контейнера")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorContainerAction,
			ErrorMessage: fmt.Sprintf("Ошибка при удалении контейнера: %v", err),
		})
	}

	response.ContainerName = actionPayload.ContainerName
	a.logger.WithField("container_id", actionPayload.ContainerID).Info("Контейнер успешно удален")
	return protocol.NewMessage(protocol.TypeContainerActionResponse, response)
}

// handleCreateContainer обрабатывает команду создания контейнера
func (a *Agent) handleCreateContainer(msg *protocol.Message) *protocol.Message {
	a.logger.Info("Обработка команды create_container")

	var createPayload protocol.CreateContainerPayload
	if err := parsePayload(msg.Payload, &createPayload); err != nil {
		a.logger.WithError(err).Error("Не удалось распарсить payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	response, err := a.dockerClient.CreateContainer(a.ctx, &createPayload)
	if err != nil {
		a.logger.WithError(err).Error("Ошибка при создании контейнера")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorContainerAction,
			ErrorMessage: fmt.Sprintf("Ошибка при создании контейнера: %v", err),
		})
	}

	a.logger.WithField("container_name", createPayload.Name).Info("Контейнер успешно создан")
	return protocol.NewMessage(protocol.TypeContainerActionResponse, response)
}

// parsePayload helper для парсинга payload
func parsePayload(payload interface{}, target interface{}) error {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("не удалось сериализовать payload: %w", err)
	}

	if err := json.Unmarshal(payloadData, target); err != nil {
		return fmt.Errorf("не удалось распарсить payload: %w", err)
	}

	return nil
}
