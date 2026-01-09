package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/godofphonk/ServerEye/pkg/types"
	"github.com/sirupsen/logrus"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *logrus.Logger
}
type Config struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Timeout int    `yaml:"timeout" default:"30"`
}

func New(cfg Config, logger *logrus.Logger) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // Отключаем проверку сертификата для IP
				},
			},
		},
		logger: logger,
	}
}

func (c *Client) Publish(ctx context.Context, metric *types.Metric) error {
	const maxRetries = 3
	const baseDelay = 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := c.publishAttempt(ctx, metric)
		if err == nil {
			return nil
		}

		// Log retry attempt
		c.logger.WithFields(logrus.Fields{
			"type":        metric.Type,
			"server_id":   metric.ServerID,
			"attempt":     attempt + 1,
			"max_retries": maxRetries,
			"error":       err,
		}).Warn("HTTP request failed, retrying")

		// Don't retry on client errors (4xx)
		if isClientError(err) {
			return err
		}

		// Exponential backoff with jitter
		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(math.Pow(2, float64(attempt)))
			jitter := time.Duration(float64(delay) * 0.1 * (2.0*float64(time.Now().UnixNano()%1000)/1000.0 - 1.0))
			time.Sleep(delay + jitter)
		}
	}

	return fmt.Errorf("failed after %d retries", maxRetries)
}

func (c *Client) publishAttempt(ctx context.Context, metric *types.Metric) error {
	// Prepare request payload
	payload := map[string]interface{}{
		"server_id":  metric.ServerID,
		"server_key": metric.ServerKey,
		"type":       metric.Type,
		"value":      metric.Value,
		"timestamp":  metric.Timestamp,
		"tags":       metric.Tags,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	// Create HTTP request with shorter timeout for individual attempts
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/v1/metrics", c.baseURL)
	req, err := http.NewRequestWithContext(reqCtx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusAccepted {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return fmt.Errorf("API error: status %d, response: %v", resp.StatusCode, errorResp)
	}

	c.logger.WithFields(logrus.Fields{
		"type":      metric.Type,
		"server_id": metric.ServerID,
		"status":    "accepted",
	}).Info("Metric published via HTTP API")

	return nil
}

func isClientError(err error) bool {
	// Check if error message contains 4xx status code
	errStr := err.Error()
	return len(errStr) > 0 && errStr[len(errStr)-3] == '4' &&
		errStr[len(errStr)-2] >= '0' && errStr[len(errStr)-2] <= '9' &&
		errStr[len(errStr)-1] >= '0' && errStr[len(errStr)-1] <= '9'
}

func (c *Client) PublishBatch(ctx context.Context, metrics []*types.Metric) error {
	// For simplicity, publish metrics one by one
	// Could be optimized to send as batch if API supports it
	for _, metric := range metrics {
		if err := c.Publish(ctx, metric); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) Close() error {
	// HTTP client doesn't need explicit cleanup
	c.httpClient.CloseIdleConnections()
	return nil
}

func (c *Client) Name() string {
	return "http"
}
