package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/godofphonk/ServerEye/internal/config"
	"github.com/godofphonk/ServerEye/pkg/commands"
	"github.com/godofphonk/ServerEye/pkg/docker"
	"github.com/godofphonk/ServerEye/pkg/metrics"
	"github.com/godofphonk/ServerEye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// Agent представляет агент ServerEye
type Agent struct {
	config            *config.AgentConfig
	logger            *logrus.Logger
	wsPublisher       *metrics.WebSocketPublisher        // WebSocket publisher
	wsCommandConsumer *commands.WebSocketCommandConsumer // WebSocket consumer
	useWebSocket      bool                               // Use WebSocket instead of HTTP
	cpuMetrics        *metrics.CPUMetrics
	systemMonitor     *metrics.SystemMonitor
	dockerClient      *docker.Client
	ctx               context.Context
	cancel            context.CancelFunc
}

// initializeWebSocketPublisher создает WebSocket publisher для метрик
func initializeWebSocketPublisher(cfg *config.AgentConfig, logger *logrus.Logger) (*metrics.WebSocketPublisher, error) {
	// Parse WebSocket URL
	wsURL := cfg.WebSocket.URL
	if wsURL == "" {
		// Fallback to API URL with WebSocket protocol
		wsURL = "ws" + cfg.API.BaseURL[4:] + "/ws"
	}

	// Parse durations with defaults
	reconnectInterval := parseDuration(cfg.WebSocket.ReconnectInterval, 5*time.Second)
	maxReconnectAttempts := cfg.WebSocket.MaxReconnectAttempts
	if maxReconnectAttempts == 0 {
		maxReconnectAttempts = 10
	}
	pingInterval := parseDuration(cfg.WebSocket.PingInterval, 30*time.Second)
	writeTimeout := parseDuration(cfg.WebSocket.WriteTimeout, 10*time.Second)
	readTimeout := parseDuration(cfg.WebSocket.ReadTimeout, 10*time.Second)
	handshakeTimeout := parseDuration(cfg.WebSocket.HandshakeTimeout, 10*time.Second)
	bufferSize := cfg.WebSocket.BufferSize
	if bufferSize == 0 {
		bufferSize = 1000
	}
	metricBufferSize := cfg.WebSocket.MetricBufferSize
	if metricBufferSize == 0 {
		metricBufferSize = 100
	}
	metricBufferFlush := parseDuration(cfg.WebSocket.MetricBufferFlush, 30*time.Second)

	wsConfig := metrics.Config{
		URL:                  wsURL,
		ServerID:             cfg.Server.ServerID,
		ServerKey:            cfg.Server.SecretKey,
		ReconnectInterval:    reconnectInterval,
		MaxReconnectAttempts: maxReconnectAttempts,
		PingInterval:         pingInterval,
		WriteTimeout:         writeTimeout,
		ReadTimeout:          readTimeout,
		HandshakeTimeout:     handshakeTimeout,
		BufferSize:           bufferSize,
		EnableCompression:    cfg.WebSocket.EnableCompression,
		MetricBufferSize:     metricBufferSize,
		MetricBufferFlush:    metricBufferFlush,
		APIURL:               cfg.API.BaseURL,
		APIKey:               cfg.API.APIKey,
	}

	return metrics.NewWebSocketPublisher(wsConfig, logger), nil
}

// initializeWebSocketCommandConsumer создает WebSocket command consumer
func initializeWebSocketCommandConsumer(cfg *config.AgentConfig, handler commands.CommandHandlerInterface, logger *logrus.Logger) (*commands.WebSocketCommandConsumer, error) {
	// Parse WebSocket URL
	wsURL := cfg.WebSocket.URL
	if wsURL == "" {
		// Fallback to API URL with WebSocket protocol
		wsURL = "ws" + cfg.API.BaseURL[4:] + "/ws"
	}

	// Parse durations with defaults
	reconnectInterval := parseDuration(cfg.WebSocket.ReconnectInterval, 5*time.Second)
	maxReconnectAttempts := cfg.WebSocket.MaxReconnectAttempts
	if maxReconnectAttempts == 0 {
		maxReconnectAttempts = 10
	}
	pingInterval := parseDuration(cfg.WebSocket.PingInterval, 30*time.Second)
	writeTimeout := parseDuration(cfg.WebSocket.WriteTimeout, 10*time.Second)
	readTimeout := parseDuration(cfg.WebSocket.ReadTimeout, 10*time.Second)
	handshakeTimeout := parseDuration(cfg.WebSocket.HandshakeTimeout, 10*time.Second)
	bufferSize := cfg.WebSocket.BufferSize
	if bufferSize == 0 {
		bufferSize = 1000
	}
	commandQueueSize := cfg.WebSocket.CommandQueueSize
	commandTimeout := parseDuration(cfg.WebSocket.CommandTimeout, 30*time.Second)

	wsConfig := commands.Config{
		URL:                  wsURL,
		ServerID:             cfg.Server.ServerID,
		ServerKey:            cfg.Server.SecretKey,
		ReconnectInterval:    reconnectInterval,
		MaxReconnectAttempts: maxReconnectAttempts,
		PingInterval:         pingInterval,
		WriteTimeout:         writeTimeout,
		ReadTimeout:          readTimeout,
		HandshakeTimeout:     handshakeTimeout,
		BufferSize:           bufferSize,
		EnableCompression:    cfg.WebSocket.EnableCompression,
		CommandQueueSize:     commandQueueSize,
		CommandTimeout:       commandTimeout,
		APIURL:               cfg.API.BaseURL,
		APIKey:               cfg.API.APIKey,
	}

	return commands.NewWebSocketCommandConsumer(wsConfig, handler, logger), nil
}

