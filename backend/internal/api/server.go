package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/servereye/servereye/backend/storage"
	"github.com/servereye/servereye/pkg/kafka"
	"github.com/servereye/servereye/pkg/publisher"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
)

type Server struct {
	config          *Config
	logger          *logrus.Logger
	kafkaProducer   *kafka.Producer
	httpServer      *http.Server
	wsServer        *WebSocketServer
	pendingCommands map[string][]PendingCommand
	pendingMutex    sync.RWMutex
	responseChans   map[string]chan CommandResponse
	responseMutex   sync.RWMutex
	storage         storage.Storage
}

type Config struct {
	Server struct {
		Host string
		Port string
	}
	Kafka struct {
		Brokers     []string
		TopicPrefix string
	}
	Auth struct {
		APIKey string
	}
}

type MetricRequest struct {
	ServerID  string            `json:"server_id"`
	ServerKey string            `json:"server_key"`
	Type      string            `json:"type"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CommandRequest struct {
	ServerKey string                 `json:"server_key"`
	Command   string                 `json:"command"`
	Params    map[string]interface{} `json:"params,omitempty"`
	RequestID string                 `json:"request_id"`
}

type CommandResponse struct {
	RequestID string      `json:"request_id"`
	ServerKey string      `json:"server_key"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
}

