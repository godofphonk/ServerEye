package streams

import (
	"context"
	"time"
)

// StreamMessage represents a message in a Redis Stream
type StreamMessage struct {
	ID      string            // Redis stream message ID (e.g., "1234567890-0")
	Values  map[string]string // Message payload
	Stream  string            // Stream name
	Retries int               // Number of delivery attempts
}

// StreamClient defines the interface for Redis Streams operations
type StreamClient interface {
	// Producer operations
	AddMessage(ctx context.Context, stream string, values map[string]string) (string, error)

	// Consumer operations (without consumer groups)
	ReadMessages(ctx context.Context, stream string, lastID string, count int64, block time.Duration) ([]StreamMessage, error)

	// Consumer Group operations
	CreateConsumerGroup(ctx context.Context, stream, group string) error
	ReadGroupMessages(ctx context.Context, stream, group, consumer string, count int64, block time.Duration) ([]StreamMessage, error)
	AckMessage(ctx context.Context, stream, group, messageID string) error

	// Stream management
	TrimStream(ctx context.Context, stream string, maxLen int64) error
	GetStreamLength(ctx context.Context, stream string) (int64, error)

	// Health check
	Ping(ctx context.Context) error
}

// Config holds configuration for Streams client
type Config struct {
	// Redis connection
	Addr     string
	Password string
	DB       int

	// Stream settings
	MaxRetries      int           // Maximum delivery attempts
	BlockDuration   time.Duration // How long to block waiting for messages
	BatchSize       int64         // Number of messages to read at once
	StreamMaxLength int64         // Maximum stream length (for trimming)

	// Consumer group settings
	ConsumerGroup string // Consumer group name
	ConsumerName  string // Consumer instance name
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	return &Config{
		MaxRetries:      3,
		BlockDuration:   5 * time.Second,
		BatchSize:       10,
		StreamMaxLength: 1000,
		ConsumerGroup:   "servereye-consumers",
		ConsumerName:    "consumer-1",
	}
}
