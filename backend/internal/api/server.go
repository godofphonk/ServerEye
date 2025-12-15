package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/servereye/servereye/pkg/kafka"
	"github.com/servereye/servereye/pkg/publisher"
	"github.com/sirupsen/logrus"
)

type Server struct {
	config        *Config
	logger        *logrus.Logger
	kafkaProducer *kafka.Producer
	httpServer    *http.Server
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

func New(cfg *Config, logger *logrus.Logger) (*Server, error) {
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

	return &Server{
		config:        cfg,
		logger:        logger,
		kafkaProducer: kafkaProducer,
	}, nil
}

func (s *Server) Start() error {
	router := mux.NewRouter()

	// Metrics endpoint
	router.HandleFunc("/v1/metrics", s.handleMetrics).Methods("POST")

	// Health check
	router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Add middleware
	router.Use(s.authMiddleware)
	router.Use(s.loggingMiddleware)

	addr := fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.WithField("addr", addr).Info("Starting HTTP server")
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
