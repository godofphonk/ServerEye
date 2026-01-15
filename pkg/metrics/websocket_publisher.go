package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/godofphonk/ServerEye/pkg/types"
	"github.com/godofphonk/ServerEye/pkg/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketPublisher publishes metrics via WebSocket
type WebSocketPublisher struct {
	wsClient *websocket.Client
	adapter  *types.WebSocketAdapter
	logger   *logrus.Logger

	// Buffer for offline metrics
	buffer      []*types.Metric
	bufferMu    sync.Mutex
	bufferSize  int
	bufferFlush time.Duration

	// Metrics
	metricsMu     sync.RWMutex
	publishedCnt  int64
	failedCnt     int64
	lastPublished time.Time
}

// Config represents WebSocket publisher configuration
type Config struct {
	URL                  string        `yaml:"url"`
	ServerID             string        `yaml:"server_id"`
	ServerKey            string        `yaml:"server_key"`
	ReconnectInterval    time.Duration `yaml:"reconnect_interval" default:"5s"`
	MaxReconnectAttempts int           `yaml:"max_reconnect_attempts" default:"10"`
	PingInterval         time.Duration `yaml:"ping_interval" default:"30s"`
	WriteTimeout         time.Duration `yaml:"write_timeout" default:"10s"`
	ReadTimeout          time.Duration `yaml:"read_timeout" default:"10s"`
	HandshakeTimeout     time.Duration `yaml:"handshake_timeout" default:"10s"`
	BufferSize           int           `yaml:"buffer_size" default:"1000"`
	EnableCompression    bool          `yaml:"enable_compression" default:"true"`
	MetricBufferSize     int           `yaml:"metric_buffer_size" default:"100"`
	MetricBufferFlush    time.Duration `yaml:"metric_buffer_flush" default:"30s"`
	APIURL               string        `yaml:"api_url"`
	APIKey               string        `yaml:"api_key"`
}

// NewWebSocketPublisher creates new WebSocket publisher
func NewWebSocketPublisher(config Config, logger *logrus.Logger) *WebSocketPublisher {
	// Create WebSocket client config
	wsConfig := websocket.Config{
		URL:                  config.URL,
		ServerID:             config.ServerID,
		ServerKey:            config.ServerKey,
		ReconnectInterval:    config.ReconnectInterval,
		MaxReconnectAttempts: config.MaxReconnectAttempts,
		PingInterval:         config.PingInterval,
		WriteTimeout:         config.WriteTimeout,
		ReadTimeout:          config.ReadTimeout,
		HandshakeTimeout:     config.HandshakeTimeout,
		BufferSize:           config.BufferSize,
		EnableCompression:    config.EnableCompression,
		APIURL:               config.APIURL,
		APIKey:               config.APIKey,
	}

	publisher := &WebSocketPublisher{
		wsClient:    websocket.NewClient(wsConfig, logger),
		adapter:     types.NewWebSocketAdapter(),
		logger:      logger,
		buffer:      make([]*types.Metric, 0, config.MetricBufferSize),
		bufferSize:  config.MetricBufferSize,
		bufferFlush: config.MetricBufferFlush,
	}

	// Register command handlers
	publisher.registerCommandHandlers()

	return publisher
}

// Start starts the WebSocket publisher
func (p *WebSocketPublisher) Start(ctx context.Context) error {
	p.logger.Info("Starting WebSocket publisher")

	// Start WebSocket client
	p.wsClient.Start()

	// Start buffer flush goroutine
	go p.bufferFlusher(ctx)

	// Start metrics reporter
	go p.metricsReporter(ctx)

	return nil
}

// Publish publishes a single metric
func (p *WebSocketPublisher) Publish(ctx context.Context, metric *types.Metric) error {
	if !p.wsClient.IsConnected() {
		// Buffer metric for later
		p.bufferMetric(metric)
		return fmt.Errorf("WebSocket not connected, metric buffered")
	}

	// Convert to WebSocket message
	wsMsg := p.adapter.ToWebSocketMessage(metric)

	// Log raw JSON before sending for debugging
	if metric.Type == "metrics" {
		jsonBytes, err := json.Marshal(wsMsg)
		if err == nil {
			p.logger.WithField("raw_json", string(jsonBytes)).Debug("Raw WebSocket message JSON")
		} else {
			p.logger.WithError(err).Error("Failed to marshal WebSocket message to JSON")
		}
	}

	// Send message
	if err := p.wsClient.SendMessage(wsMsg); err != nil {
		p.incrementFailed()
		p.bufferMetric(metric)
		return fmt.Errorf("failed to send metric: %w", err)
	}

	p.incrementPublished()
	p.setLastPublished(time.Now())

	p.logger.WithFields(logrus.Fields{
		"type":        metric.Type,
		"server_id":   metric.ServerID,
		"server_name": metric.ServerName,
	}).Debug("Metric published via WebSocket")

	return nil
}

