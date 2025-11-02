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
	"github.com/servereye/servereye/pkg/redis/streams"
	"github.com/sirupsen/logrus"
)

// RedisClientInterface общий интерфейс для Redis клиентов
type RedisClientInterface interface {
	Subscribe(ctx context.Context, channel string) (SubscriptionInterface, error)
	Publish(ctx context.Context, channel string, message []byte) error
	Close() error
}

// SubscriptionInterface общий интерфейс для подписок
type SubscriptionInterface interface {
	Channel() <-chan []byte
	Close() error
}

// Agent представляет агент ServerEye
type Agent struct {
	config        *config.AgentConfig
	logger        *logrus.Logger
	redisClient   RedisClientInterface
	streamsClient streams.StreamClient // NEW: for Streams support
	cpuMetrics    *metrics.CPUMetrics
	systemMonitor *metrics.SystemMonitor
	dockerClient  *docker.Client
	ctx           context.Context
	cancel        context.CancelFunc
	useStreams    bool // Flag to use Streams instead of Pub/Sub
}

// New создает новый агент
func New(cfg *config.AgentConfig, logger *logrus.Logger) (*Agent, error) {
	var redisClient RedisClientInterface

	// Выбираем тип клиента на основе конфигурации
	if cfg.API.BaseURL != "" {
		// Используем HTTP клиент
		timeout := 30 * time.Second
		if cfg.API.Timeout != "" {
			if parsedTimeout, err := time.ParseDuration(cfg.API.Timeout); err == nil {
				timeout = parsedTimeout
			}
		}

		httpClient, err := redis.NewHTTPClient(redis.HTTPConfig{
			BaseURL: cfg.API.BaseURL,
			Timeout: timeout,
		}, logger)
		if err != nil {
			return nil, fmt.Errorf("не удалось создать HTTP клиент: %v", err)
		}
		redisClient = &HTTPClientAdapter{client: httpClient}
		logger.Info("Используется HTTP клиент для связи с сервером")
	} else {
		// Используем прямой Redis клиент
		directClient, err := redis.NewClient(redis.Config{
			Address:  cfg.Redis.Address,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		}, logger)
		if err != nil {
			return nil, fmt.Errorf("не удалось создать Redis клиент: %v", err)
		}
		redisClient = &DirectClientAdapter{client: directClient}
		logger.Info("Используется прямой Redis клиент")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize Streams client if using HTTP API
	var streamsClient streams.StreamClient
	var useStreams bool
	if cfg.API.BaseURL != "" {
		streamsClient = streams.NewHTTPStreamClient(cfg.API.BaseURL, logger)
		useStreams = true
		logger.Info("Streams support enabled via HTTP API")
	}

	return &Agent{
		config:        cfg,
		logger:        logger,
		redisClient:   redisClient,
		streamsClient: streamsClient,
		useStreams:    useStreams,
		cpuMetrics:    metrics.NewCPUMetrics(),
		systemMonitor: metrics.NewSystemMonitor(logger),
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

	// Use Streams if available
	if a.useStreams && a.streamsClient != nil {
		a.logger.Info("Starting with Streams mode")
		go a.handleCommandsViaStreams()
	} else {
		// Fallback to Pub/Sub
		a.logger.Info("Starting with Pub/Sub mode")
		cmdChannel := redis.GetCommandChannel(a.config.Server.SecretKey)
		msgChan, err := a.redisClient.Subscribe(a.ctx, cmdChannel)
		if err != nil {
			return fmt.Errorf("не удалось подписаться на канал команд: %v", err)
		}
		a.logger.WithField("channel", cmdChannel).Info("Подписались на канал команд")
		go a.handleCommands(msgChan.Channel())
	}

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

	// Обрабатываем команду с обработкой паники
	defer func() {
		if r := recover(); r != nil {
			a.logger.WithFields(logrus.Fields{
				"command_id":   msg.ID,
				"command_type": msg.Type,
				"panic":        r,
			}).Error("Паника при обработке команды")

			response = protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
				ErrorCode:    "PANIC_ERROR",
				ErrorMessage: fmt.Sprintf("Внутренняя ошибка при обработке команды: %v", r),
			})
			response.ID = msg.ID
			a.sendResponse(response)
		}
	}()

	// Обрабатываем команду
	switch msg.Type {
	case protocol.TypeGetCPUTemp:
		response = a.handleGetCPUTemp(msg)
	case protocol.TypeGetContainers:
		response = a.handleGetContainers(msg)
	case protocol.TypeStartContainer:
		response = a.handleStartContainer(msg)
	case protocol.TypeStopContainer:
		response = a.handleStopContainer(msg)
	case protocol.TypeRestartContainer:
		response = a.handleRestartContainer(msg)
	case protocol.TypeRemoveContainer:
		response = a.handleRemoveContainer(msg)
	case protocol.TypeCreateContainer:
		response = a.handleCreateContainer(msg)
	case protocol.TypeGetMemoryInfo:
		response = a.handleGetMemoryInfo(msg)
	case protocol.TypeGetDiskInfo:
		response = a.handleGetDiskInfo(msg)
	case protocol.TypeGetUptime:
		response = a.handleGetUptime(msg)
	case protocol.TypeGetProcesses:
		response = a.handleGetProcesses(msg)
	case protocol.TypePing:
		response = a.handlePing(msg)
	default:
		response = a.handleUnknownCommand(msg)
	}

	// Отправляем ответ
	if response != nil {
		a.logger.WithFields(logrus.Fields{
			"command_id":    msg.ID,
			"response_type": response.Type,
		}).Info("Отправляем ответ")

		// Отправляем в уникальный канал с ID команды
		if err := a.sendResponseToCommand(response, msg.ID); err != nil {
			a.logger.WithError(err).Error("Не удалось отправить ответ")
		} else {
			a.logger.WithField("command_id", msg.ID).Info("Ответ успешно отправлен")
		}
	} else {
		a.logger.WithField("command_id", msg.ID).Error("Ответ не сгенерирован")
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

// sendResponse отправляет ответ в канал ответов (устаревшая версия)
func (a *Agent) sendResponse(msg *protocol.Message) error {
	data, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("не удалось сериализовать ответ: %v", err)
	}

	respChannel := redis.GetResponseChannel(a.config.Server.SecretKey)
	return a.redisClient.Publish(a.ctx, respChannel, data)
}

// sendResponseToCommand отправляет ответ в Stream или Pub/Sub
func (a *Agent) sendResponseToCommand(msg *protocol.Message, commandID string) error {
	data, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("не удалось сериализовать ответ: %v", err)
	}

	// Use Streams if available
	if a.useStreams && a.streamsClient != nil {
		respStream := fmt.Sprintf("stream:resp:%s", a.config.Server.SecretKey)
		
		values := map[string]string{
			"type":       string(msg.Type),
			"id":         msg.ID,
			"command_id": commandID,
			"payload":    string(data),
			"timestamp":  time.Now().Format(time.RFC3339),
		}
		
		_, err := a.streamsClient.AddMessage(a.ctx, respStream, values)
		if err != nil {
			a.logger.WithError(err).Error("Failed to send via Streams")
			return err
		}
		
		a.logger.WithField("stream", respStream).Debug("Response sent via Streams")
		return nil
	}

	// Fallback to Pub/Sub
	respChannel := fmt.Sprintf("resp:%s:%s", a.config.Server.SecretKey, commandID)
	a.logger.WithField("response_channel", respChannel).Debug("Отправка ответа в уникальный канал")
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
		"server_key":  a.config.Server.SecretKey,
		"server_name": a.config.Server.Name,
		"timestamp":   time.Now(),
		"status":      "online",
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

// handleStartContainer обрабатывает команду запуска контейнера
func (a *Agent) handleStartContainer(msg *protocol.Message) *protocol.Message {
	a.logger.Info("Обработка команды start_container")

	// Парсим payload
	payloadData, err := json.Marshal(msg.Payload)
	if err != nil {
		a.logger.WithError(err).Error("Не удалось сериализовать payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	var actionPayload protocol.ContainerActionPayload
	if err := json.Unmarshal(payloadData, &actionPayload); err != nil {
		a.logger.WithError(err).Error("Не удалось распарсить payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	// Выполняем команду
	response, err := a.dockerClient.StartContainer(a.ctx, actionPayload.ContainerID)
	if err != nil {
		a.logger.WithError(err).Error("Ошибка при запуске контейнера")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorContainerAction,
			ErrorMessage: fmt.Sprintf("Ошибка при запуске контейнера: %v", err),
		})
	}

	// Добавляем имя контейнера в ответ
	response.ContainerName = actionPayload.ContainerName

	a.logger.WithField("container_id", actionPayload.ContainerID).Info("Контейнер успешно запущен")
	return protocol.NewMessage(protocol.TypeContainerActionResponse, response)
}

// handleStopContainer обрабатывает команду остановки контейнера
func (a *Agent) handleStopContainer(msg *protocol.Message) *protocol.Message {
	a.logger.Info("Обработка команды stop_container")

	// Парсим payload
	payloadData, err := json.Marshal(msg.Payload)
	if err != nil {
		a.logger.WithError(err).Error("Не удалось сериализовать payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	var actionPayload protocol.ContainerActionPayload
	if err := json.Unmarshal(payloadData, &actionPayload); err != nil {
		a.logger.WithError(err).Error("Не удалось распарсить payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	// Выполняем команду
	response, err := a.dockerClient.StopContainer(a.ctx, actionPayload.ContainerID)
	if err != nil {
		a.logger.WithError(err).Error("Ошибка при остановке контейнера")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorContainerAction,
			ErrorMessage: fmt.Sprintf("Ошибка при остановке контейнера: %v", err),
		})
	}

	// Добавляем имя контейнера в ответ
	response.ContainerName = actionPayload.ContainerName

	a.logger.WithField("container_id", actionPayload.ContainerID).Info("Контейнер успешно остановлен")
	return protocol.NewMessage(protocol.TypeContainerActionResponse, response)
}

// handleRestartContainer обрабатывает команду перезапуска контейнера
func (a *Agent) handleRestartContainer(msg *protocol.Message) *protocol.Message {
	a.logger.Info("Обработка команды restart_container")

	// Парсим payload
	payloadData, err := json.Marshal(msg.Payload)
	if err != nil {
		a.logger.WithError(err).Error("Не удалось сериализовать payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	var actionPayload protocol.ContainerActionPayload
	if err := json.Unmarshal(payloadData, &actionPayload); err != nil {
		a.logger.WithError(err).Error("Не удалось распарсить payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	// Выполняем команду
	response, err := a.dockerClient.RestartContainer(a.ctx, actionPayload.ContainerID)
	if err != nil {
		a.logger.WithError(err).Error("Ошибка при перезапуске контейнера")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorContainerAction,
			ErrorMessage: fmt.Sprintf("Ошибка при перезапуске контейнера: %v", err),
		})
	}

	// Добавляем имя контейнера в ответ
	response.ContainerName = actionPayload.ContainerName

	a.logger.WithField("container_id", actionPayload.ContainerID).Info("Контейнер успешно перезапущен")
	return protocol.NewMessage(protocol.TypeContainerActionResponse, response)
}

// handleRemoveContainer обрабатывает команду удаления контейнера
func (a *Agent) handleRemoveContainer(msg *protocol.Message) *protocol.Message {
	a.logger.Info("Обработка команды remove_container")

	// Парсим payload
	payloadData, err := json.Marshal(msg.Payload)
	if err != nil {
		a.logger.WithError(err).Error("Не удалось сериализовать payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	var actionPayload protocol.ContainerActionPayload
	if err := json.Unmarshal(payloadData, &actionPayload); err != nil {
		a.logger.WithError(err).Error("Не удалось распарсить payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	// Выполняем команду
	response, err := a.dockerClient.RemoveContainer(a.ctx, actionPayload.ContainerID)
	if err != nil {
		a.logger.WithError(err).Error("Ошибка при удалении контейнера")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorContainerAction,
			ErrorMessage: fmt.Sprintf("Ошибка при удалении контейнера: %v", err),
		})
	}

	// Добавляем имя контейнера в ответ
	response.ContainerName = actionPayload.ContainerName

	a.logger.WithField("container_id", actionPayload.ContainerID).Info("Контейнер успешно удален")
	return protocol.NewMessage(protocol.TypeContainerActionResponse, response)
}

// handleCreateContainer обрабатывает команду создания контейнера
func (a *Agent) handleCreateContainer(msg *protocol.Message) *protocol.Message {
	a.logger.Info("Обработка команды create_container")

	// Парсим payload
	payloadData, err := json.Marshal(msg.Payload)
	if err != nil {
		a.logger.WithError(err).Error("Не удалось сериализовать payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	var createPayload protocol.CreateContainerPayload
	if err := json.Unmarshal(payloadData, &createPayload); err != nil {
		a.logger.WithError(err).Error("Не удалось распарсить payload")
		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Неверный формат команды",
		})
	}

	// Выполняем команду
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

	// По умолчанию показываем топ 10 процессов
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

// HTTPClientAdapter адаптер для HTTP клиента
type HTTPClientAdapter struct {
	client *redis.HTTPClient
}

func (h *HTTPClientAdapter) Subscribe(ctx context.Context, channel string) (SubscriptionInterface, error) {
	sub, err := h.client.Subscribe(ctx, channel)
	if err != nil {
		return nil, err
	}
	return &HTTPSubscriptionAdapter{sub: sub}, nil
}

func (h *HTTPClientAdapter) Publish(ctx context.Context, channel string, message []byte) error {
	return h.client.Publish(ctx, channel, message)
}

func (h *HTTPClientAdapter) Close() error {
	return h.client.Close()
}

// HTTPSubscriptionAdapter адаптер для HTTP подписки
type HTTPSubscriptionAdapter struct {
	sub *redis.HTTPSubscription
}

func (h *HTTPSubscriptionAdapter) Channel() <-chan []byte {
	return h.sub.Channel()
}

func (h *HTTPSubscriptionAdapter) Close() error {
	return h.sub.Close()
}

// DirectClientAdapter адаптер для прямого Redis клиента
type DirectClientAdapter struct {
	client *redis.Client
}

func (d *DirectClientAdapter) Subscribe(ctx context.Context, channel string) (SubscriptionInterface, error) {
	sub, err := d.client.Subscribe(ctx, channel)
	if err != nil {
		return nil, err
	}
	return &DirectSubscriptionAdapter{sub: sub}, nil
}

func (d *DirectClientAdapter) Publish(ctx context.Context, channel string, message []byte) error {
	return d.client.Publish(ctx, channel, message)
}

func (d *DirectClientAdapter) Close() error {
	return d.client.Close()
}

// handleCommandsViaStreams reads commands from Streams
func (a *Agent) handleCommandsViaStreams() {
	a.logger.Info("Streams command handler started")
	cmdStream := fmt.Sprintf("stream:cmd:%s", a.config.Server.SecretKey)
	
	lastID := "0" // Start from beginning, then use "$" for new messages
	firstRead := true
	
	for {
		select {
		case <-a.ctx.Done():
			a.logger.Info("Streams handler stopped")
			return
		default:
			// Read from stream
			id := lastID
			if !firstRead {
				id = "$" // Only new messages after first read
			}
			
			messages, err := a.streamsClient.ReadMessages(a.ctx, cmdStream, id, 10, 5*time.Second)
			if err != nil {
				if err.Error() != "XREAD failed: context deadline exceeded" {
					a.logger.WithError(err).Error("Failed to read from stream")
				}
				time.Sleep(1 * time.Second)
				continue
			}
			
			firstRead = false
			
			// Process messages
			for _, msg := range messages {
				lastID = msg.ID
				
				// Parse command
				payloadJSON := msg.Values["payload"]
				command, err := protocol.FromJSON([]byte(payloadJSON))
				if err != nil {
					a.logger.WithError(err).Error("Failed to parse command")
					continue
				}
				
				a.logger.WithFields(logrus.Fields{
					"command_id":   command.ID,
					"command_type": command.Type,
				}).Info("Получена команда via Streams")
				
				// Process command (processCommand expects []byte)
				cmdData, _ := command.ToJSON()
				a.processCommand(cmdData)
			}
		}
	}
}

// DirectSubscriptionAdapter адаптер для прямой Redis подписки
type DirectSubscriptionAdapter struct {
	sub *redis.Subscription
}

func (d *DirectSubscriptionAdapter) Channel() <-chan []byte {
	return d.sub.Channel()
}

func (d *DirectSubscriptionAdapter) Close() error {
	return d.sub.Close()
}
