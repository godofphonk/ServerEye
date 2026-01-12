package commands

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/godofphonk/ServerEye/pkg/protocol"
	"github.com/godofphonk/ServerEye/pkg/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketCommandConsumer consumes commands via WebSocket
type WebSocketCommandConsumer struct {
	wsClient *websocket.Client
	handler  CommandHandlerInterface
	logger   *logrus.Logger

	// Command processing
	commandQueue chan *websocket.CommandMessage
	processingMu sync.RWMutex
	isProcessing bool

	// Metrics
	metricsMu     sync.RWMutex
	processedCnt  int64
	failedCnt     int64
	lastProcessed time.Time
}

// CommandHandlerInterface handles incoming commands
type CommandHandlerInterface interface {
	HandleCommand(ctx context.Context, cmd *protocol.Message) (*protocol.Message, error)
}

// Config represents WebSocket consumer configuration
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
	CommandQueueSize     int           `yaml:"command_queue_size" default:"100"`
	CommandTimeout       time.Duration `yaml:"command_timeout" default:"30s"`
	APIURL               string        `yaml:"api_url"`
	APIKey               string        `yaml:"api_key"`
}

// NewWebSocketCommandConsumer creates new WebSocket command consumer
func NewWebSocketCommandConsumer(config Config, handler CommandHandlerInterface, logger *logrus.Logger) *WebSocketCommandConsumer {
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

	consumer := &WebSocketCommandConsumer{
		wsClient:     websocket.NewClient(wsConfig, logger),
		handler:      handler,
		logger:       logger,
		commandQueue: make(chan *websocket.CommandMessage, config.CommandQueueSize),
	}

	// Register command handlers
	consumer.registerCommandHandlers()

	return consumer
}

// Start starts the WebSocket command consumer
func (c *WebSocketCommandConsumer) Start(ctx context.Context) error {
	c.logger.Info("Starting WebSocket command consumer")

	// Start WebSocket client
	c.wsClient.Start()

	// Start command processor
	go c.commandProcessor(ctx)

	// Start message listener
	go c.messageListener(ctx)

	// Start metrics reporter
	go c.metricsReporter(ctx)

	return nil
}

// Stop stops the WebSocket command consumer
func (c *WebSocketCommandConsumer) Stop() error {
	c.logger.Info("Stopping WebSocket command consumer")

	c.processingMu.Lock()
	c.isProcessing = false
	c.processingMu.Unlock()

	close(c.commandQueue)
	return c.wsClient.Close()
}

// messageListener listens for incoming WebSocket messages
func (c *WebSocketCommandConsumer) messageListener(ctx context.Context) {
	messageCh := c.wsClient.ReceiveMessage()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messageCh:
			if !ok {
				c.logger.Info("WebSocket message channel closed")
				return
			}

			if msg.Type == websocket.MessageTypeCommand {
				cmdMsg := &websocket.CommandMessage{
					Type:      msg.Type,
					RequestID: fmt.Sprintf("%v", msg.Data["request_id"]),
					Data:      msg.Data,
					Timestamp: msg.Timestamp,
				}

				select {
				case c.commandQueue <- cmdMsg:
					c.logger.WithField("request_id", cmdMsg.RequestID).Debug("Command queued")
				default:
					c.logger.Warn("Command queue full, dropping command")
					c.incrementFailed()
				}
			}
		}
	}
}

// commandProcessor processes queued commands
func (c *WebSocketCommandConsumer) commandProcessor(ctx context.Context) {
	c.processingMu.Lock()
	c.isProcessing = true
	c.processingMu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return
		case cmdMsg, ok := <-c.commandQueue:
			if !ok {
				c.logger.Info("Command queue closed")
				return
			}

			c.processCommand(ctx, cmdMsg)
		}
	}
}

// processCommand processes a single command
func (c *WebSocketCommandConsumer) processCommand(ctx context.Context, cmdMsg *websocket.CommandMessage) {
	start := time.Now()

	c.logger.WithFields(logrus.Fields{
		"request_id": cmdMsg.RequestID,
		"type":       cmdMsg.Type,
	}).Debug("Processing command")

	// Convert WebSocket command to protocol message
	protoMsg, err := c.websocketToProtocolMessage(cmdMsg)
	if err != nil {
		c.logger.WithError(err).Error("Failed to convert WebSocket command to protocol message")
		c.sendErrorResponse(cmdMsg.RequestID, err)
		c.incrementFailed()
		return
	}

	// Handle command with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := c.handler.HandleCommand(timeoutCtx, protoMsg)
	if err != nil {
		c.logger.WithError(err).Error("Command handler failed")
		c.sendErrorResponse(cmdMsg.RequestID, err)
		c.incrementFailed()
		return
	}

	// Send success response
	if err := c.sendSuccessResponse(cmdMsg.RequestID, response); err != nil {
		c.logger.WithError(err).Error("Failed to send command response")
		c.incrementFailed()
		return
	}

	c.incrementProcessed()
	c.setLastProcessed(time.Now())

	c.logger.WithFields(logrus.Fields{
		"request_id": cmdMsg.RequestID,
		"duration":   time.Since(start),
	}).Debug("Command processed successfully")
}

