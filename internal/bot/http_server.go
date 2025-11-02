package bot

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	pkgredis "github.com/servereye/servereye/pkg/redis"
)

// KeyRegistrationRequest represents a request to register a generated key
type KeyRegistrationRequest struct {
	SecretKey    string `json:"secret_key"`
	AgentVersion string `json:"agent_version,omitempty"`
	OSInfo       string `json:"os_info,omitempty"`
	Hostname     string `json:"hostname,omitempty"`
}

// startHTTPServer starts HTTP server for agent API
func (b *Bot) startHTTPServer() {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("Operation failed", nil)
		}
	}()

	b.logger.Info("Info message")
	http.HandleFunc("/api/register-key", b.handleRegisterKey)
	http.HandleFunc("/api/health", b.handleHealth)
	http.HandleFunc("/api/heartbeat", b.handleHeartbeat)
	http.HandleFunc("/api/redis/publish", b.handleRedisPublish)
	http.HandleFunc("/api/redis/subscribe", b.handleRedisSubscribe)

	// Redis Streams endpoints (new)
	http.HandleFunc("/api/streams/xadd", b.handleStreamAdd)
	http.HandleFunc("/api/streams/xread", b.handleStreamRead)
	http.HandleFunc("/api/streams/xreadgroup", b.handleStreamReadGroup)
	http.HandleFunc("/api/streams/xack", b.handleStreamAck)

	http.HandleFunc("/api/monitoring/memory", b.handleMemoryRequest)
	http.HandleFunc("/api/monitoring/disk", b.handleDiskRequest)
	http.HandleFunc("/api/monitoring/uptime", b.handleUptimeRequest)
	http.HandleFunc("/api/monitoring/processes", b.handleProcessesRequest)

	b.logger.Info("Info message")

	// Create HTTP server with proper timeouts for security
	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		b.logger.Error("Error occurred", err)
	}
}

