package bot

import (
	"encoding/json"
	"net/http"
	"strings"
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
			b.legacyLogger.WithField("panic", r).Error("HTTP server panicked")
		}
	}()

	b.legacyLogger.Info("Setting up HTTP routes...")
	http.HandleFunc("/api/register-key", b.handleRegisterKey)
	http.HandleFunc("/api/health", b.handleHealth)
	
	b.legacyLogger.Info("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		b.legacyLogger.WithError(err).Error("HTTP server failed")
	}
}

// handleRegisterKey handles key registration from agent
func (b *Bot) handleRegisterKey(w http.ResponseWriter, r *http.Request) {
	b.legacyLogger.WithField("method", r.Method).WithField("url", r.URL.Path).Info("HTTP request received")
	
	if r.Method != http.MethodPost {
		b.legacyLogger.Error("Invalid method for register-key")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req KeyRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		b.legacyLogger.WithError(err).Error("Failed to decode JSON")
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	b.legacyLogger.WithField("secret_key", req.SecretKey).Info("Received key registration request")

	// Validate secret key
	if !strings.HasPrefix(req.SecretKey, "srv_") {
		http.Error(w, "Invalid secret key format", http.StatusBadRequest)
		return
	}

	// Record the key
	if err := b.recordGeneratedKey(req.SecretKey); err != nil {
		b.legacyLogger.WithError(err).Error("Failed to record generated key")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If agent info provided, update connection info
	if req.AgentVersion != "" || req.OSInfo != "" || req.Hostname != "" {
		if err := b.updateKeyConnection(req.SecretKey, req.AgentVersion, req.OSInfo, req.Hostname); err != nil {
			b.legacyLogger.WithError(err).Error("Failed to update key connection info")
		}
	}

	b.legacyLogger.WithField("key_prefix", req.SecretKey[:12]+"...").Info("Key registered via HTTP API")

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
