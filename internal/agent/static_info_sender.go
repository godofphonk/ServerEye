package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/godofphonk/ServerEye/pkg/metrics"
	"github.com/sirupsen/logrus"
)

// sendStaticInfoPeriodically sends static server information once per day
func (a *Agent) sendStaticInfoPeriodically() {
	// Send immediately on startup
	a.sendStaticInfo()

	// Then send once per day
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.sendStaticInfo()
		case <-a.ctx.Done():
			a.logger.Info("Static info sender stopped")
			return
		}
	}
}

// sendStaticInfo collects and sends static server information
func (a *Agent) sendStaticInfo() {
	a.logger.Info("Collecting and sending static server information")

	// Create new logrus logger for collector
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create static info collector
	collector := metrics.NewStaticInfoCollector(logger)

	// Collect static information
	staticInfo, err := collector.CollectStaticInfo()
	if err != nil {
		a.logger.WithError(err).Error("Failed to collect static server information")
		return
	}

	// Send via HTTP POST
	if err := a.sendStaticInfoHTTP(staticInfo); err != nil {
		a.logger.WithError(err).Error("Failed to send static server information")
	} else {
		a.logger.Info("Static server information sent successfully")
	}
}

// sendStaticInfoHTTP sends static info via HTTP POST
func (a *Agent) sendStaticInfoHTTP(staticInfo interface{}) error {
	// Build endpoint URL
	endpoint := fmt.Sprintf("%s/api/servers/by-key/%s/static-info",
		a.config.API.BaseURL,
		a.config.Server.SecretKey,
	)

	// Marshal to JSON
	jsonData, err := json.Marshal(staticInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal static info: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(a.ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if a.config.API.APIKey != "" {
		req.Header.Set("X-API-Key", a.config.API.APIKey)
	}

	// Send request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	a.logger.WithField("status_code", resp.StatusCode).Debug("Static info sent successfully")
	return nil
}
