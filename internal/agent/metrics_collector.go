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
			a.collectAndSendMetrics()
		case <-a.ctx.Done():
			a.logger.Info("Metrics collection stopped")
			return
		}
	}
}

// collectAndSendMetrics собирает и отправляет все метрики
func (a *Agent) collectAndSendMetrics() {
	// CPU Temperature (if enabled and cpuMetrics available)
	if a.config.Metrics.CPUTemperature && a.cpuMetrics != nil {
		if temp, err := a.cpuMetrics.GetTemperature(); err == nil {
			a.sendMetric("cpu_temperature", temp, "°C")
		}
	}

	// Memory метрики (if systemMonitor available)
	if a.systemMonitor != nil {
		if memInfo, err := a.systemMonitor.GetMemoryInfo(); err == nil {
			a.sendMetric("memory_usage", memInfo.UsedPercent, "%")
			a.sendMetric("memory_total", float64(memInfo.Total)/1024/1024/1024, "GB")
			a.sendMetric("memory_used", float64(memInfo.Used)/1024/1024/1024, "GB")
			a.sendMetric("memory_available", float64(memInfo.Available)/1024/1024/1024, "GB")
		}

		// Disk метрики
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

		// Network метрики
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
	}

	// Docker containers метрики
	if a.dockerClient != nil {
		if containersPayload, err := a.dockerClient.GetContainers(a.ctx); err == nil {
			// Отправляем информацию о контейнерах как метрику
			metric := a.CreateMetricFromData("containers", containersPayload, nil)
			if err := a.metricPublisher.Publish(a.ctx, metric); err != nil {
				a.logger.WithError(err).Error("Failed to send containers metric")
			} else {
				a.logger.WithField("containers_count", containersPayload.Total).Debug("Containers metric sent successfully")
			}
		} else {
			a.logger.WithError(err).Debug("Docker not available or no containers")
		}
	}
}

// sendMetric отправляет метрику в Kafka
func (a *Agent) sendMetric(metricType string, value float64, unit string) {
	if a.metricPublisher == nil {
		return
	}

	tags := map[string]string{
		"unit": unit,
	}

	metric := a.CreateMetricFromData(metricType, value, tags)

	if err := a.metricPublisher.Publish(a.ctx, metric); err != nil {
		a.logger.WithError(err).WithField("type", metricType).Error("Failed to send metric")
	} else {
		a.logger.WithField("type", metricType).Debug("Metric sent successfully")
	}
}