// PublishBatch publishes multiple metrics
func (p *WebSocketPublisher) PublishBatch(ctx context.Context, metrics []*types.Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	if !p.wsClient.IsConnected() {
		// Buffer all metrics for later
		for _, metric := range metrics {
			p.bufferMetric(metric)
		}
		return fmt.Errorf("WebSocket not connected, %d metrics buffered", len(metrics))
	}

	// Convert to WebSocket messages
	wsMessages := p.adapter.ToWebSocketMetrics(metrics)

	// Send messages
	sent := 0
	for _, msg := range wsMessages {
		if err := p.wsClient.SendMessage(msg); err != nil {
			p.incrementFailed()
			p.logger.WithError(err).Warn("Failed to send WebSocket message")
		} else {
			sent++
			p.incrementPublished()
		}
	}

	p.setLastPublished(time.Now())

	p.logger.WithFields(logrus.Fields{
		"total": len(metrics),
		"sent":  sent,
	}).Debug("Metrics batch published via WebSocket")

	if sent < len(metrics) {
		return fmt.Errorf("only %d of %d metrics sent", sent, len(metrics))
	}

	return nil
}

// Close closes the WebSocket publisher
func (p *WebSocketPublisher) Close() error {
	p.logger.Info("Closing WebSocket publisher")

	// Try to flush buffer before closing
	p.flushBuffer()

	return p.wsClient.Close()
}

// Name returns publisher name
func (p *WebSocketPublisher) Name() string {
	return "websocket"
}

// IsConnected returns connection status
func (p *WebSocketPublisher) IsConnected() bool {
	return p.wsClient.IsConnected()
}

// GetMetrics returns publisher statistics
func (p *WebSocketPublisher) GetMetrics() map[string]interface{} {
	p.metricsMu.RLock()
	defer p.metricsMu.RUnlock()

	return map[string]interface{}{
		"published_count": p.publishedCnt,
		"failed_count":    p.failedCnt,
		"last_published":  p.lastPublished,
		"buffer_size":     len(p.buffer),
		"is_connected":    p.IsConnected(),
		"server_id":       p.wsClient.ServerID(),
	}
}

// bufferMetric adds metric to buffer
func (p *WebSocketPublisher) bufferMetric(metric *types.Metric) {
	p.bufferMu.Lock()
	defer p.bufferMu.Unlock()

	// If buffer is full, remove oldest metric
	if len(p.buffer) >= p.bufferSize {
		p.buffer = p.buffer[1:]
		p.logger.Warn("Metric buffer full, dropping oldest metric")
	}

	p.buffer = append(p.buffer, metric)
}

// flushBuffer sends all buffered metrics
func (p *WebSocketPublisher) flushBuffer() {
	p.bufferMu.Lock()
	metrics := make([]*types.Metric, len(p.buffer))
	copy(metrics, p.buffer)
	p.buffer = p.buffer[:0] // Clear buffer
	p.bufferMu.Unlock()

	if len(metrics) > 0 && p.wsClient.IsConnected() {
		if err := p.PublishBatch(context.Background(), metrics); err != nil {
			p.logger.WithError(err).WithField("count", len(metrics)).Warn("Failed to flush buffered metrics")
		} else {
			p.logger.WithField("count", len(metrics)).Info("Flushed buffered metrics")
		}
	}
}

// bufferFlusher periodically flushes the buffer
func (p *WebSocketPublisher) bufferFlusher(ctx context.Context) {
	ticker := time.NewTicker(p.bufferFlush)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if p.wsClient.IsConnected() {
				p.flushBuffer()
			}
		}
	}
}

// metricsReporter periodically logs metrics
func (p *WebSocketPublisher) metricsReporter(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := p.GetMetrics()
			p.logger.WithFields(logrus.Fields{
				"published": metrics["published_count"],
				"failed":    metrics["failed_count"],
				"buffer":    metrics["buffer_size"],
				"connected": metrics["is_connected"],
			}).Info("WebSocket publisher metrics")
		}
	}
}

// registerCommandHandlers registers WebSocket command handlers
func (p *WebSocketPublisher) registerCommandHandlers() {
	// Register ping command handler
	p.wsClient.RegisterCommandHandler("ping", func(ctx context.Context, cmd *websocket.CommandMessage) (*websocket.CommandResponse, error) {
		return &websocket.CommandResponse{
			RequestID: cmd.RequestID,
			Success:   true,
			Data: map[string]interface{}{
				"message":   "pong",
				"timestamp": time.Now().Unix(),
			},
			Timestamp: time.Now().Unix(),
		}, nil
	})

	// Register status command handler
	p.wsClient.RegisterCommandHandler("status", func(ctx context.Context, cmd *websocket.CommandMessage) (*websocket.CommandResponse, error) {
		metrics := p.GetMetrics()
		return &websocket.CommandResponse{
			RequestID: cmd.RequestID,
			Success:   true,
			Data:      metrics,
			Timestamp: time.Now().Unix(),
		}, nil
	})

	// Register flush_buffer command handler
	p.wsClient.RegisterCommandHandler("flush_buffer", func(ctx context.Context, cmd *websocket.CommandMessage) (*websocket.CommandResponse, error) {
		go p.flushBuffer()
		return &websocket.CommandResponse{
			RequestID: cmd.RequestID,
			Success:   true,
			Data: map[string]interface{}{
				"message": "Buffer flush initiated",
			},
			Timestamp: time.Now().Unix(),
		}, nil
	})
}

// Helper methods for metrics
func (p *WebSocketPublisher) incrementPublished() {
	p.metricsMu.Lock()
	defer p.metricsMu.Unlock()
	p.publishedCnt++
}

func (p *WebSocketPublisher) incrementFailed() {
	p.metricsMu.Lock()
	defer p.metricsMu.Unlock()
	p.failedCnt++
}

func (p *WebSocketPublisher) setLastPublished(t time.Time) {
	p.metricsMu.Lock()
	defer p.metricsMu.Unlock()
	p.lastPublished = t
}
