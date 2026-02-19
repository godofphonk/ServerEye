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
//
//nolint:unused // This function is used within the same file
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

	// Collect all metric types
	a.collectCPUMetrics(metrics)
	a.collectMemoryMetrics(metrics)
	a.collectDiskMetrics(metrics)
	a.collectNetworkMetrics(metrics)
	a.collectTemperatureMetrics(metrics)
	a.collectSystemMetrics(metrics)

	// Create and send unified metric message
	a.sendUnifiedMetrics(metrics)
}

// collectCPUMetrics collects CPU metrics and adds them to the unified structure
func (a *Agent) collectCPUMetrics(metrics map[string]interface{}) {
	if a.cpuMetrics == nil {
		return
	}

	cpuUsage, err := a.cpuMetrics.GetDetailedUsage()
	if err != nil {
		a.logger.WithError(err).Error("Failed to get detailed CPU usage")
		return
	}

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
}

// collectMemoryMetrics collects memory metrics and adds them to the unified structure
func (a *Agent) collectMemoryMetrics(metrics map[string]interface{}) {
	if a.systemMonitor == nil {
		a.logger.Error("System monitor is nil - memory metrics collection skipped")
		return
	}

	memInfo, err := a.systemMonitor.GetMemoryInfo()
	if err != nil {
		a.logger.WithError(err).Error("Failed to get memory info")
		return
	}

	a.logger.WithField("memory_info", memInfo).Info("Memory info collected")

	// Add memory metrics to unified structure
	metrics["memory"] = memInfo.UsedPercent // For backward compatibility
	memoryDetails := map[string]interface{}{
		"total_gb":     float64(memInfo.Total) / 1024 / 1024 / 1024,
		"used_gb":      float64(memInfo.Used) / 1024 / 1024 / 1024,
		"available_gb": float64(memInfo.Available) / 1024 / 1024 / 1024,
		"free_gb":      float64(memInfo.Free) / 1024 / 1024 / 1024,
		"buffers_gb":   float64(memInfo.Buffers) / 1024 / 1024 / 1024,
		"cached_gb":    float64(memInfo.Cached) / 1024 / 1024 / 1024,
		"used_percent": memInfo.UsedPercent,
	}
	metrics["memory_details"] = memoryDetails
	a.logger.WithField("memory_details", memoryDetails).Debug("Memory details added to metrics")
}

// collectDiskMetrics collects disk metrics and adds them to the unified structure
func (a *Agent) collectDiskMetrics(metrics map[string]interface{}) {
	if a.systemMonitor == nil {
		a.logger.Error("System monitor is nil - disk metrics collection skipped")
		return
	}

	diskInfo, err := a.systemMonitor.GetDiskInfo()
	if err != nil {
		a.logger.WithError(err).Error("Failed to get disk info")
		return
	}

	a.logger.WithField("disk_info", diskInfo).Info("Disk info collected")

	// Convert disk info for unified structure
	disks := make([]map[string]interface{}, len(diskInfo.Disks))
	for i, disk := range diskInfo.Disks {
		disks[i] = map[string]interface{}{
			"path":         disk.Path,
			"total_gb":     float64(disk.Total) / 1024 / 1024 / 1024,
			"used_gb":      float64(disk.Used) / 1024 / 1024 / 1024,
			"free_gb":      float64(disk.Free) / 1024 / 1024 / 1024,
			"used_percent": disk.UsedPercent,
			"filesystem":   disk.Filesystem,
		}
	}

	// Add disk metrics to unified structure
	if len(disks) > 0 {
		// Use first disk for backward compatibility
		metrics["disk"] = disks[0]["used_percent"]
		metrics["disk_details"] = disks
		a.logger.WithField("disk_details", disks).Debug("Disk details added to metrics")
	}
}

// collectNetworkMetrics collects network metrics and adds them to the unified structure
func (a *Agent) collectNetworkMetrics(metrics map[string]interface{}) {
	if a.networkMetrics == nil {
		a.logger.Error("Network metrics is nil - network metrics collection skipped")
		return
	}

	networkInfo, err := a.networkMetrics.GetNetworkInfo()
	if err != nil {
		a.logger.WithError(err).Error("Failed to get network info")
		return
	}

	a.logger.WithField("network_info", networkInfo).Info("Network info collected")

	// Add network metrics to unified structure
	metrics["network"] = networkInfo.TotalRxMbps + networkInfo.TotalTxMbps // For backward compatibility
	networkDetails := map[string]interface{}{
		"interfaces":    networkInfo.Interfaces,
		"total_rx_mbps": networkInfo.TotalRxMbps,
		"total_tx_mbps": networkInfo.TotalTxMbps,
	}
	metrics["network_details"] = networkDetails
	a.logger.WithField("network_details", networkDetails).Debug("Network details added to metrics")
}

