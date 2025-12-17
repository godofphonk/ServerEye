package agent

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/pkg/docker"
	"github.com/servereye/servereye/pkg/http"
	"github.com/servereye/servereye/pkg/kafka"
	"github.com/servereye/servereye/pkg/metrics"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/servereye/servereye/pkg/publisher"
	"github.com/sirupsen/logrus"
)

// Agent представляет агент ServerEye
type Agent struct {
	config          *config.AgentConfig
	logger          *logrus.Logger
	metricPublisher publisher.Publisher    // unified publisher
	commandConsumer *kafka.CommandConsumer // Kafka consumer for commands
	cpuMetrics      *metrics.CPUMetrics
	systemMonitor   *metrics.SystemMonitor
	dockerClient    *docker.Client
	ctx             context.Context
	cancel          context.CancelFunc
	useKafka        bool // Flag to use Kafka for commands

	// updateFunc allows mocking performUpdate in tests
	updateFunc func(string) error
	// updateDoneChan notifies when update goroutine completes (for tests)
	updateDoneChan chan<- bool
}

// initializeMetricPublisher создает publisher на основе конфигурации
func initializeMetricPublisher(cfg *config.AgentConfig, logger *logrus.Logger) (publisher.Publisher, error) {
	// Определяем режим publisher
	publisherMode := cfg.PublisherMode
	if publisherMode == "" {
		publisherMode = "hybrid" // По умолчанию hybrid для обратной совместимости
	}

	var publishers []publisher.Publisher

	// HTTP publisher (если настроен и режим позволяет)
	if cfg.API.BaseURL != "" && (publisherMode == "http" || publisherMode == "hybrid") {
		timeout := 30
		if cfg.API.Timeout != "" {
			if t, err := strconv.Atoi(cfg.API.Timeout); err == nil {
				timeout = t
			}
		}

		httpConfig := http.Config{
			BaseURL: cfg.API.BaseURL,
			APIKey:  cfg.API.APIKey,
			Timeout: timeout,
		}

		httpClient := http.New(httpConfig, logger)
		publishers = append(publishers, httpClient)
	}

	// Kafka publisher (если включен и режим позволяет)
	if cfg.Kafka.Enabled && len(cfg.Kafka.Brokers) > 0 && (publisherMode == "kafka" || publisherMode == "hybrid") {
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
			kafkaConfig.TopicPrefix = "servereye" // Обновлено для worldwide
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
			if publisherMode == "kafka" {
				return nil, fmt.Errorf("не удалось создать Kafka publisher: %w", err)
			}
			logger.WithError(err).Warn("Failed to create Kafka publisher, using HTTP only")
		} else {
			publishers = append(publishers, kafkaPub)
			logger.Info("Kafka publisher initialized")
		}
	}

	// Проверяем результат
	if len(publishers) == 0 {
		return nil, fmt.Errorf("no publishers configured for mode: %s", publisherMode)
	}

	// Если один publisher, возвращаем его напрямую
	if len(publishers) == 1 {
		return publishers[0], nil
	}

	// Иначе создаем multi publisher для hybrid режима
	logger.Info("Multi-publisher initialized for hybrid mode")
	return publisher.NewMultiPublisher(publishers, publisher.FailIfAll, logger), nil
}

// New создает новый агент
func New(cfg *config.AgentConfig, logger *logrus.Logger) (*Agent, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize metric publisher(s)
	metricPublisher, err := initializeMetricPublisher(cfg, logger)
	if err != nil {
		cancel() // Cleanup context
		return nil, fmt.Errorf("не удалось инициализировать metric publisher: %v", err)
	}

	// Initialize Kafka command consumer if enabled
	var commandConsumer *kafka.CommandConsumer
	var useKafka bool
	if cfg.Kafka.Enabled && len(cfg.Kafka.Brokers) > 0 {
		// Создаем временный agent для consumer initialization
		tempAgent := &Agent{
			config:        cfg,
			logger:        logger,
			cpuMetrics:    metrics.NewCPUMetrics(),
			systemMonitor: metrics.NewSystemMonitor(logger),
			dockerClient:  docker.NewClient(logger),
		}

		consumerConfig := kafka.CommandConsumerConfig{
			Brokers:        cfg.Kafka.Brokers,
			GroupID:        fmt.Sprintf("agent-new-%s", cfg.Server.SecretKey),
			ServerKey:      cfg.Server.SecretKey,
			Topic:          fmt.Sprintf("cmd.%s", cfg.Server.SecretKey),
			MinBytes:       10e3, // 10KB
			MaxBytes:       10e6, // 10MB
			CommitInterval: time.Second,
		}

		consumer, err := kafka.NewCommandConsumer(consumerConfig, tempAgent, logger)
		if err != nil {
			cancel() // Cleanup context
			return nil, fmt.Errorf("не удалось создать Kafka consumer: %v", err)
		}

		commandConsumer = consumer
		useKafka = true
		logger.Info("Kafka command consumer initialized")
	}

	return &Agent{
		config:          cfg,
		logger:          logger,
		metricPublisher: metricPublisher,
		commandConsumer: commandConsumer,
		useKafka:        useKafka,
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

	// Start Kafka consumer if enabled
	if a.useKafka && a.commandConsumer != nil {
		a.logger.Info("Starting with Kafka mode")
		if err := a.commandConsumer.Start(a.ctx); err != nil {
			return fmt.Errorf("не удалось запустить Kafka consumer: %v", err)
		}
	} else {
		a.logger.Warn("Command consumer disabled - Kafka not configured")
	}

	// Запускаем heartbeat
	go a.startHeartbeat()

	// Запускаем сборщик метрик если есть publisher
	if a.metricPublisher != nil {
		a.logger.Info("Starting metrics collection")
		go a.startMetricsCollection()
		go a.startTerminalHandler()
	}

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

	// Закрываем Kafka consumer
	if a.commandConsumer != nil {
		if err := a.commandConsumer.Close(); err != nil {
			a.logger.WithError(err).Error("Ошибка при закрытии Kafka consumer")
		}
	}

	return nil
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
	case protocol.TypeGetNetworkInfo:
		response = a.handleGetNetworkInfo(msg)
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