// handleRegisterKey handles key registration from agent
func (b *Bot) handleRegisterKey(w http.ResponseWriter, r *http.Request) {
	b.logger.Info("HTTP request received")

	if r.Method != http.MethodPost {
		b.logger.Error("Error message", nil)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req KeyRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		b.logger.Error("Error occurred", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	b.logger.Info("Operation completed")

	// Validate secret key
	if !strings.HasPrefix(req.SecretKey, "srv_") {
		http.Error(w, "Invalid secret key format", http.StatusBadRequest)
		return
	}

	// Record the key
	if err := b.recordGeneratedKey(req.SecretKey); err != nil {
		b.logger.Error("Error occurred", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If agent info provided, update connection info
	if req.AgentVersion != "" || req.OSInfo != "" || req.Hostname != "" {
		if err := b.updateKeyConnection(req.SecretKey, req.AgentVersion, req.OSInfo, req.Hostname); err != nil {
			b.logger.Error("Error occurred", err)
		}
	}

	b.logger.Info("Operation completed")

	response := map[string]interface{}{
		"success": true,
		"message": "Key registered successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealth handles health check requests
func (b *Bot) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "servereye-bot",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HeartbeatRequest represents an agent heartbeat
type HeartbeatRequest struct {
	ServerKey string `json:"server_key"`
}

// handleHeartbeat handles heartbeat requests from agents
func (b *Bot) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Simply acknowledge the heartbeat
	response := map[string]interface{}{
		"status":     "ok",
		"server_key": req.ServerKey,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RedisPublishRequest represents a request to publish to Redis
type RedisPublishRequest struct {
	Channel string `json:"channel"`
	Message string `json:"message"`
}

// RedisSubscribeRequest represents a request to subscribe to Redis
type RedisSubscribeRequest struct {
	Channel string `json:"channel"`
	Timeout int    `json:"timeout,omitempty"` // seconds
}

// handleRedisPublish handles Redis publish requests from agents
func (b *Bot) handleRedisPublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RedisPublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		b.logger.Error("Failed to decode JSON in publish request", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	b.logger.Info("Received publish request")

	// Validate channel format (should be resp:srv_* or heartbeat:srv_*)
	if !strings.HasPrefix(req.Channel, "resp:srv_") && !strings.HasPrefix(req.Channel, "heartbeat:srv_") {
		http.Error(w, "Invalid channel format", http.StatusBadRequest)
		return
	}

	// Publish to Redis
	b.logger.Info("Publishing to Redis")
	if err := b.redisClient.Publish(b.ctx, req.Channel, []byte(req.Message)); err != nil {
		b.logger.Error("Redis publish failed", err)
		http.Error(w, "Redis publish failed", http.StatusInternalServerError)
		return
	}
	b.logger.Info("Redis publish successful")

	response := map[string]interface{}{
		"success": true,
		"message": "Published successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRedisSubscribe handles Redis subscribe requests from agents
func (b *Bot) handleRedisSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RedisSubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate channel format (should be cmd:srv_*)
	if !strings.HasPrefix(req.Channel, "cmd:srv_") {
		http.Error(w, "Invalid channel format", http.StatusBadRequest)
		return
	}

	// Set default timeout - very short to not miss commands
	if req.Timeout == 0 {
		req.Timeout = 1
	}
	// No delay - need immediate response

	// Subscribe to Redis channel
	subscription, err := b.redisClient.Subscribe(b.ctx, req.Channel)
	if err != nil {
		b.logger.Error("Error occurred", err)
		http.Error(w, "Redis subscribe failed", http.StatusInternalServerError)
		return
	}
	defer func() {
		if subscription != nil {
			subscription.Close()
		}
	}()

	// Set response headers for streaming
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Wait for message or timeout
	timeout := time.After(time.Duration(req.Timeout) * time.Second)

	select {
	case message := <-subscription.Channel():
		if message != nil {
			response := map[string]interface{}{
				"success": true,
				"message": string(message),
				"channel": req.Channel,
			}
			json.NewEncoder(w).Encode(response)
		} else {
			response := map[string]interface{}{
				"success": false,
				"message": "Channel closed",
			}
			json.NewEncoder(w).Encode(response)
		}
	case <-timeout:
		response := map[string]interface{}{
			"success": false,
			"message": "Timeout waiting for message",
		}
		json.NewEncoder(w).Encode(response)
	case <-b.ctx.Done():
		response := map[string]interface{}{
			"success": false,
			"message": "Server shutting down",
		}
		json.NewEncoder(w).Encode(response)
	}
}

// handleMemoryRequest handles direct memory requests from agents
func (b *Bot) handleMemoryRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get server key from request
	var req struct {
		ServerKey string `json:"server_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Get memory info directly
	memInfo, err := b.getMemoryInfo(req.ServerKey)
	if err != nil {
		b.logger.Error("Failed to get memory info", err)
		http.Error(w, "Failed to get memory info", http.StatusInternalServerError)
		return
	}

	// Return memory info as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    memInfo,
	})
}

// Placeholder handlers for other monitoring endpoints
func (b *Bot) handleDiskRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (b *Bot) handleUptimeRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (b *Bot) handleProcessesRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

// getRawRedisClient returns the underlying redis.Client for Streams operations
func (b *Bot) getRawRedisClient() (*redis.Client, bool) {
	if rawClient, ok := b.redisRawClient.(*pkgredis.Client); ok {
		return rawClient.GetRawClient(), true
	}
	return nil, false
}

// handleStreamAdd handles XADD requests (add message to stream)
func (b *Bot) handleStreamAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Stream string            `json:"stream"`
		Values map[string]string `json:"values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	rdb, ok := b.getRawRedisClient()
	if !ok {
		http.Error(w, "Redis client not available", http.StatusInternalServerError)
		return
	}

	id, err := rdb.XAdd(r.Context(), &redis.XAddArgs{
		Stream: req.Stream,
		Values: req.Values,
	}).Result()

	if err != nil {
		b.logger.Error("XADD failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

// handleStreamRead handles XREAD requests (read messages)
func (b *Bot) handleStreamRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Stream string `json:"stream"`
		LastID string `json:"last_id"`
		Count  int64  `json:"count"`
		Block  int64  `json:"block_ms"` // milliseconds
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.LastID == "" {
		req.LastID = "0"
	}
	if req.Count <= 0 {
		req.Count = 10
	}

	rdb, ok := b.getRawRedisClient()
	if !ok {
		http.Error(w, "Redis client not available", http.StatusInternalServerError)
		return
	}

	streams, err := rdb.XRead(r.Context(), &redis.XReadArgs{
		Streams: []string{req.Stream, req.LastID},
		Count:   req.Count,
		Block:   time.Duration(req.Block) * time.Millisecond,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			// No messages
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"messages": []interface{}{}})
			return
		}
		b.logger.Error("XREAD failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"streams": streams})
}

// handleStreamReadGroup handles XREADGROUP requests (consumer group read)
func (b *Bot) handleStreamReadGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Stream   string `json:"stream"`
		Group    string `json:"group"`
		Consumer string `json:"consumer"`
		Count    int64  `json:"count"`
		Block    int64  `json:"block_ms"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Count <= 0 {
		req.Count = 10
	}

	rdb, ok := b.getRawRedisClient()
	if !ok {
		http.Error(w, "Redis client not available", http.StatusInternalServerError)
		return
	}

	streams, err := rdb.XReadGroup(r.Context(), &redis.XReadGroupArgs{
		Group:    req.Group,
		Consumer: req.Consumer,
		Streams:  []string{req.Stream, ">"},
		Count:    req.Count,
		Block:    time.Duration(req.Block) * time.Millisecond,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"messages": []interface{}{}})
			return
		}
		b.logger.Error("XREADGROUP failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"streams": streams})
}

// handleStreamAck handles XACK requests (acknowledge message)
func (b *Bot) handleStreamAck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Stream string `json:"stream"`
		Group  string `json:"group"`
		ID     string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	rdb, ok := b.getRawRedisClient()
	if !ok {
		http.Error(w, "Redis client not available", http.StatusInternalServerError)
		return
	}

	err := rdb.XAck(r.Context(), req.Stream, req.Group, req.ID).Err()
	if err != nil {
		b.logger.Error("XACK failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
