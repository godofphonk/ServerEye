package agent

import (
	"time"

	"github.com/godofphonk/ServerEye/pkg/protocol"
	"github.com/godofphonk/ServerEye/pkg/types"
)

// collectAndSendMetrics запускает цикл сбора и отправки метрик
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
	metrics := a.collectMetrics()
	if metrics == nil {
		a.logger.Error("Failed to collect metrics")
		return
	}

	// Send via WebSocket or HTTP
	a.sendMetrics(metrics)
}

// collectMetrics collects all metrics
func (a *Agent) collectMetrics() *protocol.Metrics {
	a.logger.Info("Collecting metrics...")

	metrics := &protocol.Metrics{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Collect CPU metrics
	if a.cpuMetrics != nil {
		cpuUsage, err := a.cpuMetrics.GetDetailedUsage()
		if err == nil {
			metrics.CPUUsage = protocol.CPUUsage{
				UsageTotal:   cpuUsage.UsageTotal,
				UsageUser:    cpuUsage.UsageUser,
				UsageSystem:  cpuUsage.UsageSystem,
				UsageIdle:    cpuUsage.UsageIdle,
				FrequencyMHz: cpuUsage.Frequency,
			}

			if cpuUsage.LoadAverage != nil {
				metrics.CPUUsage.LoadAverage = protocol.LoadAverage{
					Load1Min:  cpuUsage.LoadAverage.Load1Min,
					Load5Min:  cpuUsage.LoadAverage.Load5Min,
					Load15Min: cpuUsage.LoadAverage.Load15Min,
				}
			}
		}
	}

	// Collect Memory metrics
	if a.systemMonitor != nil {
		memInfo, err := a.systemMonitor.GetMemoryInfo()
		if err == nil {
			metrics.Memory = protocol.Memory{
				TotalGB:     float64(memInfo.Total) / 1024 / 1024 / 1024,
				UsedGB:      float64(memInfo.Used) / 1024 / 1024 / 1024,
				AvailableGB: float64(memInfo.Available) / 1024 / 1024 / 1024,
				FreeGB:      float64(memInfo.Free) / 1024 / 1024 / 1024,
				BuffersGB:   float64(memInfo.Buffers) / 1024 / 1024 / 1024,
				CachedGB:    float64(memInfo.Cached) / 1024 / 1024 / 1024,
				UsedPercent: memInfo.UsedPercent,
			}
		}

		// Collect Disk metrics
		diskInfo, err := a.systemMonitor.GetDiskInfo()
		if err == nil {
			for _, disk := range diskInfo.Disks {
				// Skip special filesystems
				if disk.Path == "/boot/efi" || disk.Path == "/sys/firmware/efi/efivars" {
					continue
				}

				diskV2 := protocol.Disk{
					MountPoint:  disk.Path,
					DeviceName:  disk.Filesystem,
					UsedGB:      int(disk.Used / 1024 / 1024 / 1024),
					FreeGB:      int(disk.Free / 1024 / 1024 / 1024),
					UsedPercent: disk.UsedPercent,
				}
				metrics.Disks = append(metrics.Disks, diskV2)
			}
		}

		// Collect Network metrics
		if a.networkMetrics != nil {
			netInfo, err := a.networkMetrics.GetNetworkInfo()
			if err == nil {
				metrics.Network.TotalRxMbps = netInfo.TotalRxMbps
				metrics.Network.TotalTxMbps = netInfo.TotalTxMbps

				for _, iface := range netInfo.Interfaces {
					// Skip loopback
					if iface.Name == "lo" {
						continue
					}

					ifaceV2 := protocol.NetworkInterface{
						Name:        iface.Name,
						RxBytes:     int64(iface.RxBytes),
						TxBytes:     int64(iface.TxBytes),
						RxSpeedMbps: iface.RxSpeedMbps,
						TxSpeedMbps: iface.TxSpeedMbps,
						Status:      iface.Status,
					}
					metrics.Network.Interfaces = append(metrics.Network.Interfaces, ifaceV2)
				}
			}
		}
	}

	// Collect Temperature metrics
	if a.temperatureMetrics != nil {
		tempInfo, err := a.temperatureMetrics.GetTemperatureInfo()
		if err == nil {
			metrics.Temperature.CPU = tempInfo.CPUTemperature
			metrics.Temperature.GPU = tempInfo.GPUTemperature
			metrics.Temperature.Highest = tempInfo.HighestTemperature

			for _, storage := range tempInfo.StorageTemperatures {
				storageTemp := protocol.StorageTemp{
					Device:      storage.Device,
					Temperature: storage.Temperature,
				}
				metrics.Temperature.Storage = append(metrics.Temperature.Storage, storageTemp)
			}
		}
	}

	// Collect System metrics
	if a.systemMonitor != nil {
		sysDetails, err := a.systemMonitor.GetSystemDetails()
		if err == nil {
			metrics.System = protocol.System{
				ProcessesTotal:    sysDetails.ProcessesTotal,
				ProcessesRunning:  sysDetails.ProcessesRunning,
				ProcessesSleeping: sysDetails.ProcessesSleeping,
				UptimeSeconds:     sysDetails.UptimeSeconds,
			}
		}
	}

	a.logger.WithField("metrics", metrics).Debug("Metrics collected")
	return metrics
}

// sendMetrics sends metrics
func (a *Agent) sendMetrics(metrics *protocol.Metrics) {
	// Create metric message
	metric := &types.Metric{
		Type:      "metrics",
		ServerID:  a.config.Server.ServerID,
		ServerKey: a.config.Server.SecretKey,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"metrics": map[string]interface{}{
				"cpu_usage":   metrics.CPUUsage,
				"memory":      metrics.Memory,
				"disks":       metrics.Disks,
				"network":     metrics.Network,
				"temperature": metrics.Temperature,
				"system":      metrics.System,
				"timestamp":   metrics.Timestamp,
			},
		},
	}

	// Send via WebSocket or HTTP
	if a.useWebSocket && a.wsPublisher != nil {
		a.logger.Debug("Sending metrics via WebSocket")
		if err := a.wsPublisher.Publish(a.ctx, metric); err != nil {
			a.logger.WithError(err).Debug("Failed to send metrics via WebSocket, falling back to HTTP")
			if err := a.httpPublisher.Publish(a.ctx, metric); err != nil {
				a.logger.WithError(err).Error("Failed to send metrics via HTTP")
			} else {
				a.logger.Info("Metrics sent successfully via HTTP")
			}
		} else {
			a.logger.Debug("Metrics sent successfully via WebSocket")
		}
	} else {
		// Use HTTP
		if err := a.httpPublisher.Publish(a.ctx, metric); err != nil {
			a.logger.WithError(err).Error("Failed to send metrics via HTTP")
		} else {
			a.logger.Info("Metrics sent successfully via HTTP")
		}
	}
}
