package websocket

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Config represents WebSocket client configuration
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
	APIURL               string        `yaml:"api_url"`
	APIKey               string        `yaml:"api_key"`
}

// Client represents WebSocket client
type Client struct {
	config          Config
	conn            *websocket.Conn
	logger          *logrus.Logger
	send            chan Message
	receive         chan Message
	mu              sync.RWMutex
	writeMu         sync.Mutex // Separate mutex for write operations
	isConnected     bool
	isAuth          bool
	ctx             context.Context
	cancel          context.CancelFunc
	reconnectCh     chan struct{}
	done            chan struct{}
	commandHandlers map[string]CommandHandler
}

// CommandHandler handles incoming commands
type CommandHandler func(ctx context.Context, cmd *CommandMessage) (*CommandResponse, error)

// NewClient creates new WebSocket client
func NewClient(config Config, logger *logrus.Logger) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		config:          config,
		logger:          logger,
		send:            make(chan Message, config.BufferSize),
		receive:         make(chan Message, config.BufferSize),
		ctx:             ctx,
		cancel:          cancel,
		reconnectCh:     make(chan struct{}, 1),
		done:            make(chan struct{}),
		commandHandlers: make(map[string]CommandHandler),
	}
}

// getServerID obtains server_id from API using server_key
func (c *Client) getServerID() (string, error) {
	if c.config.APIURL == "" || c.config.APIKey == "" {
		return "", fmt.Errorf("API URL and API key required for server_id lookup")
	}

	// Query server info by server_key
	url := fmt.Sprintf("%s/api/servers/by-key/%s", c.config.APIURL, c.config.ServerKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query server info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	var response struct {
		ServerID string `json:"server_id"`
		Hostname string `json:"hostname"`
		Status   string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if response.ServerID == "" {
		return "", fmt.Errorf("server_id not found in response")
	}

	c.logger.WithFields(logrus.Fields{
		"server_id": response.ServerID,
		"hostname":  response.Hostname,
	}).Info("Server ID obtained from API")

	return response.ServerID, nil
}

// Connect establishes WebSocket connection
func (c *Client) Connect() error {
	// Use server_id from configuration
	// Note: server_id is obtained during initial registration and stored in config
	c.logger.WithFields(logrus.Fields{
		"url":       c.config.URL,
		"server_id": c.config.ServerID,
	}).Info("Connecting to WebSocket server")

	// Parse WebSocket URL
	u, err := url.Parse(c.config.URL)
	if err != nil {
		return fmt.Errorf("failed to parse WebSocket URL: %w", err)
	}

	// Set up dialer with TLS config
	dialer := websocket.Dialer{
		HandshakeTimeout: c.config.HandshakeTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	// Connect to WebSocket
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("WebSocket connection failed: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.isConnected = true
	c.mu.Unlock()

	c.logger.Info("WebSocket connection established")

	// Start read and write pumps
	go c.readPump()
	go c.writePump()

	// Wait a moment for connection to be fully established
	c.logger.Debug("Waiting for WebSocket connection to stabilize...")
	time.Sleep(100 * time.Millisecond)

	// Send authentication message
	if err := c.authenticate(); err != nil {
		c.Close()
		return fmt.Errorf("authentication failed: %w", err)
	}

	return nil
}

// authenticate sends authentication message
func (c *Client) authenticate() error {
	// Check connection state before sending auth
	c.mu.RLock()
	if !c.isConnected || c.conn == nil {
		c.mu.RUnlock()
		return fmt.Errorf("WebSocket not connected")
	}
	if c.isAuth {
		c.mu.RUnlock()
		return fmt.Errorf("already authenticated")
	}
	c.mu.RUnlock()

	authMsg := AuthMessage{
		Type:      MessageTypeAuth,
		ServerID:  c.config.ServerID,
		ServerKey: c.config.ServerKey,
	}

	c.logger.WithField("server_id", c.config.ServerID).Debug("Sending authentication message")

	if err := c.SendMessage(Message{
		Type:      authMsg.Type,
		ServerID:  authMsg.ServerID,
		ServerKey: authMsg.ServerKey,
	}); err != nil {
		return fmt.Errorf("failed to send auth message: %w", err)
	}

	// Wait for authentication response
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	select {
	case msg := <-c.receive:
		switch msg.Type {
		case MessageTypeAuthSuccess:
			c.mu.Lock()
			c.isAuth = true
			c.mu.Unlock()

			c.logger.Info("WebSocket authentication successful")
			return nil
		case MessageTypeError:
			return fmt.Errorf("authentication failed: %s", msg.Data["error"])
		default:
			// Ignore other messages during auth
		}
		return fmt.Errorf("unexpected auth response: %s", msg.Type)
	case <-ctx.Done():
		return fmt.Errorf("authentication timeout")
	}
}

// SendMessage sends message to WebSocket server
func (c *Client) SendMessage(msg Message) error {
	c.mu.RLock()
	if !c.isConnected || c.conn == nil {
		c.mu.RUnlock()
		return fmt.Errorf("WebSocket not connected")
	}
	// For auth messages, don't require authentication yet
	if msg.Type != MessageTypeAuth && !c.isAuth {
		c.mu.RUnlock()
		return fmt.Errorf("WebSocket not authenticated")
	}
	c.mu.RUnlock()

	select {
	case c.send <- msg:
		return nil
	case <-c.ctx.Done():
		return fmt.Errorf("client context cancelled")
	default:
		return fmt.Errorf("send buffer full")
	}
}

// ReceiveMessage receives message from WebSocket server
func (c *Client) ReceiveMessage() <-chan Message {
	return c.receive
}

// Start begins WebSocket client operation with reconnection
func (c *Client) Start() {
	go c.run()
}

// run manages WebSocket connection with reconnection logic
func (c *Client) run() {
	reconnectAttempts := 0

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("WebSocket client stopping")
			return
		case <-c.reconnectCh:
			// Reconnection triggered
		default:
			// Initial connection or reconnection
		}

		if err := c.Connect(); err != nil {
			reconnectAttempts++

			c.logger.WithFields(logrus.Fields{
				"error":   err.Error(),
				"attempt": reconnectAttempts,
				"max":     c.config.MaxReconnectAttempts,
			}).Error("WebSocket connection failed")

			if reconnectAttempts >= c.config.MaxReconnectAttempts {
				c.logger.Error("Max reconnection attempts reached, stopping")
				return
			}

			// Wait before reconnection
			select {
			case <-time.After(c.config.ReconnectInterval):
				continue
			case <-c.ctx.Done():
				return
			}
		}

		// Connection successful, reset attempts
		reconnectAttempts = 0

		// Wait for disconnection or context cancellation
		select {
		case <-c.done:
			c.logger.Info("WebSocket connection ended")
		case <-c.ctx.Done():
			c.logger.Info("WebSocket client context cancelled")
			return
		}

		// Schedule reconnection
		select {
		case c.reconnectCh <- struct{}{}:
		default:
		}
	}
}

// readPump reads messages from WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.signalDisconnection()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB max message size
	c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
		return nil
	})

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.WithError(err).Error("WebSocket unexpected close")
			} else {
				c.logger.WithError(err).Debug("WebSocket read error")
			}
			break
		}

		// Handle different message types
		switch msg.Type {
		case MessageTypeCommand:
			c.handleCommand(&msg)
		case MessageTypeHeartbeat:
			// Server heartbeat, ignore
		case MessageTypeError:
			c.logger.WithField("error", msg.Data).Error("WebSocket error message")
		default:
			// Forward to receive channel for other handlers
			select {
			case c.receive <- msg:
			case <-c.ctx.Done():
				return
			default:
				c.logger.Warn("Receive buffer full, dropping message")
			}
		}
	}
}

