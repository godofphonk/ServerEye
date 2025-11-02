package streams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// HTTPStreamClient implements StreamClient over HTTP
type HTTPStreamClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewHTTPStreamClient creates HTTP-based streams client
func NewHTTPStreamClient(baseURL string, logger *logrus.Logger) *HTTPStreamClient {
	return &HTTPStreamClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 35 * time.Second, // Longer for blocking reads
		},
		logger: logger,
	}
}

// AddMessage adds message via HTTP
func (c *HTTPStreamClient) AddMessage(ctx context.Context, stream string, values map[string]string) (string, error) {
	req := map[string]interface{}{
		"stream": stream,
		"values": values,
	}

	var resp struct {
		ID string `json:"id"`
	}

	if err := c.doRequest(ctx, "/api/streams/xadd", req, &resp); err != nil {
		return "", err
	}

	return resp.ID, nil
}

// ReadMessages reads messages via HTTP
func (c *HTTPStreamClient) ReadMessages(ctx context.Context, stream string, lastID string, count int64, block time.Duration) ([]StreamMessage, error) {
	req := map[string]interface{}{
		"stream":   stream,
		"last_id":  lastID,
		"count":    count,
		"block_ms": block.Milliseconds(),
	}

	var resp struct {
		Streams []struct {
			Stream   string
			Messages []struct {
				ID     string
				Values map[string]interface{}
			}
		}
	}

	if err := c.doRequest(ctx, "/api/streams/xread", req, &resp); err != nil {
		return nil, err
	}

	var messages []StreamMessage
	for _, s := range resp.Streams {
		for _, msg := range s.Messages {
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

// CreateConsumerGroup creates group via HTTP (not implemented in HTTP API yet)
func (c *HTTPStreamClient) CreateConsumerGroup(ctx context.Context, stream, group string) error {
	c.logger.Warn("CreateConsumerGroup not implemented in HTTP API")
	return nil
}

// ReadGroupMessages reads from consumer group via HTTP
func (c *HTTPStreamClient) ReadGroupMessages(ctx context.Context, stream, group, consumer string, count int64, block time.Duration) ([]StreamMessage, error) {
	req := map[string]interface{}{
		"stream":   stream,
		"group":    group,
		"consumer": consumer,
		"count":    count,
		"block_ms": block.Milliseconds(),
	}

	var resp struct {
		Streams []struct {
			Stream   string
			Messages []struct {
				ID     string
				Values map[string]interface{}
			}
		}
	}

	if err := c.doRequest(ctx, "/api/streams/xreadgroup", req, &resp); err != nil {
		return nil, err
	}

	var messages []StreamMessage
	for _, s := range resp.Streams {
		for _, msg := range s.Messages {
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

// AckMessage acknowledges message via HTTP
func (c *HTTPStreamClient) AckMessage(ctx context.Context, stream, group, messageID string) error {
	req := map[string]interface{}{
		"stream": stream,
		"group":  group,
		"id":     messageID,
	}

	var resp struct {
		Success bool `json:"success"`
	}

	return c.doRequest(ctx, "/api/streams/xack", req, &resp)
}

// TrimStream trims stream (not implemented)
func (c *HTTPStreamClient) TrimStream(ctx context.Context, stream string, maxLen int64) error {
	return nil
}

// GetStreamLength gets stream length (not implemented)
func (c *HTTPStreamClient) GetStreamLength(ctx context.Context, stream string) (int64, error) {
	return 0, nil
}

// Ping checks connection
func (c *HTTPStreamClient) Ping(ctx context.Context) error {
	return nil
}

// doRequest performs HTTP request
func (c *HTTPStreamClient) doRequest(ctx context.Context, endpoint string, reqBody, respBody interface{}) error {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return err
		}
	}

	return nil
}