// collectTemperatureMetrics collects temperature metrics and adds them to the unified structure
func (a *Agent) collectTemperatureMetrics(metrics map[string]interface{}) {
	if a.temperatureMetrics == nil {
		a.logger.Error("Temperature metrics is nil - temperature metrics collection skipped")
		return
	}

	tempInfo, err := a.temperatureMetrics.GetTemperatureInfo()
	if err != nil {
		a.logger.WithError(err).Error("Failed to get temperature info")
		return
	}

	a.logger.WithField("temperature_info", tempInfo).Info("Temperature info collected")

	// Add temperature metrics to unified structure
	// Use highest temperature for backward compatibility
	metrics["temperature"] = tempInfo.HighestTemperature
	temperatureDetails := map[string]interface{}{
		"cpu_temperature":      tempInfo.CPUTemperature,
		"gpu_temperature":      tempInfo.GPUTemperature,
		"system_temperature":   tempInfo.SystemTemperature,
		"storage_temperatures": tempInfo.StorageTemperatures,
		"highest_temperature":  tempInfo.HighestTemperature,
		"temperature_unit":     tempInfo.TemperatureUnit,
	}
	metrics["temperature_details"] = temperatureDetails
	a.logger.WithField("temperature_details", temperatureDetails).Debug("Temperature details added to metrics")
}

// collectSystemMetrics collects system metrics and adds them to the unified structure
func (a *Agent) collectSystemMetrics(metrics map[string]interface{}) {
	if a.systemMonitor == nil {
		a.logger.Error("System monitor is nil - system details collection skipped")
		return
	}

	systemDetails, err := a.systemMonitor.GetSystemDetails()
	if err != nil {
		a.logger.WithError(err).Error("Failed to get system details")
		return
	}

	a.logger.WithField("system_details", systemDetails).Info("System details collected")

	// Add system metrics to unified structure
	systemDetailsMap := map[string]interface{}{
		"uptime_seconds":     systemDetails.UptimeSeconds,
		"uptime_human":       systemDetails.UptimeHuman,
		"processes_total":    systemDetails.ProcessesTotal,
		"processes_running":  systemDetails.ProcessesRunning,
		"processes_sleeping": systemDetails.ProcessesSleeping,
	}
	metrics["system_details"] = systemDetailsMap
	a.logger.WithField("system_details", systemDetailsMap).Debug("System details added to metrics")
}

// sendUnifiedMetrics creates and sends the unified metrics message
func (a *Agent) sendUnifiedMetrics(metrics map[string]interface{}) {
	// Get system details and add to metrics
	if a.systemMonitor != nil {
		if details, err := a.systemMonitor.GetSystemDetails(); err == nil {
			// Update system_details with full system info
			if systemDetails, ok := metrics["system_details"].(map[string]interface{}); ok {
				systemDetails["hostname"] = details.Hostname
				systemDetails["os"] = details.OS
				systemDetails["kernel"] = details.Kernel
				systemDetails["architecture"] = details.Architecture
				systemDetails["uptime_seconds"] = details.UptimeSeconds
				metrics["system_details"] = systemDetails
			}
		}
	}

	// Create unified metric message with correct structure for API
	unifiedMetric := &types.Metric{
		ServerID:   a.config.Server.ServerID,
		ServerKey:  a.config.Server.SecretKey,
		ServerName: a.config.Server.Name,
		Type:       "metrics",
		Version:    "1.0",
		Value:      nil,
		Timestamp:  time.Now(),
		Data: map[string]interface{}{
			"metrics": metrics,
		},
	}

	// Send unified metrics via HTTP
	if err := a.httpPublisher.Publish(a.ctx, unifiedMetric); err != nil {
		a.logger.WithError(err).Error("Failed to send unified metrics via HTTP")
	} else {
		a.logger.WithField("metrics_count", len(metrics)).Info("Unified metrics sent successfully via HTTP")
	}
}
