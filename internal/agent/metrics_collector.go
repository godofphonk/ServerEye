package agent

import (
	"time"
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

	// Network метрики (временно отключены для dev тестирования)
	// TODO: Добавить NetworkUsage поле в конфигурацию
	/*
		if a.config.Metrics.NetworkUsage && a.systemMonitor != nil {
			if networkInfo, err := a.systemMonitor.GetNetworkInfo(); err == nil {
				for _, iface := range networkInfo.Interfaces {
					tags := map[string]string{
						"interface": iface.Name,
					}

					// Bytes sent/recv в GB
					bytesSentGB := float64(iface.BytesSent) / 1024 / 1024 / 1024
					bytesRecvGB := float64(iface.BytesRecv) / 1024 / 1024 / 1024

					metric := a.CreateMetricFromData("network_bytes_sent", bytesSentGB, tags)
					if err := a.wsPublisher.Publish(a.ctx, metric); err != nil {
						a.logger.WithError(err).Error("Failed to send network metric")
					}

					metric = a.CreateMetricFromData("network_bytes_recv", bytesRecvGB, tags)
					if err := a.wsPublisher.Publish(a.ctx, metric); err != nil {
						a.logger.WithError(err).Error("Failed to send network metric")
					}
				}
			}
		}
	*/

	// Docker containers метрики
	a.logger.Info("Checking docker client for containers metrics")
	if a.dockerClient != nil {
		a.logger.Info("Docker client is not nil, getting containers")
		if containers, err := a.dockerClient.GetContainers(a.ctx); err == nil {
			a.logger.WithField("containers_count", containers.Total).Info("Got containers payload, attempting to publish")
			// Отправляем информацию о контейнерах как метрику
			metric := a.CreateMetricFromData("containers", containers, nil)
			if err := a.wsPublisher.Publish(a.ctx, metric); err != nil {
				a.logger.WithError(err).Error("Failed to send containers metric")
			} else {
				a.logger.WithField("containers_count", containers.Total).Info("Containers metric sent successfully")
			}
		} else {
			a.logger.WithError(err).Info("Docker not available or no containers")
		}
	} else {
		a.logger.Info("Docker client is nil, skipping containers metrics")
	}
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
