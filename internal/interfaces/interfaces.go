package interfaces

import (
	"context"
	"time"

	"github.com/godofphonk/ServerEye/internal/config"
	"github.com/godofphonk/ServerEye/pkg/protocol"
	"github.com/godofphonk/ServerEye/pkg/types"
)

// Logger defines logging interface
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	WithError(err error) Logger
}

// Config defines configuration interface
type Config interface {
	LoadAgentConfig(filepath string) (*config.AgentConfig, error)
	Validate(cfg *config.AgentConfig) error
}

// MetricsCollector defines metrics collection interface
type MetricsCollector interface {
	GetTemperature() (float64, error)
	GetSensorInfo() string
}

// SystemMonitor defines system monitoring interface
type SystemMonitor interface {
	GetMemoryInfo() (*protocol.MemoryInfo, error)
	GetDiskInfo() (*protocol.DiskInfoPayload, error)
	GetUptime() (*protocol.UptimeInfo, error)
	GetProcesses() (*protocol.ProcessesPayload, error)
	GetNetworkInfo() (*protocol.NetworkInfo, error)
	GetSystemInfo() (*protocol.SystemInfoPayload, error)
}

// DockerManager defines Docker management interface
type DockerManager interface {
	GetContainers(ctx context.Context) (*protocol.ContainersPayload, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	RestartContainer(ctx context.Context, containerID string) error
	RemoveContainer(ctx context.Context, containerID string) error
	CreateContainer(ctx context.Context, config *protocol.CreateContainerPayload) error
	IsAvailable() bool
}

// MetricsPublisher defines metrics publishing interface
type MetricsPublisher interface {
	Start(ctx context.Context) error
	Publish(ctx context.Context, metric *types.Metric) error
	PublishBatch(ctx context.Context, metrics []*types.Metric) error
	Close() error
	Name() string
	IsConnected() bool
	GetMetrics() map[string]interface{}
}

// CommandConsumer defines command consumption interface
type CommandConsumer interface {
	Start(ctx context.Context) error
	Stop() error
	GetMetrics() map[string]interface{}
}

// CommandHandler defines command handling interface
type CommandHandler interface {
	HandleCommand(ctx context.Context, msg *protocol.Message) (*protocol.Message, error)
}

// WebSocketClient defines WebSocket client interface
type WebSocketClient interface {
	Start() error
	Close() error
	IsConnected() bool
	ServerID() string
	SendMessage(msg interface{}) error
	ReceiveMessage() <-chan interface{}
	RegisterCommandHandler(command string, handler interface{})
}

// Agent defines the main agent interface
type Agent interface {
	Start() error
	Stop() error
	HandleCommand(ctx context.Context, msg *protocol.Message) (*protocol.Message, error)
	CreateMetricFromData(metricType string, value interface{}, tags map[string]string) *types.Metric
}

// HealthChecker defines health checking interface
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
	IsHealthy() bool
	GetLastCheck() time.Time
}

// MetricsProvider defines metrics providing interface
type MetricsProvider interface {
	ProvideCPUMetrics(ctx context.Context) ([]*types.Metric, error)
	ProvideMemoryMetrics(ctx context.Context) ([]*types.Metric, error)
	ProvideDiskMetrics(ctx context.Context) ([]*types.Metric, error)
	ProvideContainerMetrics(ctx context.Context) ([]*types.Metric, error)
	ProvideSystemMetrics(ctx context.Context) ([]*types.Metric, error)
}
