package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/pkg/docker"
	"github.com/servereye/servereye/pkg/kafka"
	"github.com/servereye/servereye/pkg/metrics"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/servereye/servereye/pkg/publisher"
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
	config          *config.AgentConfig
	logger          *logrus.Logger
	redisClient     RedisClientInterface
	streamsClient   streams.StreamClient // NEW: for Streams support
	metricPublisher publisher.Publisher  // NEW: unified publisher (может быть multi-publisher)
	cpuMetrics      *metrics.CPUMetrics
	systemMonitor   *metrics.SystemMonitor
	dockerClient    *docker.Client
	ctx             context.Context
	cancel          context.CancelFunc
	useStreams      bool // Flag to use Streams instead of Pub/Sub

	// updateFunc allows mocking performUpdate in tests
	updateFunc func(string) error
	// updateDoneChan notifies when update goroutine completes (for tests)
	updateDoneChan chan<- bool
}

// initializeMetricPublisher создает publisher на основе конфигурации
func initializeMetricPublisher(cfg *config.AgentConfig, logger *logrus.Logger) (publisher.Publisher, error) {
	var publishers []publisher.Publisher

	// Kafka publisher (если включен)
	if cfg.Kafka.Enabled && len(cfg.Kafka.Brokers) > 0 {
		kafkaConfig := kafka.Config{
			Brokers:      cfg.Kafka.Brokers,
			TopicPrefix:  cfg.Kafka.TopicPrefix,
			Compression:  cfg.Kafka.Compression,
			MaxAttempts:  cfg.Kafka.MaxAttempts,
			BatchSize:    cfg.Kafka.BatchSize,
			RequiredAcks: cfg.Kafka.RequiredAcks,
		}

		// Установка дефолтных значений если не указаны
		if kafkaConfig.TopicPrefix == "" {
			kafkaConfig.TopicPrefix = "metrics"
		}
		if kafkaConfig.Compression == "" {
			kafkaConfig.Compression = "snappy"
		}
		if kafkaConfig.MaxAttempts == 0 {
			kafkaConfig.MaxAttempts = 3
		}
		if kafkaConfig.BatchSize == 0 {
			kafkaConfig.BatchSize = 100
		}
		if kafkaConfig.RequiredAcks == 0 {
			kafkaConfig.RequiredAcks = 1
		}

		kafkaPub, err := kafka.NewProducer(kafkaConfig, logger)
		if err != nil {
			return nil, fmt.Errorf("не удалось создать Kafka publisher: %w", err)
		}

		publishers = append(publishers, kafkaPub)
		logger.Info("Kafka publisher инициализирован")
	}

	// Если нет publishers, возвращаем nil (агент работает только через Redis Streams)
	if len(publishers) == 0 {
		logger.Info("Metric publishers не настроены, используется только Redis Streams")
		return nil, nil
	}

	// Если один publisher, возвращаем его напрямую
	if len(publishers) == 1 {
		return publishers[0], nil
	}

	// Если несколько publishers, создаем multi-publisher
	// Используем FailIfPrimary - ошибка только если Kafka (первый) упадет
	multiPub := publisher.NewMultiPublisher(publishers, publisher.FailIfPrimary, logger)
	logger.WithField("count", len(publishers)).Info("Multi-publisher инициализирован")

	return multiPub, nil
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

	// Initialize metric publisher(s)
	metricPublisher, err := initializeMetricPublisher(cfg, logger)
	if err != nil {
		cancel() // Cleanup context
		return nil, fmt.Errorf("не удалось инициализировать metric publisher: %v", err)
	}

	return &Agent{
		config:          cfg,
		logger:          logger,
		redisClient:     redisClient,
		streamsClient:   streamsClient,
		metricPublisher: metricPublisher,
		useStreams:      useStreams,
		cpuMetrics:      metrics.NewCPUMetrics(),
		systemMonitor:   metrics.NewSystemMonitor(logger),
		dockerClient:    docker.NewClient(logger),
		ctx:             ctx,
		cancel:          cancel,
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

	// Закрываем metric publisher если есть
	if a.metricPublisher != nil {
		if err := a.metricPublisher.Close(); err != nil {
			a.logger.WithError(err).Error("Ошибка при закрытии metric publisher")
		}
	}

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
			if err := a.sendResponse(response); err != nil {
				a.logger.WithError(err).Error("Failed to send response")
			}
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
	case protocol.TypeUpdateAgent:
		response = a.handleUpdateAgent(msg)
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

		// Отправляем в уникальный канал с ID команды (Redis Streams)
		if err := a.sendResponseToCommand(response, msg.ID); err != nil {
			a.logger.WithError(err).Error("Не удалось отправить ответ")
		} else {
			a.logger.WithField("command_id", msg.ID).Info("Ответ успешно отправлен")
		}

		// Дополнительно отправляем метрику в Kafka (если настроен)
		a.publishMetricToKafka(response)
	} else {
		a.logger.WithField("command_id", msg.ID).Error("Ответ не сгенерирован")
	}
}

// Command handlers are in separate files:
// - docker_handlers.go: Docker container management
// - monitoring_handlers.go: System monitoring (CPU, memory, disk, etc.)
// - update.go: Agent update functionality
// - heartbeat.go: Heartbeat functionality
// - helpers.go: Utility functions (ping, sendResponse, etc.)

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
