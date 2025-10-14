package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/pkg/docker"
	"github.com/servereye/servereye/pkg/metrics"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/servereye/servereye/pkg/redis"
	"github.com/sirupsen/logrus"
)

// Agent представляет агент ServerEye
type Agent struct {
	config        *config.AgentConfig
	logger        *logrus.Logger
	redisClient   *redis.Client
	cpuMetrics    *metrics.CPUMetrics
	dockerClient  *docker.Client
	ctx           context.Context
	cancel        context.CancelFunc
}

// New создает новый агент
func New(cfg *config.AgentConfig, logger *logrus.Logger) (*Agent, error) {
	// Создаем Redis клиент
	redisClient, err := redis.NewClient(redis.Config{
		Address:  cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать Redis клиент: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Agent{
		config:        cfg,
		logger:        logger,
		redisClient:   redisClient,
		cpuMetrics:    metrics.NewCPUMetrics(),
		dockerClient:  docker.NewClient(logger),
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// Start запускает агент
func (a *Agent) Start() error {
	a.logger.WithFields(logrus.Fields{
		"server_name": a.config.Server.Name,
		"secret_key":  a.config.Server.SecretKey,
	}).Info("Запуск агента ServerEye")

	// Подписываемся на канал команд
	cmdChannel := redis.GetCommandChannel(a.config.Server.SecretKey)
	msgChan, err := a.redisClient.Subscribe(a.ctx, cmdChannel)
	if err != nil {
		return fmt.Errorf("не удалось подписаться на канал команд: %v", err)
	}

	a.logger.WithField("channel", cmdChannel).Info("Подписались на канал команд")

	// Запускаем обработку команд
	go a.handleCommands(msgChan)

	// Запускаем heartbeat
	go a.startHeartbeat()

	return nil
}

// Stop останавливает агент
func (a *Agent) Stop() error {
	a.logger.Info("Остановка агента")
	a.cancel()
	return a.redisClient.Close()
}

// handleCommands обрабатывает входящие команды
func (a *Agent) handleCommands(msgChan <-chan []byte) {
	for {
		select {
		case msg := <-msgChan:
			if msg == nil {
				return
			}
			a.processCommand(msg)
		case <-a.ctx.Done():
			return
		}
	}
}

// processCommand обрабатывает одну команду
func (a *Agent) processCommand(data []byte) {
	// Парсим сообщение
	msg, err := protocol.FromJSON(data)
	if err != nil {
		a.logger.WithError(err).Error("Не удалось парсить команду")
		return
	}

	a.logger.WithFields(logrus.Fields{
		"command_id":   msg.ID,
		"command_type": msg.Type,
	}).Info("Получена команда")

	var response *protocol.Message

	// Обрабатываем команду
	switch msg.Type {
	case protocol.TypeGetCPUTemp:
		response = a.handleGetCPUTemp(msg)
	case protocol.TypeGetContainers:
		response = a.handleGetContainers(msg)
	case protocol.TypePing:
		response = a.handlePing(msg)
	default:
		response = a.handleUnknownCommand(msg)
	}

	// Отправляем ответ
	if err := a.sendResponse(response); err != nil {
		a.logger.WithError(err).Error("Не удалось отправить ответ")
	}
}

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
	response.ID = msg.ID // Используем тот же ID для связи запроса и ответа
	return response
}

// handleGetContainers обрабатывает команду получения списка Docker контейнеров
func (a *Agent) handleGetContainers(msg *protocol.Message) *protocol.Message {
	containers, err := a.dockerClient.GetContainers(a.ctx)
	if err != nil {
		a.logger.WithError(err).Error("Не удалось получить список Docker контейнеров")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorCommandTimeout,
			ErrorMessage: err.Error(),
		})
	}

	response := protocol.NewMessage(protocol.TypeContainersResponse, containers)
	response.ID = msg.ID // Используем тот же ID для связи запроса и ответа
	return response
}

// handlePing обрабатывает ping команду
func (a *Agent) handlePing(msg *protocol.Message) *protocol.Message {
	payload := protocol.PongPayload{
		Status: "healthy",
		Uptime: "unknown", // TODO: реализовать подсчет uptime
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

// sendResponse отправляет ответ в канал ответов
func (a *Agent) sendResponse(msg *protocol.Message) error {
	data, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("не удалось сериализовать ответ: %v", err)
	}

	respChannel := redis.GetResponseChannel(a.config.Server.SecretKey)
	return a.redisClient.Publish(a.ctx, respChannel, data)
}

// startHeartbeat запускает отправку heartbeat сообщений
func (a *Agent) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.sendHeartbeat()
		case <-a.ctx.Done():
			return
		}
	}
}

// sendHeartbeat отправляет heartbeat сообщение
func (a *Agent) sendHeartbeat() {
	heartbeat := map[string]interface{}{
		"server_key": a.config.Server.SecretKey,
		"server_name": a.config.Server.Name,
		"timestamp": time.Now(),
		"status": "online",
	}

	data, err := json.Marshal(heartbeat)
	if err != nil {
		a.logger.WithError(err).Error("Не удалось сериализовать heartbeat")
		return
	}

	heartbeatChannel := fmt.Sprintf("heartbeat:%s", a.config.Server.SecretKey)
	if err := a.redisClient.Publish(a.ctx, heartbeatChannel, data); err != nil {
		a.logger.WithError(err).Error("Не удалось отправить heartbeat")
	}
}
