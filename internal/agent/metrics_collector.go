package agent

import (
	"time"
)

// startMetricsCollection запускает периодический сбор метрик
func (a *Agent) startMetricsCollection() {
	interval, err := time.ParseDuration(a.config.Metrics.Interval)
	if err != nil || interval == 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	a.logger.Info("Metrics collection started")

	// Send first batch immediately
	a.collectAndSendMetrics()

	for {
		select {
		case <-ticker.C:
			a.logger.Info("Ticker fired - calling collectAndSendMetrics()")
			a.collectAndSendMetrics()
		case <-a.ctx.Done():
			a.logger.Info("Metrics collection stopped")
			return
		}
	}
}

// collectAndSendMetrics собирает и отправляет все метрики
func (a *Agent) collectAndSendMetrics() {
	a.logger.Info("collectAndSendMetrics() called - starting metrics collection")

	// Debug: выводим конфигурацию метрик
	a.logger.WithFields(map[string]interface{}{
		"CPUUsage":       a.config.Metrics.CPUUsage,
		"MemoryUsage":    a.config.Metrics.MemoryUsage,
		"DiskUsage":      a.config.Metrics.DiskUsage,
		"CPUTemperature": a.config.Metrics.CPUTemperature,
		"Interval":       a.config.Metrics.Interval,
	}).Info("Metrics configuration loaded")

	// Debug: проверяем metricPublisher
	if a.metricPublisher == nil {
		a.logger.Error("metricPublisher is nil - cannot publish metrics")
		return
	}

	a.logger.Info("metricPublisher is available, proceeding with metrics collection")

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
					if err := a.metricPublisher.Publish(a.ctx, metric); err != nil {
						a.logger.WithError(err).Error("Failed to send disk metric")
					}
				}
			}
		}
	}

	// Network метрики (временно отключены для dev тестирования)
	// TODO: Добавить NetworkUsage поле в конфигурацию
	/*
		if networkInfo, err := a.systemMonitor.GetNetworkInfo(); err == nil {
			a.sendMetric("network_download_speed", networkInfo.DownloadSpeed, "Mbps")
			a.sendMetric("network_upload_speed", networkInfo.UploadSpeed, "Mbps")
			a.sendMetric("network_total_download", float64(networkInfo.TotalDownload), "GB")
			a.sendMetric("network_total_upload", float64(networkInfo.TotalUpload), "GB")

			// Отправляем метрики для каждого интерфейса
			for _, iface := range networkInfo.Interfaces {
				tags := map[string]string{
					"interface": iface.Name,
				}

				// Bytes sent/recv в GB
				bytesSentGB := float64(iface.BytesSent) / 1024 / 1024 / 1024
				bytesRecvGB := float64(iface.BytesRecv) / 1024 / 1024 / 1024

				metric := a.CreateMetricFromData("network_bytes_sent", bytesSentGB, tags)
				if err := a.metricPublisher.Publish(a.ctx, metric); err != nil {
					a.logger.WithError(err).Error("Failed to send network metric")
				}

				metric = a.CreateMetricFromData("network_bytes_recv", bytesRecvGB, tags)
				if err := a.metricPublisher.Publish(a.ctx, metric); err != nil {
					a.logger.WithError(err).Error("Failed to send network metric")
				}
			}
		}
	*/

	// Docker containers метрики
	a.logger.Info("Checking docker client for containers metrics")
	if a.dockerClient != nil {
		a.logger.Info("Docker client is not nil, getting containers")
		if containersPayload, err := a.dockerClient.GetContainers(a.ctx); err == nil {
			a.logger.WithField("containers_count", containersPayload.Total).Info("Got containers payload, attempting to publish")
			// Отправляем информацию о контейнерах как метрику
			metric := a.CreateMetricFromData("containers", containersPayload, nil)
			if err := a.metricPublisher.Publish(a.ctx, metric); err != nil {
				a.logger.WithError(err).Error("Failed to send containers metric")
			} else {
				a.logger.WithField("containers_count", containersPayload.Total).Info("Containers metric sent successfully")
			}
		} else {
			a.logger.WithError(err).Info("Docker not available or no containers")
		}
	} else {
		a.logger.Info("Docker client is nil, skipping containers metrics")
	}
}

// sendMetric отправляет метрику в Kafka
func (a *Agent) sendMetric(metricType string, value float64, unit string) {
	if a.metricPublisher == nil {
		a.logger.Error("metricPublisher is nil - cannot send metric")
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
	}).Info("Publishing metric to Kafka")

	if err := a.metricPublisher.Publish(a.ctx, metric); err != nil {
		a.logger.WithError(err).WithField("type", metricType).Error("Failed to send metric")
	} else {
		a.logger.WithField("type", metricType).Info("Metric sent successfully")
	}
}
