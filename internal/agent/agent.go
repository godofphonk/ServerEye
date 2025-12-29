package agent

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/godofphonk/ServerEye/internal/config"
	"github.com/godofphonk/ServerEye/pkg/commands"
	"github.com/godofphonk/ServerEye/pkg/docker"
	"github.com/godofphonk/ServerEye/pkg/http"
	"github.com/godofphonk/ServerEye/pkg/metrics"
	"github.com/godofphonk/ServerEye/pkg/protocol"
	"github.com/godofphonk/ServerEye/pkg/publisher"
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
		result := a.handlePing(msg)
		return result, nil
	case protocol.TypeGetCPUTemp:
		result := a.handleGetCPUTemp(msg)
		return result, nil
	case protocol.TypeGetContainers:
		result := a.handleGetContainers(msg)
		return result, nil
	case protocol.TypeStartContainer:
		result := a.handleStartContainer(msg)
		return result, nil
	case protocol.TypeStopContainer:
		result := a.handleStopContainer(msg)
		return result, nil
	case protocol.TypeRestartContainer:
		result := a.handleRestartContainer(msg)
		return result, nil
	case protocol.TypeRemoveContainer:
		result := a.handleRemoveContainer(msg)
		return result, nil
	case protocol.TypeCreateContainer:
		response.Type = protocol.TypeErrorResponse
		response.Payload = protocol.ErrorPayload{
			ErrorCode:    protocol.ErrorInvalidCommand,
			ErrorMessage: "Create container command not implemented",
		}
	case protocol.TypeGetMemoryInfo:
		result := a.handleGetMemoryInfo(msg)
		return result, nil
	case protocol.TypeGetDiskInfo:
		result := a.handleGetDiskInfo(msg)
		return result, nil
	case protocol.TypeGetUptime:
		result := a.handleGetUptime(msg)
		return result, nil
	case protocol.TypeGetProcesses:
		result := a.handleGetProcesses(msg)
		return result, nil
	case protocol.TypeGetNetworkInfo:
		result := a.handleGetNetworkInfo(msg)
		return result, nil
	case protocol.TypeUpdateAgent:
		result := a.handleUpdateAgent(msg)
		return result, nil
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
