package api

import (
	"context"
	"net/http"
	"time"

	"github.com/godofphonk/ServerEye/backend/internal/config"
	"github.com/godofphonk/ServerEye/backend/internal/storage"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Server struct {
	config   *config.Config
	storage  *storage.PostgresStorage
	logger   *logrus.Logger
	server   *http.Server
	wsServer *WebSocketServer
}

func New(cfg *config.Config, storage *storage.PostgresStorage, logger *logrus.Logger) *Server {
	s := &Server{
		config:  cfg,
		storage: storage,
		logger:  logger,
	}

	// Initialize WebSocket server
	s.wsServer = NewWebSocketServer(s, logger)

	// Initialize rate limiter (100 requests per minute)
	rateLimiter := NewRateLimiter(100, 20, logger)

	router := s.setupRoutes()

	// Apply global middleware (not auth)
	router.Use(s.loggingMiddleware)
	router.Use(rateLimiter.Middleware)
	router.Use(s.corsMiddleware)

	s.server = &http.Server{
		Addr:         cfg.GetAddr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// GetWebSocketServer returns the WebSocket server instance
func (s *Server) GetWebSocketServer() *WebSocketServer {
	return s.wsServer
}

func (s *Server) setupRoutes() *mux.Router {
	router := mux.NewRouter()

	s.logger.Info("Setting up routes...")

	// Public routes (no auth required)
	public := router.PathPrefix("/public").Subrouter()
	router.HandleFunc("/api/v1/register-key", s.handleRegisterKey).Methods("POST")
	router.HandleFunc("/v1/register-key", s.handleRegisterKey).Methods("POST")
	router.HandleFunc("/health", s.handleHealth).Methods("GET")

	s.logger.Info("Registered public routes")
	s.logger.Info("Available routes:")
	router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		s.logger.Infof("Route: %s %s", methods, path)
		return nil
	})

	// Internal webhook route (authenticated by webhook secret)
	internal := router.PathPrefix("/internal").Subrouter()
	internal.HandleFunc("/register-key", s.handleInternalWebhook).Methods("POST")

	// Authenticated routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Direct v1 routes (for Cloudflare compatibility)
	v1 := router.PathPrefix("/v1").Subrouter()

	// Metrics endpoints
	api.HandleFunc("/metrics", s.handleGetMetrics).Methods("GET")
	api.HandleFunc("/metrics/{serverID}", s.handleGetServerMetrics).Methods("GET")
	api.HandleFunc("/metrics/{serverID}/history", s.handleGetMetricsHistory).Methods("GET")
	api.HandleFunc("/metrics", s.handlePostMetrics).Methods("POST")

	// V1 metrics endpoints
	v1.HandleFunc("/metrics", s.handleGetMetrics).Methods("GET")
	v1.HandleFunc("/metrics/{serverID}", s.handleGetServerMetrics).Methods("GET")
	v1.HandleFunc("/metrics/{serverID}/history", s.handleGetMetricsHistory).Methods("GET")
	v1.HandleFunc("/metrics", s.handlePostMetrics).Methods("POST")

	// Servers endpoints
	api.HandleFunc("/servers", s.handleGetServers).Methods("GET")
	api.HandleFunc("/servers/{serverID}", s.handleGetServer).Methods("GET")

	// V1 servers endpoints
	v1.HandleFunc("/servers", s.handleGetServers).Methods("GET")
	v1.HandleFunc("/servers/{serverID}", s.handleGetServer).Methods("GET")

	// WebSocket for live updates
	api.HandleFunc("/ws", s.wsServer.handleWebSocket).Methods("GET")
	v1.HandleFunc("/ws", s.wsServer.handleWebSocket).Methods("GET")

	// Commands endpoints
	api.HandleFunc("/commands/{serverID}", s.handleGetCommands).Methods("GET")
	v1.HandleFunc("/commands/{serverID}", s.handleGetCommands).Methods("GET")

	// Middleware - apply auth only to authenticated routes
	router.Use(s.corsMiddleware)
	api.Use(s.authMiddleware)
	// v1 routes are public - no auth middleware

	return router
}

func (s *Server) Start() error {
	s.logger.WithField("addr", s.server.Addr).Info("Starting API server")
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down API server")
	return s.server.Shutdown(ctx)
}
