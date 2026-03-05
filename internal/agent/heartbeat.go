package agent

import (
	"time"

	"github.com/godofphonk/ServerEye/pkg/types"
)

// startHeartbeat запускает отправку heartbeat сообщений через HTTP
func (a *Agent) startHeartbeat() {
	ticker := time.NewTicker(60 * time.Second) // 60 секунд как требует бэкенд
	defer ticker.Stop()

	// Send first heartbeat immediately
	a.sendHeartbeat()

	for {
		select {
		case <-ticker.C:
			a.sendHeartbeat()
		case <-a.ctx.Done():
			return
		}
	}
}

// sendHeartbeat отправляет heartbeat сообщение через WebSocket или HTTP
func (a *Agent) sendHeartbeat() {
	// Create heartbeat metric with empty data as per API spec
	metric := &types.Metric{
		ServerID:   a.config.Server.ServerID,
		ServerKey:  a.config.Server.SecretKey,
		ServerName: a.config.Server.Name,
		Type:       "heartbeat",
		Version:    "1.0",
		Value:      nil,
		Timestamp:  time.Now(),
		Data:       map[string]interface{}{}, // Empty data for heartbeat
	}

	// Always try WebSocket first when enabled
	if a.useWebSocket && a.wsPublisher != nil {
		a.logger.Debug("Sending heartbeat via WebSocket")
		if err := a.wsPublisher.Publish(a.ctx, metric); err != nil {
			a.logger.WithError(err).Debug("Failed to send heartbeat via WebSocket, falling back to HTTP")
			a.sendHeartbeatHTTP(metric)
		} else {
			a.logger.Debug("Heartbeat sent successfully via WebSocket")
		}
	} else {
		// Use HTTP fallback
		a.sendHeartbeatHTTP(metric)
	}
}

// sendHeartbeatHTTP отправляет heartbeat через HTTP
func (a *Agent) sendHeartbeatHTTP(metric *types.Metric) {
	// Check if HTTP publisher is available
	if a.httpPublisher == nil {
		a.logger.Warn("Heartbeat skipped: No HTTP publisher available")
		return
	}

	a.logger.Debug("Sending heartbeat via HTTP")
	if err := a.httpPublisher.Publish(a.ctx, metric); err != nil {
		a.logger.WithError(err).Error("Failed to send heartbeat via HTTP")
	} else {
		a.logger.Debug("Heartbeat sent successfully via HTTP")
	}
}