type PendingCommand struct {
	ID        string                 `json:"id"`
	Command   string                 `json:"command"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type KeyRegistrationRequest struct {
	SecretKey    string `json:"secret_key"`
	AgentVersion string `json:"agent_version"`
	OSInfo       string `json:"os_info"`
	Hostname     string `json:"hostname"`
}

func New(cfg *Config, logger *logrus.Logger, storage storage.Storage) (*Server, error) {
	// Initialize Kafka producer
	kafkaConfig := kafka.Config{
		Brokers:      cfg.Kafka.Brokers,
		TopicPrefix:  cfg.Kafka.TopicPrefix,
		Compression:  "snappy",
		MaxAttempts:  3,
		BatchSize:    100,
		BatchTimeout: 1 * time.Second,
		RequiredAcks: 1,
		WriteTimeout: 10 * time.Second,
	}

	kafkaProducer, err := kafka.NewProducer(kafkaConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	server := &Server{
		config:          cfg,
		logger:          logger,
		kafkaProducer:   kafkaProducer,
		pendingCommands: make(map[string][]PendingCommand),
		responseChans:   make(map[string]chan CommandResponse),
		storage:         storage,
	}

	// Initialize WebSocket server
	server.wsServer = NewWebSocketServer(server, logger)

	return server, nil
}

func (s *Server) Start() error {
	router := mux.NewRouter()

	// Public endpoints (no auth required)
	router.HandleFunc("/health", s.handleHealth).Methods("GET")
	router.HandleFunc("/register-key", s.handleRegisterKey).Methods("POST")
	router.HandleFunc("/ws", s.wsServer.handleWebSocket)

	// Protected endpoints
	protected := router.PathPrefix("/v1").Subrouter()
	protected.HandleFunc("/metrics", s.handleMetrics).Methods("POST")
	protected.HandleFunc("/commands", s.handleSendCommand).Methods("POST")
	protected.HandleFunc("/commands/{server_key}", s.handleGetCommands).Methods("GET")
	protected.HandleFunc("/commands/{server_key}/response", s.handleCommandResponse).Methods("POST")
	protected.Use(s.authMiddleware)

	// Global middleware
	router.Use(s.loggingMiddleware)

	addr := fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Enable HTTP/2 support
	if err := http2.ConfigureServer(s.httpServer, &http2.Server{}); err != nil {
		s.logger.WithError(err).Error("Failed to configure HTTP/2")
	}

	s.logger.WithField("addr", addr).Info("Starting HTTP server with HTTP/2 support")
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	var req MetricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate required fields
	if req.ServerID == "" || req.Type == "" {
		s.writeError(w, http.StatusBadRequest, "Missing required fields", "server_id and type are required")
		return
	}

	// Create publisher metric
	metric := &publisher.Metric{
		ServerID:  req.ServerID,
		ServerKey: req.ServerKey,
		Type:      req.Type,
		Value:     req.Value,
		Timestamp: req.Timestamp,
		Tags:      req.Tags,
		Version:   "1.0",
	}

	// Publish to Kafka
	if err := s.kafkaProducer.Publish(r.Context(), metric); err != nil {
		s.logger.WithError(err).WithField("type", req.Type).Error("Failed to publish metric to Kafka")
		s.writeError(w, http.StatusInternalServerError, "Internal server error", "Failed to publish metric")
		return
	}

	s.logger.WithFields(logrus.Fields{
		"server_id": req.ServerID,
		"type":      req.Type,
		"value":     req.Value,
	}).Info("Metric published to Kafka")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/health/kafka" {
			next.ServeHTTP(w, r)
			return
		}

		apiKey := r.Header.Get("X-API-Key")
		if apiKey != s.config.Auth.APIKey {
			s.writeError(w, http.StatusUnauthorized, "Unauthorized", "Invalid API key")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)

		s.logger.WithFields(logrus.Fields{
			"method":   r.Method,
			"path":     r.URL.Path,
			"duration": time.Since(start).Milliseconds(),
		}).Info("Request processed")
	})
}

func (s *Server) writeError(w http.ResponseWriter, status int, errorType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errorType,
		Code:    status,
		Message: message,
	})
}

func (s *Server) handleSendCommand(w http.ResponseWriter, r *http.Request) {
	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	if req.ServerKey == "" || req.Command == "" {
		s.writeError(w, http.StatusBadRequest, "Missing required fields", "server_key and command are required")
		return
	}

	if req.RequestID == "" {
		req.RequestID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	cmd := PendingCommand{
		ID:        req.RequestID,
		Command:   req.Command,
		Params:    req.Params,
		Timestamp: time.Now(),
	}

	s.pendingMutex.Lock()
	s.pendingCommands[req.ServerKey] = append(s.pendingCommands[req.ServerKey], cmd)
	s.pendingMutex.Unlock()

	responseChan := make(chan CommandResponse, 1)
	s.responseMutex.Lock()
	s.responseChans[req.RequestID] = responseChan
	s.responseMutex.Unlock()

	s.logger.WithFields(logrus.Fields{
		"server_key": req.ServerKey,
		"command":    req.Command,
		"request_id": req.RequestID,
	}).Info("Command queued for agent")

	select {
	case resp := <-responseChan:
		s.responseMutex.Lock()
		delete(s.responseChans, req.RequestID)
		s.responseMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if resp.Success {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(resp)

	case <-time.After(30 * time.Second):
		s.responseMutex.Lock()
		delete(s.responseChans, req.RequestID)
		s.responseMutex.Unlock()

		s.pendingMutex.Lock()
		cmds := s.pendingCommands[req.ServerKey]
		for i, c := range cmds {
			if c.ID == req.RequestID {
				s.pendingCommands[req.ServerKey] = append(cmds[:i], cmds[i+1:]...)
				break
			}
		}
		s.pendingMutex.Unlock()

		s.writeError(w, http.StatusGatewayTimeout, "Timeout", "Agent did not respond in time")
	}
}

func (s *Server) handleGetCommands(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverKey := vars["server_key"]

	if serverKey == "" {
		s.writeError(w, http.StatusBadRequest, "Missing server_key", "server_key is required")
		return
	}

	s.pendingMutex.Lock()
	commands := s.pendingCommands[serverKey]
	s.pendingCommands[serverKey] = nil
	s.pendingMutex.Unlock()

	if commands == nil {
		commands = []PendingCommand{}
	}

	s.logger.WithFields(logrus.Fields{
		"server_key": serverKey,
		"count":      len(commands),
	}).Debug("Commands fetched by agent")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"commands": commands,
	})
}

func (s *Server) handleCommandResponse(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverKey := vars["server_key"]

	var resp CommandResponse
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	resp.ServerKey = serverKey

	s.responseMutex.RLock()
	responseChan, exists := s.responseChans[resp.RequestID]
	s.responseMutex.RUnlock()

	if exists {
		select {
		case responseChan <- resp:
			s.logger.WithFields(logrus.Fields{
				"server_key": serverKey,
				"request_id": resp.RequestID,
				"success":    resp.Success,
			}).Info("Command response received")
		default:
			s.logger.Warn("Response channel full, dropping response")
		}
	} else {
		s.logger.WithField("request_id", resp.RequestID).Warn("No waiting request for response")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

// WebSocket structures
type WebSocketServer struct {
	server   *Server
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
	clientsM sync.RWMutex
	logger   *logrus.Logger
}

type WSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(server *Server, logger *logrus.Logger) *WebSocketServer {
	return &WebSocketServer{
		server:  server,
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		logger: logger,
	}
}

// GetWebSocketServer returns the WebSocket server instance
func (s *Server) GetWebSocketServer() *WebSocketServer {
	return s.wsServer
}

// handleWebSocket handles WebSocket connections
func (ws *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		ws.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}
	defer conn.Close()

	// Add client
	ws.clientsM.Lock()
	ws.clients[conn] = true
	ws.clientsM.Unlock()

	defer func() {
		ws.clientsM.Lock()
		delete(ws.clients, conn)
		ws.clientsM.Unlock()
	}()

	ws.logger.Info("WebSocket client connected")

	// Send welcome message
	welcome := WSMessage{
		Type:      "welcome",
		Data:      map[string]string{"status": "connected"},
		Timestamp: time.Now(),
	}
	if err := conn.WriteJSON(welcome); err != nil {
		ws.logger.WithError(err).Error("Failed to send welcome message")
		return
	}

	// Keep connection alive and handle messages
	for {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				ws.logger.WithError(err).Error("WebSocket error")
			}
			break
		}

		// Handle subscription messages
		if msg.Type == "subscribe" {
			ws.handleSubscription(conn, msg)
		}
	}
}

// handleSubscription handles subscription messages from clients
func (ws *WebSocketServer) handleSubscription(conn *websocket.Conn, msg WSMessage) {
	// For now, just log subscription - can be extended later
	ws.logger.WithField("data", msg.Data).Info("Client subscription received")
}

// BroadcastMetric sends metric to all connected WebSocket clients
func (ws *WebSocketServer) BroadcastMetric(metric interface{}) {
	ws.clientsM.RLock()
	defer ws.clientsM.RUnlock()

	message := WSMessage{
		Type:      "metric",
		Data:      metric,
		Timestamp: time.Now(),
	}

	for conn := range ws.clients {
		if err := conn.WriteJSON(message); err != nil {
			ws.logger.WithError(err).Error("Failed to broadcast metric to WebSocket client")
			// Remove client on error
			delete(ws.clients, conn)
			conn.Close()
		}
	}
}

func (s *Server) handleRegisterKey(w http.ResponseWriter, r *http.Request) {
	var req KeyRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate required fields
	if req.SecretKey == "" {
		s.writeError(w, http.StatusBadRequest, "Missing required fields", "secret_key is required")
		return
	}
	if req.Hostname == "" {
		req.Hostname = "unknown"
	}
	if req.AgentVersion == "" {
		req.AgentVersion = "unknown"
	}
	if req.OSInfo == "" {
		req.OSInfo = "unknown"
	}

	// Insert into database
	if err := s.storage.InsertGeneratedKey(r.Context(), req.SecretKey, req.AgentVersion, req.OSInfo, req.Hostname); err != nil {
		s.logger.WithError(err).WithField("secret_key", req.SecretKey).Error("Failed to insert generated key")
		s.writeError(w, http.StatusInternalServerError, "Internal server error", "Failed to register key")
		return
	}

	s.logger.WithField("secret_key", req.SecretKey).Info("Key registered successfully")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Key registered successfully"})
}
