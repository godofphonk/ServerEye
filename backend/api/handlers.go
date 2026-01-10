package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/godofphonk/ServerEye/pkg/publisher"
	"github.com/gorilla/mux"
)

type MetricsResponse struct {
	ServerID string              `json:"server_id"`
	Metrics  []*publisher.Metric `json:"metrics"`
	Count    int                 `json:"count"`
	From     time.Time           `json:"from,omitempty"`
	To       time.Time           `json:"to,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Services  map[string]interface{} `json:"services"`
}

type KeyRegistrationRequest struct {
	SecretKey    string `json:"secret_key"`
	AgentVersion string `json:"agent_version"`
	OSInfo       string `json:"os_info"`
	Hostname     string `json:"hostname"`
}

type KeyRegistrationResponse struct {
	Status    string `json:"status"`
	SecretKey string `json:"secret_key"`
}

type InternalWebhookRequest struct {
	SecretKey    string `json:"secret_key"`
	AgentVersion string `json:"agent_version"`
	OSInfo       string `json:"os_info"`
	Hostname     string `json:"hostname"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func (s *Server) handleInternalWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify webhook secret
	webhookSecret := r.Header.Get("X-Webhook-Secret")
	if webhookSecret != s.config.WebhookSecret {
		s.writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req InternalWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.SecretKey == "" {
		s.writeError(w, "secret_key is required", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.AgentVersion == "" {
		req.AgentVersion = "unknown"
	}
	if req.OSInfo == "" {
		req.OSInfo = "unknown"
	}
	if req.Hostname == "" {
		req.Hostname = "unknown"
	}
	if req.Status == "" {
		req.Status = "active"
	}

	// Insert into database
	if err := s.storage.InsertGeneratedKey(r.Context(), req.SecretKey, req.AgentVersion, req.OSInfo, req.Hostname); err != nil {
		s.logger.WithError(err).WithField("secret_key", req.SecretKey).Error("Failed to insert key from webhook")
		s.writeError(w, "Failed to store key", http.StatusInternalServerError)
		return
	}

	s.logger.WithField("secret_key", req.SecretKey).Info("Key synced from D1 webhook")

	response := KeyRegistrationResponse{
		Status:    "ok",
		SecretKey: req.SecretKey,
	}
	s.writeJSON(w, response)
}

func (s *Server) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	metricType := r.URL.Query().Get("type")
	serverID := r.URL.Query().Get("server_id")

	// Parse time range
	var fromTime, toTime time.Time
	var err error

	if from != "" {
		fromTime, err = time.Parse(time.RFC3339, from)
		if err != nil {
			s.writeError(w, "Invalid 'from' parameter", http.StatusBadRequest)
			return
		}
	} else {
		fromTime = time.Now().Add(-24 * time.Hour)
	}

	if to != "" {
		toTime, err = time.Parse(time.RFC3339, to)
		if err != nil {
			s.writeError(w, "Invalid 'to' parameter", http.StatusBadRequest)
			return
		}
	} else {
		toTime = time.Now()
	}

	// Get metrics
	var metrics []*publisher.Metric
	if serverID != "" {
		metrics, err = s.storage.GetMetricsHistory(r.Context(), serverID, metricType, fromTime, toTime)
	} else {
		// Return latest metrics for all servers
		servers, err := s.storage.GetServers(r.Context())
		if err != nil {
			s.writeError(w, "Failed to get servers", http.StatusInternalServerError)
			return
		}

		for _, sid := range servers {
			serverMetrics, err := s.storage.GetLatestMetrics(r.Context(), sid)
			if err != nil {
				s.logger.WithError(err).WithField("server", sid).Warn("Failed to get latest metrics")
				continue
			}
			metrics = append(metrics, serverMetrics...)
		}
	}

	if err != nil {
		s.writeError(w, "Failed to get metrics", http.StatusInternalServerError)
		return
	}

	response := MetricsResponse{
		ServerID: serverID,
		Metrics:  metrics,
		Count:    len(metrics),
		From:     fromTime,
		To:       toTime,
	}

	s.writeJSON(w, response)
}

func (s *Server) handleGetServerMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID := vars["serverID"]

	if serverID == "" {
		s.writeError(w, "Server ID is required", http.StatusBadRequest)
		return
	}

	metrics, err := s.storage.GetLatestMetrics(r.Context(), serverID)
	if err != nil {
		s.writeError(w, "Failed to get server metrics", http.StatusInternalServerError)
		return
	}

	response := MetricsResponse{
		ServerID: serverID,
		Metrics:  metrics,
		Count:    len(metrics),
	}

	s.writeJSON(w, response)
}

func (s *Server) handleGetMetricsHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID := vars["serverID"]

	if serverID == "" {
		s.writeError(w, "Server ID is required", http.StatusBadRequest)
		return
	}

	// Get query parameters
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	metricType := r.URL.Query().Get("type")
	limit := r.URL.Query().Get("limit")

	// Parse time range
	var fromTime, toTime time.Time
	var err error

	if from != "" {
		fromTime, err = time.Parse(time.RFC3339, from)
		if err != nil {
			s.writeError(w, "Invalid 'from' parameter", http.StatusBadRequest)
			return
		}
	} else {
		fromTime = time.Now().Add(-24 * time.Hour)
	}

	if to != "" {
		toTime, err = time.Parse(time.RFC3339, to)
		if err != nil {
			s.writeError(w, "Invalid 'to' parameter", http.StatusBadRequest)
			return
		}
	} else {
		toTime = time.Now()
	}

	// Get metrics
	metrics, err := s.storage.GetMetricsHistory(r.Context(), serverID, metricType, fromTime, toTime)
	if err != nil {
		s.writeError(w, "Failed to get metrics history", http.StatusInternalServerError)
		return
	}

	// Apply limit if specified
	if limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l < len(metrics) {
			metrics = metrics[:l]
		}
	}

	response := MetricsResponse{
		ServerID: serverID,
		Metrics:  metrics,
		Count:    len(metrics),
		From:     fromTime,
		To:       toTime,
	}

	s.writeJSON(w, response)
}

