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

// sendHeartbeat отправляет heartbeat сообщение через HTTP
func (a *Agent) sendHeartbeat() {
	// Check if HTTP publisher is available
	if a.httpPublisher == nil {
		a.logger.Warn("Heartbeat skipped: No HTTP publisher available")
		return
	}

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

	a.logger.Info("Sending heartbeat via HTTP")
	if err := a.httpPublisher.Publish(a.ctx, metric); err != nil {
		a.logger.WithError(err).Error("Failed to send heartbeat via HTTP")
	} else {
		a.logger.Info("Heartbeat sent successfully via HTTP")
	}
}
