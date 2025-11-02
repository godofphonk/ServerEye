package streams

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// Client implements StreamClient interface using go-redis
type Client struct {
	client *redis.Client
	config *Config
	logger *logrus.Logger
}

// NewClient creates a new Redis Streams client
func NewClient(config *Config, logger *logrus.Logger) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if logger == nil {
		logger = logrus.New()
	}

	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"addr": config.Addr,
		"db":   config.DB,
	}).Info("Redis Streams client connected")

	return &Client{
		client: client,
		config: config,
		logger: logger,
	}, nil
}

// AddMessage adds a message to a stream
func (c *Client) AddMessage(ctx context.Context, stream string, values map[string]string) (string, error) {
	start := time.Now()
	
	// XADD stream * field1 value1 field2 value2 ...
	args := &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}

	// Optional: limit stream length to prevent unbounded growth
	if c.config.StreamMaxLength > 0 {
		args.MaxLen = c.config.StreamMaxLength
		args.Approx = true // Use approximate trimming for better performance
	}

	id, err := c.client.XAdd(ctx, args).Result()
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"stream": stream,
			"error":  err,
		}).Error("Failed to add message to stream")
		return "", fmt.Errorf("XADD failed: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"stream":   stream,
		"id":       id,
		"duration": time.Since(start),
	}).Debug("Message added to stream")

	return id, nil
}

// ReadMessages reads messages from a stream (simple read, no consumer group)
func (c *Client) ReadMessages(ctx context.Context, stream string, lastID string, count int64, block time.Duration) ([]StreamMessage, error) {
	if lastID == "" {
		lastID = "0" // Start from beginning
	}

	if count <= 0 {
		count = c.config.BatchSize
	}

	args := &redis.XReadArgs{
		Streams: []string{stream, lastID},
		Count:   count,
		Block:   block,
	}

	streams, err := c.client.XRead(ctx, args).Result()
	if err != nil {
		if err == redis.Nil {
			// No messages available
			return nil, nil
		}
		return nil, fmt.Errorf("XREAD failed: %w", err)
	}

	var messages []StreamMessage
	for _, s := range streams {
		for _, msg := range s.Messages {
			// Convert map[string]interface{} to map[string]string
			values := make(map[string]string)
			for k, v := range msg.Values {
				if str, ok := v.(string); ok {
					values[k] = str
				}
			}
			messages = append(messages, StreamMessage{
				ID:     msg.ID,
				Values: values,
				Stream: s.Stream,
			})
		}
	}

	return messages, nil
}

// CreateConsumerGroup creates a consumer group for a stream
func (c *Client) CreateConsumerGroup(ctx context.Context, stream, group string) error {
	// Create group starting from the beginning ("0")
	// Use MKSTREAM to create the stream if it doesn't exist
	err := c.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil {
		// Ignore error if group already exists
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			c.logger.WithFields(logrus.Fields{
				"stream": stream,
				"group":  group,
			}).Debug("Consumer group already exists")
			return nil
		}
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"stream": stream,
		"group":  group,
	}).Info("Consumer group created")

	return nil
}

// ReadGroupMessages reads messages using a consumer group
func (c *Client) ReadGroupMessages(ctx context.Context, stream, group, consumer string, count int64, block time.Duration) ([]StreamMessage, error) {
	if count <= 0 {
		count = c.config.BatchSize
	}

	args := &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"}, // ">" means only new messages
		Count:    count,
		Block:    block,
	}

	streams, err := c.client.XReadGroup(ctx, args).Result()
	if err != nil {
		if err == redis.Nil {
			// No messages available
			return nil, nil
		}
		return nil, fmt.Errorf("XREADGROUP failed: %w", err)
	}

	var messages []StreamMessage
	for _, s := range streams {
		for _, msg := range s.Messages {
			// Convert map[string]interface{} to map[string]string
			values := make(map[string]string)
			for k, v := range msg.Values {
				if str, ok := v.(string); ok {
					values[k] = str
				}
			}
			messages = append(messages, StreamMessage{
				ID:     msg.ID,
				Values: values,
				Stream: s.Stream,
			})
		}
	}

	c.logger.WithFields(logrus.Fields{
		"stream":   stream,
		"group":    group,
		"consumer": consumer,
		"count":    len(messages),
	}).Debug("Messages read from consumer group")

	return messages, nil
}

// AckMessage acknowledges a message as processed
func (c *Client) AckMessage(ctx context.Context, stream, group, messageID string) error {
	err := c.client.XAck(ctx, stream, group, messageID).Err()
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"stream": stream,
			"group":  group,
			"id":     messageID,
			"error":  err,
		}).Error("Failed to ACK message")
		return fmt.Errorf("XACK failed: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"stream": stream,
		"id":     messageID,
	}).Debug("Message acknowledged")

	return nil
}

// TrimStream trims a stream to a maximum length
func (c *Client) TrimStream(ctx context.Context, stream string, maxLen int64) error {
	// Use approximate trimming (~) for better performance
	err := c.client.XTrimMaxLenApprox(ctx, stream, maxLen, 100).Err()
	if err != nil {
		return fmt.Errorf("XTRIM failed: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"stream": stream,
		"maxLen": maxLen,
	}).Debug("Stream trimmed")

	return nil
}

// GetStreamLength returns the number of messages in a stream
func (c *Client) GetStreamLength(ctx context.Context, stream string) (int64, error) {
	length, err := c.client.XLen(ctx, stream).Result()
	if err != nil {
		return 0, fmt.Errorf("XLEN failed: %w", err)
	}
	return length, nil
}

// Ping checks if Redis connection is alive
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}
