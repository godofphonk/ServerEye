//go:build wireinject
// +build wireinject

package agent

import (
	"context"
	"time"

	"github.com/godofphonk/ServerEye/internal/config"
	"github.com/godofphonk/ServerEye/internal/interfaces"
	"github.com/godofphonk/ServerEye/pkg/commands"
	"github.com/godofphonk/ServerEye/pkg/docker"
	"github.com/godofphonk/ServerEye/pkg/metrics"
	"github.com/godofphonk/ServerEye/pkg/websocket"
	"github.com/google/wire"
	"github.com/sirupsen/logrus"
)

// ProviderSet is a collection of providers for the agent package
var ProviderSet = wire.NewSet(
	provideLogger,
	provideUnifiedConfig,
	provideWebSocketClient,
	provideMetricsCollector,
	provideSystemMonitor,
	provideDockerManager,
	provideMetricsPublisher,
	provideStaticInfoPublisher,
	provideCommandConsumer,
	provideAgent,
	provideConfigValidator,
	provideCommandHandler,
)

// InitializeAgent creates a new agent with dependency injection using Google Wire
func InitializeAgent(ctx context.Context, configPath string) (*Agent, error) {
	wire.Build(
		ProviderSet,
	)
	return nil, nil // This will be replaced by Wire
}

// InitializeAgentEnhanced creates a new agent with enhanced configuration
func InitializeAgentEnhanced(ctx context.Context, configPath string) (*Agent, error) {
	wire.Build(
		ProviderSet,
		wire.Bind(new(interfaces.CommandHandlerInterface), new(*Agent)),
	)
	return nil, nil // This will be replaced by Wire
}

// provideLogger creates a new logger instance with proper configuration
func provideLogger() interfaces.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
	return interfaces.NewLogrusAdapter(logger)
}

// provideUnifiedConfig creates a new unified configuration
func provideUnifiedConfig(configPath string) (*config.UnifiedConfig, error) {
	return config.LoadUnifiedConfig(configPath)
}

// provideConfigValidator creates a configuration validator
func provideConfigValidator() *config.ConfigValidator {
	return config.NewConfigValidator()
}

// provideCommandHandler creates a command handler (will be the agent itself)
func provideCommandHandler(agent *Agent) interfaces.CommandHandlerInterface {
	return agent
}

// provideWebSocketClient creates a new WebSocket client with unified config
func provideWebSocketClient(cfg *config.UnifiedConfig, logger interfaces.Logger) *websocket.Client {
	// Convert adapter back to logrus for WebSocket client
	logrusLogger := logger.(*interfaces.LogrusAdapter).Entry.Logger

	wsConfig := websocket.Config{
		URL:                  cfg.GetWebSocketURL(),
		ServerID:             cfg.Server.ServerID,
		ServerKey:            cfg.Server.SecretKey,
		ReconnectInterval:    parseDuration(cfg.WebSocket.ReconnectInterval, 5*time.Second),
		MaxReconnectAttempts: cfg.WebSocket.MaxReconnectAttempts,
		PingInterval:         parseDuration(cfg.WebSocket.PingInterval, 30*time.Second),
		WriteTimeout:         parseDuration(cfg.WebSocket.WriteTimeout, 10*time.Second),
		ReadTimeout:          parseDuration(cfg.WebSocket.ReadTimeout, 10*time.Second),
		HandshakeTimeout:     parseDuration(cfg.WebSocket.HandshakeTimeout, 10*time.Second),
		BufferSize:           cfg.WebSocket.BufferSize,
		EnableCompression:    cfg.WebSocket.EnableCompression,
		APIURL:               cfg.API.BaseURL,
		APIKey:               cfg.API.APIKey,
	}

	return websocket.NewClient(wsConfig, logrusLogger)
}

// provideMetricsCollector creates a new metrics collector
func provideMetricsCollector() interfaces.MetricsCollector {
	return metrics.NewCPUMetrics()
}

// provideSystemMonitor creates a new system monitor
func provideSystemMonitor(logger interfaces.Logger) interfaces.SystemMonitor {
	// Convert adapter back to logrus for now (temporary solution)
	logrusLogger := logger.(*interfaces.LogrusAdapter).Entry.Logger
	monitor := metrics.NewSystemMonitor(logrusLogger)
	return interfaces.NewSystemMonitorAdapter(monitor)
}

// provideDockerManager creates a new Docker manager
func provideDockerManager(logger interfaces.Logger) interfaces.DockerManager {
	// Convert adapter back to logrus for now (temporary solution)
	logrusLogger := logger.(*interfaces.LogrusAdapter).Entry.Logger
	client := docker.NewClient(logrusLogger)
	return interfaces.NewDockerManagerAdapter(client)
}

