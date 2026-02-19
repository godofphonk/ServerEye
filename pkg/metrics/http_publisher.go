package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/godofphonk/ServerEye/pkg/types"
	"github.com/sirupsen/logrus"
)

// HTTPPublisher publishes metrics via HTTP POST
type HTTPPublisher struct {
	client       *http.Client
	logger       *logrus.Logger
	serverKey    string
	serverID     string
	baseURL      string
	metricsURL   string
	heartbeatURL string
}

// NewHTTPPublisher creates a new HTTP publisher
func NewHTTPPublisher(serverKey, serverID, baseURL string, logger *logrus.Logger) *HTTPPublisher {
	return &HTTPPublisher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:       logger,
		serverKey:    serverKey,
		serverID:     serverID,
		baseURL:      baseURL,
		metricsURL:   fmt.Sprintf("%s/api/servers/by-key/%s/metrics", baseURL, serverKey),
		heartbeatURL: fmt.Sprintf("%s/api/servers/by-key/%s/heartbeat", baseURL, serverKey),
	}
}

// Publish publishes a metric via HTTP POST
func (p *HTTPPublisher) Publish(ctx context.Context, metric *types.Metric) error {
	switch metric.Type {
	case "metrics":
		return p.publishMetrics(ctx, metric)
	case "heartbeat":
		return p.publishHeartbeat(ctx)
	default:
		return fmt.Errorf("unsupported metric type: %s", metric.Type)
	}
}

// publishMetrics sends metrics via HTTP POST
func (p *HTTPPublisher) publishMetrics(ctx context.Context, metric *types.Metric) error {
	// Extract metrics data from the metric
	metricsData, ok := metric.Data["metrics"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid metrics data format")
	}

	// Create the request body according to API specification
	requestBody := map[string]interface{}{
		"metrics": metricsData,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"url":    p.metricsURL,
		"length": len(jsonBody),
	}).Debug("Sending metrics via HTTP")

	req, err := http.NewRequestWithContext(ctx, "POST", p.metricsURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	p.logger.WithFields(logrus.Fields{
		"status_code": resp.StatusCode,
		"server_id":   p.serverID,
	}).Debug("Metrics sent successfully via HTTP")

	return nil
}

// publishHeartbeat sends heartbeat via HTTP POST
func (p *HTTPPublisher) publishHeartbeat(ctx context.Context) error {
	// According to API, heartbeat can have empty body or {"status": "online"}
	requestBody := map[string]string{
		"status": "online",
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"url":       p.heartbeatURL,
		"server_id": p.serverID,
	}).Debug("Sending heartbeat via HTTP")

	req, err := http.NewRequestWithContext(ctx, "POST", p.heartbeatURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	p.logger.WithFields(logrus.Fields{
		"status_code": resp.StatusCode,
		"server_id":   p.serverID,
	}).Debug("Heartbeat sent successfully via HTTP")

	return nil
}

// Close closes the HTTP publisher (no-op for HTTP)
func (p *HTTPPublisher) Close() error {
	// No cleanup needed for HTTP client
	return nil
}
