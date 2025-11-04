package agent

import (
	"encoding/json"
	"fmt"
	"time"
)

// startHeartbeat запускает отправку heartbeat сообщений
func (a *Agent) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.sendHeartbeat()
		case <-a.ctx.Done():
			return
		}
	}
}

// sendHeartbeat отправляет heartbeat сообщение
func (a *Agent) sendHeartbeat() {
	heartbeat := map[string]interface{}{
		"server_key":  a.config.Server.SecretKey,
		"server_name": a.config.Server.Name,
		"timestamp":   time.Now(),
		"status":      "online",
	}

	data, err := json.Marshal(heartbeat)
	if err != nil {
		a.logger.WithError(err).Error("Не удалось сериализовать heartbeat")
		return
	}

	heartbeatChannel := fmt.Sprintf("heartbeat:%s", a.config.Server.SecretKey)
	if err := a.redisClient.Publish(a.ctx, heartbeatChannel, data); err != nil {
		a.logger.WithError(err).Error("Не удалось отправить heartbeat")
	}
}
