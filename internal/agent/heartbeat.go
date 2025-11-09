package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// startHeartbeat запускает отправку heartbeat сообщений
func (a *Agent) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
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

// sendHeartbeat отправляет heartbeat сообщение в Web API
func (a *Agent) sendHeartbeat() {
	// Check if Web API base URL is configured
	webAPIURL := a.config.API.BaseURL
	if webAPIURL == "" {
		// Fallback to Redis if Web API not configured
		a.sendHeartbeatRedis()
		return
	}

	heartbeat := map[string]interface{}{
		"api_key": a.config.Server.SecretKey,
	}

	data, err := json.Marshal(heartbeat)
	if err != nil {
		a.logger.WithError(err).Error("Не удалось сериализовать heartbeat")
		return
	}

	// Send to Web API
	url := fmt.Sprintf("%s/api/v1/servers/heartbeat", webAPIURL)
	req, err := http.NewRequestWithContext(a.ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		a.logger.WithError(err).Error("Не удалось создать heartbeat запрос")
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		a.logger.WithError(err).Error("Не удалось отправить heartbeat в Web API")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.logger.WithField("status", resp.StatusCode).Warn("Heartbeat вернул не-OK статус")
	}
}

// sendHeartbeatRedis отправляет heartbeat в Redis (legacy/fallback)
func (a *Agent) sendHeartbeatRedis() {
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