// websocketToProtocolMessage converts WebSocket command to protocol message
func (c *WebSocketCommandConsumer) websocketToProtocolMessage(cmdMsg *websocket.CommandMessage) (*protocol.Message, error) {
	// Extract command type and parameters
	commandTypeStr, ok := cmdMsg.Data["command"].(string)
	if !ok {
		return nil, fmt.Errorf("missing command type")
	}

	// Convert string to MessageType
	commandType := protocol.MessageType(commandTypeStr)

	// Extract parameters
	params, ok := cmdMsg.Data["params"].(map[string]interface{})
	if !ok {
		params = make(map[string]interface{})
	}

	// Create protocol message
	return &protocol.Message{
		ID:        cmdMsg.RequestID,
		Type:      commandType,
		Timestamp: time.Unix(cmdMsg.Timestamp, 0),
		Version:   "1.0",
		Payload:   params,
	}, nil
}

// sendSuccessResponse sends success response to WebSocket server
func (c *WebSocketCommandConsumer) sendSuccessResponse(requestID string, response *protocol.Message) error {
	responseData := map[string]interface{}{
		"request_id": requestID,
		"success":    true,
		"data":       response.Payload,
		"timestamp":  time.Now().Unix(),
	}

	if response.Type != "" {
		responseData["type"] = response.Type
	}

	return c.wsClient.SendMessage(websocket.Message{
		Type: websocket.MessageTypeCommand,
		Data: responseData,
	})
}

// sendErrorResponse sends error response to WebSocket server
func (c *WebSocketCommandConsumer) sendErrorResponse(requestID string, err error) {
	responseData := map[string]interface{}{
		"request_id": requestID,
		"success":    false,
		"error":      err.Error(),
		"timestamp":  time.Now().Unix(),
	}

	if sendErr := c.wsClient.SendMessage(websocket.Message{
		Type: websocket.MessageTypeCommand,
		Data: responseData,
	}); sendErr != nil {
		c.logger.WithError(sendErr).Error("Failed to send error response")
	}
}

// registerCommandHandlers registers WebSocket command handlers
func (c *WebSocketCommandConsumer) registerCommandHandlers() {
	// Register echo command handler
	c.wsClient.RegisterCommandHandler("echo", func(ctx context.Context, cmd *websocket.CommandMessage) (*websocket.CommandResponse, error) {
		return &websocket.CommandResponse{
			RequestID: cmd.RequestID,
			Success:   true,
			Data: map[string]interface{}{
				"echo": cmd.Data,
			},
			Timestamp: time.Now().Unix(),
		}, nil
	})

	// Register status command handler
	c.wsClient.RegisterCommandHandler("consumer_status", func(ctx context.Context, cmd *websocket.CommandMessage) (*websocket.CommandResponse, error) {
		metrics := c.GetMetrics()
		return &websocket.CommandResponse{
			RequestID: cmd.RequestID,
			Success:   true,
			Data:      metrics,
			Timestamp: time.Now().Unix(),
		}, nil
	})

	// Register restart command handler
	c.wsClient.RegisterCommandHandler("restart", func(ctx context.Context, cmd *websocket.CommandMessage) (*websocket.CommandResponse, error) {
		c.logger.Info("Received restart command")

		// Handle restart asynchronously to avoid blocking
		go func() {
			time.Sleep(1 * time.Second) // Give time for response to be sent
			c.logger.Info("Executing restart...")
			// In a real implementation, this would restart the agent
			// For now, just log it
		}()

		return &websocket.CommandResponse{
			RequestID: cmd.RequestID,
			Success:   true,
			Data: map[string]interface{}{
				"message": "Restart initiated",
			},
			Timestamp: time.Now().Unix(),
		}, nil
	})

	// Register custom command handler that forwards to the main handler
	c.wsClient.RegisterCommandHandler("custom", func(ctx context.Context, cmd *websocket.CommandMessage) (*websocket.CommandResponse, error) {
		// Convert to protocol message and handle
		protoMsg, err := c.websocketToProtocolMessage(cmd)
		if err != nil {
			return nil, err
		}

		response, err := c.handler.HandleCommand(ctx, protoMsg)
		if err != nil {
			return nil, err
		}

		return &websocket.CommandResponse{
			RequestID: cmd.RequestID,
			Success:   true,
			Data:      response.Payload,
			Timestamp: time.Now().Unix(),
		}, nil
	})
}

// GetMetrics returns consumer statistics
func (c *WebSocketCommandConsumer) GetMetrics() map[string]interface{} {
	c.metricsMu.RLock()
	defer c.metricsMu.RUnlock()

	c.processingMu.RLock()
	isProcessing := c.isProcessing
	c.processingMu.RUnlock()

	return map[string]interface{}{
		"processed_count": c.processedCnt,
		"failed_count":    c.failedCnt,
		"last_processed":  c.lastProcessed,
		"queue_size":      len(c.commandQueue),
		"is_processing":   isProcessing,
		"is_connected":    c.wsClient.IsConnected(),
		"server_id":       c.wsClient.ServerID(),
	}
}

// metricsReporter periodically logs metrics
func (c *WebSocketCommandConsumer) metricsReporter(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := c.GetMetrics()
			c.logger.WithFields(logrus.Fields{
				"processed": metrics["processed_count"],
				"failed":    metrics["failed_count"],
				"queue":     metrics["queue_size"],
				"connected": metrics["is_connected"],
			}).Info("WebSocket command consumer metrics")
		}
	}
}

// Helper methods for metrics
func (c *WebSocketCommandConsumer) incrementProcessed() {
	c.metricsMu.Lock()
	defer c.metricsMu.Unlock()
	c.processedCnt++
}

func (c *WebSocketCommandConsumer) incrementFailed() {
	c.metricsMu.Lock()
	defer c.metricsMu.Unlock()
	c.failedCnt++
}

func (c *WebSocketCommandConsumer) setLastProcessed(t time.Time) {
	c.metricsMu.Lock()
	defer c.metricsMu.Unlock()
	c.lastProcessed = t
}