// parseDuration парсит строку duration с fallback
func parseDuration(str string, fallback time.Duration) time.Duration {
	if str == "" {
		return fallback
	}
	if duration, err := time.ParseDuration(str); err == nil {
		return duration
	}
	return fallback
}

// New создает новый агент
func New(cfg *config.AgentConfig, logger *logrus.Logger) (*Agent, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize WebSocket components (always enabled now)
	wsPub, err := initializeWebSocketPublisher(cfg, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось инициализировать WebSocket publisher: %v", err)
	}

	// Create temp agent for command handling
	tempAgent := &Agent{
		config:        cfg,
		logger:        logger,
		cpuMetrics:    metrics.NewCPUMetrics(),
		systemMonitor: metrics.NewSystemMonitor(logger),
		dockerClient:  docker.NewClient(logger),
	}

	wsCons, err := initializeWebSocketCommandConsumer(cfg, tempAgent, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось инициализировать WebSocket command consumer: %v", err)
	}

	logger.Info("WebSocket components initialized")

	// Create agent instance
	agent := &Agent{
		config:            cfg,
		logger:            logger,
		wsPublisher:       wsPub,
		wsCommandConsumer: wsCons,
		useWebSocket:      true,
		cpuMetrics:        metrics.NewCPUMetrics(),
		systemMonitor:     metrics.NewSystemMonitor(logger),
		dockerClient:      docker.NewClient(logger),
		ctx:               ctx,
		cancel:            cancel,
	}

	return agent, nil
}

// Start запускает агент
func (a *Agent) Start() error {
	a.logger.WithFields(logrus.Fields{
		"server_name":   a.config.Server.Name,
		"secret_key":    a.config.Server.SecretKey,
		"use_websocket": a.useWebSocket,
	}).Info("Запуск агента ServerEye")

	if a.useWebSocket {
		// Start WebSocket components
		a.logger.Info("Starting with WebSocket command mode")

		// Start WebSocket publisher
		if a.wsPublisher != nil {
			if err := a.wsPublisher.Start(a.ctx); err != nil {
				a.logger.WithError(err).Error("WebSocket publisher failed to start")
			}
		}

		// Start WebSocket command consumer
		if a.wsCommandConsumer != nil {
			go func() {
				if err := a.wsCommandConsumer.Start(a.ctx); err != nil {
					a.logger.WithError(err).Error("WebSocket command consumer failed")
				}
			}()
		}
	}

	// Запускаем heartbeat
	go a.startHeartbeat()

	// Запускаем сборщик метрик
	if a.useWebSocket && a.wsPublisher != nil {
		a.logger.Info("Starting metrics collection")
		go a.collectAndSendMetrics()
	} else {
		a.logger.Warn("No WebSocket publisher available - metrics disabled")
	}

	return nil
}

// Stop останавливает агент
func (a *Agent) Stop() error {
	a.logger.Info("Остановка агента")
	a.cancel()

	// Закрываем WebSocket компоненты
	if a.wsPublisher != nil {
		if err := a.wsPublisher.Close(); err != nil {
			a.logger.WithError(err).Error("Ошибка при закрытии WebSocket publisher")
		}
	}

	if a.wsCommandConsumer != nil {
		if err := a.wsCommandConsumer.Stop(); err != nil {
			a.logger.WithError(err).Error("Ошибка при закрытии WebSocket command consumer")
		}
	}

	return nil
}

// HandleCommand реализует интерфейс CommandHandler для WebSocket
func (a *Agent) HandleCommand(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	a.logger.WithFields(logrus.Fields{
		"command_id":   msg.ID,
		"command_type": msg.Type,
	}).Info("Handling command")

	// Создаем базовый response
	response := &protocol.Message{
		ID:        msg.ID,
		Timestamp: time.Now(),
		ServerID:  a.config.Server.Name,
		ServerKey: a.config.Server.SecretKey,
	}

	// Обрабатываем команду
	switch msg.Type {
	case "ping":
		response.Type = "ping_response"
		response.Payload = map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"message":   "pong",
		}
	case "restart":
		response.Type = "error_response"
		response.Payload = map[string]interface{}{
			"error": "restart command not implemented",
		}
	case "update":
		response.Type = "error_response"
		response.Payload = map[string]interface{}{
			"error": "update command not implemented",
		}
	default:
		response.Type = "error_response"
		response.Payload = map[string]interface{}{
			"error": fmt.Sprintf("unknown command type: %s", msg.Type),
		}
	}

	return response, nil
}

// Command handlers are in separate files:
// - handlers.go: All command handlers
