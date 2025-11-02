package redis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// HTTPClient implements Redis operations over HTTP
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
}

// HTTPConfig configuration for HTTP Redis client
type HTTPConfig struct {
	BaseURL string
	Timeout time.Duration
}

// NewHTTPClient creates a new HTTP Redis client
func NewHTTPClient(config HTTPConfig, logger *logrus.Logger) (*HTTPClient, error) {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &HTTPClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: logger,
	}, nil
}

// PublishRequest represents a publish request
type PublishRequest struct {
	Channel string `json:"channel"`
	Message string `json:"message"`
}

// SubscribeRequest represents a subscribe request
type SubscribeRequest struct {
	Channel string `json:"channel"`
	Timeout int    `json:"timeout,omitempty"`
}

// HTTPResponse represents HTTP API response
type HTTPResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Channel string `json:"channel,omitempty"`
}

// HTTPSubscription implements subscription over HTTP
type HTTPSubscription struct {
	channel  string
	client   *HTTPClient
	msgChan  chan []byte
	ctx      context.Context
	cancel   context.CancelFunc
	closeOnce sync.Once
}

// Channel returns the message channel
func (s *HTTPSubscription) Channel() <-chan []byte {
	return s.msgChan
}

// Close closes the subscription
func (s *HTTPSubscription) Close() error {
	s.cancel()
	s.closeOnce.Do(func() {
		close(s.msgChan)
	})
	return nil
}

// Publish publishes a message to Redis via HTTP
func (c *HTTPClient) Publish(ctx context.Context, channel string, message []byte) error {
	req := PublishRequest{
		Channel: channel,
		Message: string(message),
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/redis/publish", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var response HTTPResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("publish failed: %s", response.Message)
	}

	c.logger.WithFields(logrus.Fields{
		"channel": channel,
		"size":    len(message),
	}).Debug("Message published via HTTP")

	return nil
}

// Subscribe subscribes to a Redis channel via HTTP
func (c *HTTPClient) Subscribe(ctx context.Context, channel string) (*HTTPSubscription, error) {
	// Removed delay - need to catch commands immediately

	subCtx, cancel := context.WithCancel(ctx)

	subscription := &HTTPSubscription{
		channel: channel,
		client:  c,
		msgChan: make(chan []byte, 100),
		ctx:     subCtx,
		cancel:  cancel,
	}

	// Start polling for messages in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.logger.WithField("panic", r).Error("Panic in HTTP subscription goroutine")
			}
		}()

		c.logger.WithField("channel", channel).Info("Started HTTP subscription polling")

		for {
			select {
			case <-subCtx.Done():
				return
			default:
				// Poll for messages
				if err := c.pollForMessage(subCtx, subscription); err != nil {
					if subCtx.Err() != nil {
						return // Context cancelled
					}
					c.logger.WithError(err).Error("Error polling for messages")
					time.Sleep(5 * time.Second) // Wait before retry
				} else {
					// Minimal delay between successful polls to not miss commands
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}()

	return subscription, nil
}

// pollForMessage polls for a single message
func (c *HTTPClient) pollForMessage(ctx context.Context, subscription *HTTPSubscription) error {
	req := SubscribeRequest{
		Channel: subscription.channel,
		Timeout: 30, // 30 seconds timeout
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/redis/subscribe", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var response HTTPResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Success && response.Message != "" {
		// Got a message, send it to the channel
		select {
		case subscription.msgChan <- []byte(response.Message):
			c.logger.WithFields(logrus.Fields{
				"channel": subscription.channel,
				"size":    len(response.Message),
			}).Debug("Message received via HTTP")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	// If no message or timeout, just continue polling

	return nil
}

// Close closes the HTTP client
func (c *HTTPClient) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}
