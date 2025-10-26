package bot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/servereye/servereye/pkg/protocol"
)

// TelegramAPI defines the interface for Telegram bot operations
type TelegramAPI interface {
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
	StopReceivingUpdates()
}

// RedisClient defines the interface for Redis operations
type RedisClient interface {
	Subscribe(ctx context.Context, channel string) (Subscription, error)
	Publish(ctx context.Context, channel string, message []byte) error
	Close() error
}

// Subscription defines the interface for Redis subscriptions
type Subscription interface {
	Channel() <-chan []byte
	Close() error
}

// Database defines the interface for database operations
type Database interface {
	RegisterUser(user *tgbotapi.User) error
	GetUserServers(userID int64) ([]string, error)
	GetUserServersWithInfo(userID int64) ([]ServerInfo, error)
	ConnectServer(userID int64, serverKey string) error
	ConnectServerWithName(userID int64, serverKey, serverName string) error
	RenameServer(serverKey, newName string) error
	RemoveServer(userID int64, serverKey string) error
	InitSchema() error
	Close() error
}

// AgentClient defines the interface for agent communication
type AgentClient interface {
	GetCPUTemperature(ctx context.Context, serverKey string) (float64, error)
	GetContainers(ctx context.Context, serverKey string) (*protocol.ContainersPayload, error)
	GetMemoryInfo(ctx context.Context, serverKey string) (*protocol.MemoryInfo, error)
	GetDiskInfo(ctx context.Context, serverKey string) (*protocol.DiskInfoPayload, error)
	GetUptime(ctx context.Context, serverKey string) (*protocol.UptimeInfo, error)
	GetProcesses(ctx context.Context, serverKey string) (*protocol.ProcessesPayload, error)
	SendContainerAction(ctx context.Context, serverKey string, messageType protocol.MessageType, payload protocol.ContainerActionPayload) (*protocol.ContainerActionResponse, error)
}

// Logger defines the interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, err error, fields ...Field)
	Fatal(msg string, err error, fields ...Field)
}

// Field represents a structured log field
type Field struct {
	Key   string
	Value interface{}
}

// MessageHandler defines the interface for handling different message types
type MessageHandler interface {
	HandleMessage(ctx context.Context, message *tgbotapi.Message) (string, error)
	HandleCallbackQuery(ctx context.Context, query *tgbotapi.CallbackQuery) error
}

// CommandRouter defines the interface for command routing
type CommandRouter interface {
	Route(command string) (CommandHandler, bool)
	RegisterHandler(command string, handler CommandHandler)
}

// CommandHandler defines the interface for individual command handlers
type CommandHandler interface {
	Handle(ctx context.Context, message *tgbotapi.Message) (string, error)
	Description() string
	Usage() string
}

// Validator defines the interface for input validation
type Validator interface {
	ValidateServerKey(key string) error
	ValidateServerName(name string) error
	ValidateContainerID(id string) error
}

// Metrics defines the interface for metrics collection
type Metrics interface {
	IncrementCommand(command string)
	IncrementError(errorType string)
	RecordLatency(operation string, duration float64)
	RecordActiveUsers(count int64)
}
