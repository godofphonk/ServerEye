package agent

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/pkg/commands"
	"github.com/servereye/servereye/pkg/docker"
	"github.com/servereye/servereye/pkg/http"
	"github.com/servereye/servereye/pkg/metrics"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/servereye/servereye/pkg/publisher"
	"github.com/sirupsen/logrus"
)

// Agent представляет агент ServerEye
type Agent struct {
	config              *config.AgentConfig
	logger              *logrus.Logger
	metricPublisher     publisher.Publisher           // unified publisher
	httpCommandConsumer *commands.HTTPCommandConsumer // HTTP consumer for commands
	cpuMetrics          *metrics.CPUMetrics
	systemMonitor       *metrics.SystemMonitor
	dockerClient        *docker.Client
	ctx                 context.Context
	cancel              context.CancelFunc
}

// initializeMetricPublisher создает HTTP publisher для метрик
func initializeMetricPublisher(cfg *config.AgentConfig, logger *logrus.Logger) (publisher.Publisher, error) {
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

	return http.New(httpConfig, logger), nil
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

	// Initialize HTTP command consumer (worldwide mode)
	// Create temp agent for command handling
	tempAgent := &Agent{
		config:        cfg,
		logger:        logger,
		cpuMetrics:    metrics.NewCPUMetrics(),
		systemMonitor: metrics.NewSystemMonitor(logger),
		dockerClient:  docker.NewClient(logger),
	}

	httpConfig := commands.HTTPConsumerConfig{
		APIURL:       cfg.API.BaseURL,
		APIKey:       cfg.API.APIKey,
		ServerKey:    cfg.Server.SecretKey,
		PollInterval: 2 * time.Second,
	}

	httpCommandConsumer := commands.NewHTTPCommandConsumer(httpConfig, tempAgent, logger)
	logger.Info("HTTP command consumer initialized (worldwide mode)")

	return &Agent{
		config:              cfg,
		logger:              logger,
		metricPublisher:     metricPublisher,
		httpCommandConsumer: httpCommandConsumer,
		cpuMetrics:          metrics.NewCPUMetrics(),
		systemMonitor:       metrics.NewSystemMonitor(logger),
		dockerClient:        docker.NewClient(logger),
		ctx:                 ctx,
		cancel:              cancel,
	}, nil
}

// Start запускает агент
func (a *Agent) Start() error {
	a.logger.WithFields(logrus.Fields{
		"server_name": a.config.Server.Name,
		"secret_key":  a.config.Server.SecretKey,
	}).Info("Запуск агента ServerEye")

	// Start command consumer (HTTP mode)
	if a.httpCommandConsumer != nil {
		a.logger.Info("Starting with HTTP command mode (worldwide)")
		go func() {
			if err := a.httpCommandConsumer.Start(a.ctx); err != nil {
				a.logger.WithError(err).Error("HTTP command consumer failed")
			}
		}()
	} else {
		a.logger.Warn("No HTTP command consumer available - commands disabled")
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

	// Закрываем HTTP command consumer
	if a.httpCommandConsumer != nil {
		if err := a.httpCommandConsumer.Close(); err != nil {
			a.logger.WithError(err).Error("Ошибка при закрытии HTTP command consumer")
		}
	}

	return nil
}

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

	// Обрабатываем команду
	switch msg.Type {
	case protocol.TypePing:
		response.Type = protocol.TypePong
		response.Payload = protocol.PongPayload{
			Status: "healthy",
			Uptime: "unknown",
		}
	case protocol.TypeGetCPUTemp:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorCommandTimeout,
			ErrorMessage: "CPU temperature command not implemented",
		}
	case protocol.TypeGetContainers:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorDockerUnavailable,
			ErrorMessage: "Docker not available",
		}
	case protocol.TypeStartContainer:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Start container command not implemented",
		}
	case protocol.TypeStopContainer:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Stop container command not implemented",
		}
	case protocol.TypeRestartContainer:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Restart container command not implemented",
		}
	case protocol.TypeRemoveContainer:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Remove container command not implemented",
		}
	case protocol.TypeCreateContainer:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Create container command not implemented",
		}
	case protocol.TypeGetMemoryInfo:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorCommandTimeout,
			ErrorMessage: "Memory info command not implemented",
		}
	case protocol.TypeGetDiskInfo:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorCommandTimeout,
			ErrorMessage: "Disk info command not implemented",
		}
	case protocol.TypeGetUptime:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorCommandTimeout,
			ErrorMessage: "Uptime command not implemented",
		}
	case protocol.TypeGetProcesses:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Get processes command not implemented",
		}
	case protocol.TypeGetNetworkInfo:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Get network info command not implemented",
		}
	case protocol.TypeUpdateAgent:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Update agent command not implemented",
		}
	default:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: fmt.Sprintf("Неизвестная команда: %s", msg.Type),
		}
	}

	return response, nil
}

// Command handlers are in separate files:
// - handlers.go: All command handlers
