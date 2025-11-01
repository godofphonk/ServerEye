package bot

import (
	"context"
	"database/sql"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/servereye/servereye/pkg/redis"
	"github.com/sirupsen/logrus"
)

// LogrusAdapter adapts logrus.Logger to our Logger interface
type LogrusAdapter struct {
	logger *logrus.Logger
}

// NewLogrusAdapter creates a new logrus adapter
func NewLogrusAdapter(logger *logrus.Logger) *LogrusAdapter {
	return &LogrusAdapter{logger: logger}
}

// WithField returns a new logger with the given field
func (l *LogrusAdapter) WithField(key string, value interface{}) *logrus.Entry {
	return l.logger.WithField(key, value)
}

// WithFields returns a new logger with the given fields
func (l *LogrusAdapter) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.logger.WithFields(fields)
}

// WithError returns a new logger with the given error
func (l *LogrusAdapter) WithError(err error) *logrus.Entry {
	return l.logger.WithError(err)
}

// Info logs an info message
func (l *LogrusAdapter) Info(msg string) {
	l.logger.Info(msg)
}

// Error logs an error message
func (l *LogrusAdapter) Error(msg string) {
	l.logger.Error(msg)
}

// Debug logs a debug message
func (l *LogrusAdapter) Debug(msg string) {
	l.logger.Debug(msg)
}

// Warn logs a warning message
func (l *LogrusAdapter) Warn(msg string) {
	l.logger.Warn(msg)
}

// DatabaseAdapter adapts sql.DB to our Database interface
type DatabaseAdapter struct {
	db  *sql.DB
	bot *Bot // Reference to bot for existing methods
}

// NewDatabaseAdapter creates a new database adapter
func NewDatabaseAdapter(db *sql.DB, bot *Bot) *DatabaseAdapter {
	return &DatabaseAdapter{db: db, bot: bot}
}

// RegisterUser registers a new user
func (d *DatabaseAdapter) RegisterUser(user *tgbotapi.User) error {
	return d.bot.registerUser(user)
}

// GetUserServers gets user servers
func (d *DatabaseAdapter) GetUserServers(userID int64) ([]string, error) {
	return d.bot.getUserServers(userID)
}

// GetUserServersWithInfo gets user servers with info
func (d *DatabaseAdapter) GetUserServersWithInfo(userID int64) ([]ServerInfo, error) {
	return d.bot.getUserServersWithInfo(userID)
}

// ConnectServer connects a server
func (d *DatabaseAdapter) ConnectServer(userID int64, serverKey string) error {
	return d.bot.connectServer(userID, serverKey)
}

// ConnectServerWithName connects a server with name
func (d *DatabaseAdapter) ConnectServerWithName(userID int64, serverKey, serverName string) error {
	return d.bot.connectServerWithName(userID, serverKey, serverName)
}

// RenameServer renames a server
func (d *DatabaseAdapter) RenameServer(serverKey, newName string) error {
	return d.bot.renameServer(serverKey, newName)
}

// RemoveServer removes a server
func (d *DatabaseAdapter) RemoveServer(userID int64, serverKey string) error {
	return d.bot.removeServer(userID, serverKey)
}

// InitSchema initializes the database schema
func (d *DatabaseAdapter) InitSchema() error {
	return d.bot.initDatabase()
}

// Close closes the database connection
func (d *DatabaseAdapter) Close() error {
	return d.db.Close()
}

// RedisAdapter adapts redis.Client to our RedisClient interface
type RedisAdapter struct {
	client *redis.Client
}

// NewRedisAdapter creates a new Redis adapter
func NewRedisAdapter(client *redis.Client) *RedisAdapter {
	return &RedisAdapter{client: client}
}

// Subscribe subscribes to a Redis channel
func (r *RedisAdapter) Subscribe(ctx context.Context, channel string) (Subscription, error) {
	sub, err := r.client.Subscribe(ctx, channel)
	if err != nil {
		return nil, err
	}
	return &SubscriptionAdapter{sub: sub}, nil
}

// Publish publishes a message to a Redis channel
func (r *RedisAdapter) Publish(ctx context.Context, channel string, message []byte) error {
	return r.client.Publish(ctx, channel, message)
}

// Close closes the Redis connection
func (r *RedisAdapter) Close() error {
	return r.client.Close()
}

// SubscriptionAdapter adapts redis subscription to our Subscription interface
type SubscriptionAdapter struct {
	sub interface {
		Channel() <-chan []byte
		Close() error
	}
}

// Channel returns the subscription channel
func (s *SubscriptionAdapter) Channel() <-chan []byte {
	return s.sub.Channel()
}

// Close closes the subscription
func (s *SubscriptionAdapter) Close() error {
	defer func() {
		if r := recover(); r != nil {
			// Ignore panic from closing already closed channel
		}
	}()
	return s.sub.Close()
}

// AgentClientAdapter adapts existing agent methods to AgentClient interface
type AgentClientAdapter struct {
	bot *Bot
}

// NewAgentClientAdapter creates a new agent client adapter
func NewAgentClientAdapter(bot *Bot) *AgentClientAdapter {
	return &AgentClientAdapter{bot: bot}
}

// GetCPUTemperature gets CPU temperature from agent
func (a *AgentClientAdapter) GetCPUTemperature(ctx context.Context, serverKey string) (float64, error) {
	return a.bot.getCPUTemperature(serverKey)
}

// GetContainers gets containers from agent
func (a *AgentClientAdapter) GetContainers(ctx context.Context, serverKey string) (*protocol.ContainersPayload, error) {
	return a.bot.getContainers(serverKey)
}

// GetMemoryInfo gets memory info from agent
func (a *AgentClientAdapter) GetMemoryInfo(ctx context.Context, serverKey string) (*protocol.MemoryInfo, error) {
	return a.bot.getMemoryInfo(serverKey)
}

// GetDiskInfo gets disk info from agent
func (a *AgentClientAdapter) GetDiskInfo(ctx context.Context, serverKey string) (*protocol.DiskInfoPayload, error) {
	return a.bot.getDiskInfo(serverKey)
}

// GetUptime gets uptime from agent
func (a *AgentClientAdapter) GetUptime(ctx context.Context, serverKey string) (*protocol.UptimeInfo, error) {
	return a.bot.getUptime(serverKey)
}

// GetProcesses gets processes from agent
func (a *AgentClientAdapter) GetProcesses(ctx context.Context, serverKey string) (*protocol.ProcessesPayload, error) {
	return a.bot.getProcesses(serverKey)
}

// SendContainerAction sends container action to agent
func (a *AgentClientAdapter) SendContainerAction(ctx context.Context, serverKey string, messageType protocol.MessageType, payload protocol.ContainerActionPayload) (*protocol.ContainerActionResponse, error) {
	return a.bot.sendContainerAction(serverKey, messageType, payload)
}