// writePump writes messages to WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.writeMu.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			if !ok {
				// Channel closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.writeMu.Unlock()
				return
			}

			if err := c.conn.WriteJSON(msg); err != nil {
				c.logger.WithError(err).Error("Failed to write WebSocket message")
				c.writeMu.Unlock()
				return
			}
			c.writeMu.Unlock()

		case <-ticker.C:
			c.writeMu.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.WithError(err).Error("Failed to send ping")
				c.writeMu.Unlock()
				return
			}
			c.writeMu.Unlock()

		case <-c.ctx.Done():
			return
		}
	}
}

// handleCommand processes incoming command messages
func (c *Client) handleCommand(msg *Message) {
	cmdMsg := &CommandMessage{
		Type:      msg.Type,
		RequestID: fmt.Sprintf("%v", msg.Data["request_id"]),
		Data:      msg.Data,
		Timestamp: msg.Timestamp,
	}

	// Handle command asynchronously
	go func() {
		_, cancel := context.WithTimeout(c.ctx, 30*time.Second)
		defer cancel()

		// Default command handler
		response := &CommandResponse{
			RequestID: cmdMsg.RequestID,
			Success:   false,
			Error:     "Command handler not implemented",
			Timestamp: time.Now().Unix(),
		}

		// Send response
		if err := c.SendMessage(Message{
			Type: MessageTypeCommand,
			Data: map[string]interface{}{
				"request_id": response.RequestID,
				"success":    response.Success,
				"data":       response.Data,
				"error":      response.Error,
				"timestamp":  response.Timestamp,
			},
		}); err != nil {
			c.logger.WithError(err).Error("Failed to send command response")
		}
	}()
}

// signalDisconnection signals that connection is lost
func (c *Client) signalDisconnection() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already disconnected
	if !c.isConnected {
		return
	}

	c.isConnected = false
	c.isAuth = false

	// Signal disconnection safely
	select {
	case c.done <- struct{}{}:
	default:
		// Channel might be full or closed, ignore
	}
}

// IsConnected returns connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected && c.isAuth
}

// ServerID returns configured server ID
func (c *Client) ServerID() string {
	return c.config.ServerID
}

// Close closes WebSocket connection
func (c *Client) Close() error {
	c.cancel()

	c.mu.Lock()
	// Check if already closed
	if c.conn == nil && !c.isConnected {
		c.mu.Unlock()
		return nil
	}

	// Mark as closing to prevent new operations
	c.isConnected = false
	c.isAuth = false

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()

	// Close channels safely - only if not already closed
	defer func() {
		if r := recover(); r != nil {
			c.logger.Warn("Recovered from panic during channel close")
		}
	}()

	select {
	case <-c.done:
		// Already closed
	default:
		close(c.done)
	}

	c.logger.Info("WebSocket client closed")
	return nil
}

// RegisterCommandHandler registers a command handler
func (c *Client) RegisterCommandHandler(command string, handler CommandHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.commandHandlers[command] = handler
}
