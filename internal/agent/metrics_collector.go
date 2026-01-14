package agent

import (
	"time"

	"github.com/godofphonk/ServerEye/pkg/types"
)

// collectAndSendMetrics собирает и отправляет все метрики в цикле
func (a *Agent) collectAndSendMetrics() {
	a.logger.Info("Starting metrics collection loop")

	// Parse metrics interval from config
	interval := 30 * time.Second // default
	if a.config.Metrics.Interval != "" {
		if parsedInterval, err := time.ParseDuration(a.config.Metrics.Interval); err == nil {
			interval = parsedInterval
		} else {
			a.logger.WithError(err).Warn("Failed to parse metrics interval, using default 30s")
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Send first metrics immediately
	a.collectAndSendMetricsOnce()

	for {
		select {
		case <-ticker.C:
			a.collectAndSendMetricsOnce()
		case <-a.ctx.Done():
			a.logger.Info("Metrics collection stopped")
			return
		}
	}
}

// collectAndSendMetricsOnce собирает и отправляет метрики один раз
func (a *Agent) collectAndSendMetricsOnce() {
	a.logger.Info("collectAndSendMetrics() called - starting metrics collection")

	// Check if WebSocket publisher is available
	if a.wsPublisher == nil {
		a.logger.Error("No WebSocket publisher available - cannot publish metrics")
		return
	}

	a.logger.Info("Publisher is available, proceeding with metrics collection")

	// Collect all metrics and send as unified message
	a.collectAndSendUnifiedMetrics()

	// DISABLED: Individual metrics to avoid conflicts with unified metrics
	// Only send unified metrics to prevent Redis overwrites

	/*
		// CPU Temperature (if enabled and cpuMetrics available)
		if a.config.Metrics.CPUTemperature && a.cpuMetrics != nil {
			a.logger.Info("Attempting to collect CPU temperature")
			if temp, err := a.cpuMetrics.GetTemperature(); err == nil {
				a.logger.WithField("temperature", temp).Info("CPU temperature collected")
				a.sendMetric("cpu_temperature", temp, "°C")
			} else {
				a.logger.WithError(err).Error("Failed to get CPU temperature")
			}
		} else {
			a.logger.WithFields(map[string]interface{}{
				"enabled":    a.config.Metrics.CPUTemperature,
				"cpuMetrics": a.cpuMetrics != nil,
			}).Info("CPU temperature collection skipped")
		}

		// Memory метрики (if enabled and systemMonitor available)
		if a.config.Metrics.MemoryUsage && a.systemMonitor != nil {
			if memInfo, err := a.systemMonitor.GetMemoryInfo(); err == nil {
				a.sendMetric("memory_usage", memInfo.UsedPercent, "%")
				a.sendMetric("memory_total", float64(memInfo.Total)/1024/1024/1024, "GB")
				a.sendMetric("memory_used", float64(memInfo.Used)/1024/1024/1024, "GB")
				a.sendMetric("memory_available", float64(memInfo.Available)/1024/1024/1024, "GB")
			}

			// Disk метрики (if enabled)
			if a.config.Metrics.DiskUsage {
				if diskInfo, err := a.systemMonitor.GetDiskInfo(); err == nil {
					for _, disk := range diskInfo.Disks {
						// Отправляем информацию о каждом диске
						tags := map[string]string{
							"path": disk.Path,
						}
						metric := a.CreateMetricFromData("disk_usage", disk.UsedPercent, tags)
						if err := a.wsPublisher.Publish(a.ctx, metric); err != nil {
							a.logger.WithError(err).Error("Failed to send disk metric")
						}
					}
				}
			}
		}
	*/
}

// sendMetric отправляет метрику через publisher
func (a *Agent) sendMetric(metricType string, value float64, unit string) {
	// Check if WebSocket publisher is available
	if a.wsPublisher == nil {
		a.logger.Error("No WebSocket publisher available - cannot send metric")
		return
	}

	tags := map[string]string{
		"unit": unit,
	}

	metric := a.CreateMetricFromData(metricType, value, tags)

	// Log the actual topic and metric data before publishing
	a.logger.WithFields(map[string]interface{}{
		"type":       metric.Type,
		"server_id":  metric.ServerID,
		"server_key": metric.ServerKey,
		"value":      metric.Value,
	}).Info("Publishing metric via WebSocket API")

	if err := a.wsPublisher.Publish(a.ctx, metric); err != nil {
		a.logger.WithError(err).WithField("type", metricType).Error("Failed to send metric")
	} else {
		a.logger.WithField("type", metricType).Info("Metric sent successfully")
	}
}

// collectAndSendUnifiedMetrics collects all metrics and sends them as unified message
func (a *Agent) collectAndSendUnifiedMetrics() {
	// Create unified metrics structure
	metrics := map[string]interface{}{
		"time": time.Now().Format(time.RFC3339),
	}

	// Collect CPU usage (detailed statistics)
	if a.cpuMetrics != nil {
		if cpuUsage, err := a.cpuMetrics.GetDetailedUsage(); err == nil {
			a.logger.WithField("cpu_usage", cpuUsage).Info("Detailed CPU usage collected")

			// Add CPU metrics to unified structure
			metrics["cpu"] = cpuUsage.UsageTotal // For backward compatibility
			metrics["cpu_usage"] = map[string]interface{}{
				"usage_total":  cpuUsage.UsageTotal,
				"usage_user":   cpuUsage.UsageUser,
				"usage_system": cpuUsage.UsageSystem,
				"usage_idle":   cpuUsage.UsageIdle,
				"cores":        cpuUsage.Cores,
				"frequency":    cpuUsage.Frequency,
			}

			if cpuUsage.LoadAverage != nil {
				metrics["cpu_usage"].(map[string]interface{})["load_average"] = map[string]interface{}{
					"load_1min":  cpuUsage.LoadAverage.Load1Min,
					"load_5min":  cpuUsage.LoadAverage.Load5Min,
					"load_15min": cpuUsage.LoadAverage.Load15Min,
				}
			}
		} else {
			a.logger.WithError(err).Error("Failed to get detailed CPU usage")
		}
	}

	// Create unified metric message
	unifiedMetric := &types.Metric{
		ServerID:   a.config.Server.ServerID,
		ServerKey:  a.config.Server.SecretKey,
		ServerName: a.config.Server.Name,
		Type:       "metrics",
		Version:    "1.0",
		Value:      nil,
		Timestamp:  time.Now(),
		Data: map[string]interface{}{
			"server_id": a.config.Server.ServerID,
			"metrics":   metrics,
		},
	}

	// Send unified metrics
	if err := a.wsPublisher.Publish(a.ctx, unifiedMetric); err != nil {
		a.logger.WithError(err).Error("Failed to send unified metrics")
	} else {
		a.logger.WithField("metrics_count", len(metrics)).Info("Unified metrics sent successfully")
	}
}
