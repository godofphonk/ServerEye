package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

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

type WSSubscribeMessage struct {
	ServerID string `json:"server_id"`
	Metric   string `json:"metric"`
}

func NewWebSocketServer(server *Server, logger *logrus.Logger) *WebSocketServer {
	return &WebSocketServer{
		server:  server,
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin in development
				// In production, check against allowed origins
				return true
			},
		},
		logger: logger,
	}
}

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

	// Handle messages
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

func (ws *WebSocketServer) handleSubscription(conn *websocket.Conn, msg WSMessage) {
	data, err := json.Marshal(msg.Data)
	if err != nil {
		ws.logger.WithError(err).Error("Failed to marshal subscription data")
		return
	}

	var sub WSSubscribeMessage
	if err := json.Unmarshal(data, &sub); err != nil {
		ws.logger.WithError(err).Error("Failed to unmarshal subscription message")
		return
	}

	// Send current metrics for the subscription
	if sub.ServerID != "" {
		metrics, err := ws.server.storage.GetLatestMetrics(context.Background(), sub.ServerID)
		if err != nil {
			ws.logger.WithError(err).WithField("server", sub.ServerID).Error("Failed to get latest metrics")
			return
		}

		response := WSMessage{
			Type:      "metrics",
			Data:      metrics,
			Timestamp: time.Now(),
		}

		if err := conn.WriteJSON(response); err != nil {
			ws.logger.WithError(err).Error("Failed to send metrics")
		}
	}
}

// BroadcastMetric sends metric to all connected clients
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
			ws.logger.WithError(err).Error("Failed to broadcast metric")
			// Remove client on error
			delete(ws.clients, conn)
			conn.Close()
		}
	}
}

// GetClientCount returns the number of connected WebSocket clients
func (ws *WebSocketServer) GetClientCount() int {
	ws.clientsM.RLock()
	defer ws.clientsM.RUnlock()
	return len(ws.clients)
}
