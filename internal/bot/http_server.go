package bot

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// KeyRegistrationRequest represents a request to register a generated key
type KeyRegistrationRequest struct {
	SecretKey     string `json:"secret_key"`
	AgentVersion  string `json:"agent_version,omitempty"`
	OSInfo        string `json:"os_info,omitempty"`
	Hostname      string `json:"hostname,omitempty"`
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
	http.HandleFunc("/api/redis/publish", b.handleRedisPublish)
	http.HandleFunc("/api/redis/subscribe", b.handleRedisSubscribe)
	
	b.logger.Info("Info message")
	if err := http.ListenAndServe(":8080", nil); err != nil {
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
		"status": "healthy",
		"service": "servereye-bot",
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
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate channel format (should be resp:srv_*)
	if !strings.HasPrefix(req.Channel, "resp:srv_") {
		http.Error(w, "Invalid channel format", http.StatusBadRequest)
		return
	}

	// Publish to Redis
	if err := b.redisClient.Publish(b.ctx, req.Channel, []byte(req.Message)); err != nil {
		b.logger.Error("Error occurred", err)
		http.Error(w, "Redis publish failed", http.StatusInternalServerError)
		return
	}

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

	// Set default timeout
	if req.Timeout == 0 {
		req.Timeout = 30 // 30 seconds default
	}

	// Subscribe to Redis channel
	subscription, err := b.redisClient.Subscribe(b.ctx, req.Channel)
	if err != nil {
		b.logger.Error("Error occurred", err)
		http.Error(w, "Redis subscribe failed", http.StatusInternalServerError)
		return
	}
	defer subscription.Close()

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