// provideMetricsPublisher creates a new metrics publisher with unified config
func provideMetricsPublisher(cfg *config.UnifiedConfig, logger interfaces.Logger) interfaces.MetricsPublisher {
	// Convert adapter back to logrus for metrics publisher
	logrusLogger := logger.(*interfaces.LogrusAdapter).Entry.Logger

	metricsConfig := metrics.Config{
		URL:                  cfg.GetWebSocketURL(),
		ServerID:             cfg.Server.ServerID,
		ServerKey:            cfg.Server.SecretKey,
		ReconnectInterval:    parseDuration(cfg.WebSocket.ReconnectInterval, 5*time.Second),
		MaxReconnectAttempts: cfg.WebSocket.MaxReconnectAttempts,
		PingInterval:         parseDuration(cfg.WebSocket.PingInterval, 30*time.Second),
		WriteTimeout:         parseDuration(cfg.WebSocket.WriteTimeout, 10*time.Second),
		ReadTimeout:          parseDuration(cfg.WebSocket.ReadTimeout, 10*time.Second),
		HandshakeTimeout:     parseDuration(cfg.WebSocket.HandshakeTimeout, 10*time.Second),
		BufferSize:           cfg.WebSocket.BufferSize,
		EnableCompression:    cfg.WebSocket.EnableCompression,
		MetricBufferSize:     cfg.WebSocket.MetricBufferSize,
		MetricBufferFlush:    parseDuration(cfg.WebSocket.MetricBufferFlush, 30*time.Second),
		APIURL:               cfg.API.BaseURL,
		APIKey:               cfg.API.APIKey,
	}

	return metrics.NewWebSocketPublisher(metricsConfig, logrusLogger)
}

// provideCommandConsumer creates a new command consumer with unified config
func provideCommandConsumer(cfg *config.UnifiedConfig, logger interfaces.Logger, handler interfaces.CommandHandlerInterface) interfaces.CommandConsumer {
	// Convert adapter back to logrus for command consumer
	logrusLogger := logger.(*interfaces.LogrusAdapter).Entry.Logger

	commandsConfig := commands.Config{
		URL:                  cfg.GetWebSocketURL(),
		ServerID:             cfg.Server.ServerID,
		ServerKey:            cfg.Server.SecretKey,
		ReconnectInterval:    parseDuration(cfg.WebSocket.ReconnectInterval, 5*time.Second),
		MaxReconnectAttempts: cfg.WebSocket.MaxReconnectAttempts,
		PingInterval:         parseDuration(cfg.WebSocket.PingInterval, 30*time.Second),
		WriteTimeout:         parseDuration(cfg.WebSocket.WriteTimeout, 10*time.Second),
		ReadTimeout:          parseDuration(cfg.WebSocket.ReadTimeout, 10*time.Second),
		HandshakeTimeout:     parseDuration(cfg.WebSocket.HandshakeTimeout, 10*time.Second),
		BufferSize:           cfg.WebSocket.BufferSize,
		EnableCompression:    cfg.WebSocket.EnableCompression,
		CommandQueueSize:     cfg.WebSocket.CommandQueueSize,
		CommandTimeout:       parseDuration(cfg.WebSocket.CommandTimeout, 30*time.Second),
		APIURL:               cfg.API.BaseURL,
		APIKey:               cfg.API.APIKey,
	}

	return commands.NewWebSocketCommandConsumer(commandsConfig, handler, logrusLogger)
}

// provideAgent creates a new agent with all dependencies injected
func provideAgent(
	ctx context.Context,
	configPath string,
	cfg *config.UnifiedConfig,
	logger interfaces.Logger,
	metricsCollector interfaces.MetricsCollector,
	systemMonitor interfaces.SystemMonitor,
	dockerManager interfaces.DockerManager,
	metricsPublisher interfaces.MetricsPublisher,
	commandConsumer interfaces.CommandConsumer,
	staticInfoPublisher *metrics.StaticInfoPublisher,
) (*Agent, error) {
	// Convert unified config to legacy config for backward compatibility
	legacyConfig := cfg.ToAgentConfig()

	// Convert adapter back to logrus for agent
	logrusLogger := logger.(*interfaces.LogrusAdapter).Entry.Logger

	agent := &Agent{
		config:            legacyConfig,
		logger:            interfaces.NewLogrusAdapter(logrusLogger),
		wsPublisher:       metricsPublisher,
		wsCommandConsumer: commandConsumer,
		useWebSocket:      cfg.WebSocket.Enabled,
		cpuMetrics:        metricsCollector,
		systemMonitor:     systemMonitor,
		dockerClient:      dockerManager,
		staticInfoPublisher: staticInfoPublisher,
		startTime:         time.Now(),
	}

	// Create context for the agent
	agent.ctx, agent.cancel = context.WithCancel(ctx)

	return agent, nil
}

// parseDuration parses a duration string with fallback
func parseDuration(str string, fallback time.Duration) time.Duration {
	if str == "" {
		return fallback
	}
	if duration, err := time.ParseDuration(str); err == nil {
		return duration
	}
	return fallback
}
}

// provideStaticInfoPublisher creates static info publisher
func provideStaticInfoPublisher(
	cfg *config.UnifiedConfig,
	logger interfaces.Logger,
) (*metrics.StaticInfoPublisher, error) {
	logrusLogger := logger.(*interfaces.LogrusAdapter).Entry.Logger
	
	staticConfig := metrics.StaticInfoConfig{
		APIURL:     cfg.API.BaseURL,
		APIKey:     cfg.API.APIKey,
		ServerID:   cfg.Server.ServerID,
		ServerKey:  cfg.Server.SecretKey,
		ServerName: cfg.Server.Name,
		Interval:   24 * time.Hour,
	}
	
	return metrics.NewStaticInfoPublisher(staticConfig, logrusLogger), nil
}