func (s *Server) handleGetServers(w http.ResponseWriter, r *http.Request) {
	servers, err := s.storage.GetServers(r.Context())
	if err != nil {
		s.writeError(w, "Failed to get servers", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"servers": servers,
		"count":   len(servers),
	}

	s.writeJSON(w, response)
}

func (s *Server) handleGetServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID := vars["serverID"]

	if serverID == "" {
		s.writeError(w, "Server ID is required", http.StatusBadRequest)
		return
	}

	metrics, err := s.storage.GetLatestMetrics(r.Context(), serverID)
	if err != nil {
		s.writeError(w, "Failed to get server", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"server_id": serverID,
		"metrics":   metrics,
		"last_seen": time.Now(), // Should be stored in DB
	}

	s.writeJSON(w, response)
}

func (s *Server) handleRegisterKey(w http.ResponseWriter, r *http.Request) {
	var req KeyRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.SecretKey == "" {
		s.writeError(w, "secret_key is required", http.StatusBadRequest)
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
		s.writeError(w, "Failed to register key", http.StatusInternalServerError)
		return
	}

	response := KeyRegistrationResponse{
		Status:    "ok",
		SecretKey: req.SecretKey,
	}
	s.writeJSON(w, response)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Services: map[string]interface{}{
			"database": "ok",
		},
	}

	// Check database health
	if err := s.storage.Ping(); err != nil {
		response.Status = "degraded"
		response.Services["database"] = "error"
	}

	s.writeJSON(w, response)
}

func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.WithError(err).Error("Failed to write JSON response")
	}
}

func (s *Server) writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	response := ErrorResponse{
		Error:   message,
		Code:    code,
		Message: http.StatusText(code),
	}

	json.NewEncoder(w).Encode(response)
}

// handleGetCommands retrieves pending commands for a server
func (s *Server) handleGetCommands(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID := vars["serverID"]

	if serverID == "" {
		s.writeError(w, "Server ID is required", http.StatusBadRequest)
		return
	}

	commands, err := s.storage.GetPendingCommands(r.Context(), serverID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get commands")
		s.writeError(w, "Failed to get commands", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"commands": commands,
		"count":    len(commands),
	}

	s.writeJSON(w, response)
}

// handlePostMetrics handles incoming metrics from agents
func (s *Server) handlePostMetrics(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var metric struct {
		ServerID  string                 `json:"server_id"`
		ServerKey string                 `json:"server_key"`
		Type      string                 `json:"type"`
		Value     interface{}            `json:"value"`
		Timestamp time.Time              `json:"timestamp"`
		Tags      map[string]string      `json:"tags"`
		Data      map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		s.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if metric.ServerID == "" {
		s.writeError(w, "server_id is required", http.StatusBadRequest)
		return
	}
	if metric.ServerKey == "" {
		s.writeError(w, "server_key is required", http.StatusBadRequest)
		return
	}
	if metric.Type == "" {
		s.writeError(w, "type is required", http.StatusBadRequest)
		return
	}

	// Store metric
	metricToStore := &publisher.Metric{
		ServerID:  metric.ServerID,
		ServerKey: metric.ServerKey,
		Type:      metric.Type,
		Value:     metric.Value,
		Timestamp: metric.Timestamp,
		Tags:      metric.Tags,
		Data:      metric.Data,
	}

	if err := s.storage.StoreMetric(r.Context(), metricToStore); err != nil {
		s.logger.WithError(err).Error("Failed to store metric")
		s.writeError(w, "Failed to store metric", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	s.writeJSON(w, map[string]string{"status": "accepted"})
}
