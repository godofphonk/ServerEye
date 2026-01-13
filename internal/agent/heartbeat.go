package agent

import (
	"time"
)

// startHeartbeat запускает отправку heartbeat сообщений через WebSocket
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

// sendHeartbeat отправляет heartbeat сообщение через WebSocket
func (a *Agent) sendHeartbeat() {
	// Check if WebSocket publisher is available
	if a.wsPublisher == nil {
		a.logger.Warn("Heartbeat skipped: No WebSocket publisher available")
		return
	}

	// Check if WebSocket is connected
	if !a.wsPublisher.IsConnected() {
		a.logger.Warn("Heartbeat skipped: WebSocket not connected")
		return
	}

	// Create heartbeat metric
	heartbeatData := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now().Unix(),
		"uptime":    "unknown", // TODO: добавить реальный uptime
	}

	tags := map[string]string{
		"type": "heartbeat",
	}

	metric := a.CreateMetricFromData("heartbeat", heartbeatData, tags)

	a.logger.Info("Sending heartbeat via WebSocket")
	if err := a.wsPublisher.Publish(a.ctx, metric); err != nil {
		a.logger.WithError(err).Error("Failed to send heartbeat")
	} else {
		a.logger.Info("Heartbeat sent successfully")
	}
}
