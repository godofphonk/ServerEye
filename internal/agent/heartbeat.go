package agent

import (
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
		// In Kafka-only mode, we don't send heartbeat via Redis
		a.logger.Warn("Heartbeat skipped: Web API not configured and Redis removed")
		return
	}

	// Send to Web API
	url := fmt.Sprintf("%s/health", webAPIURL)
	req, err := http.NewRequestWithContext(a.ctx, "GET", url, nil)
	if err != nil {
		a.logger.WithError(err).Error("Не удалось создать heartbeat запрос")
		return
	}

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
