package types

import (
	"time"

	"github.com/godofphonk/ServerEye/pkg/websocket"
)

// WebSocketAdapter converts Metric to WebSocket message format
type WebSocketAdapter struct{}

// NewWebSocketAdapter creates new adapter
func NewWebSocketAdapter() *WebSocketAdapter {
	return &WebSocketAdapter{}
}

// ToWebSocketMessage converts Metric to WebSocket metrics message
func (a *WebSocketAdapter) ToWebSocketMessage(metric *Metric) websocket.Message {
	serverMetrics := websocket.ServerMetrics{
		Time: time.Now(),
	}
	systemInfo := websocket.SystemInfo{
		Hostname: metric.ServerName,
	}

	// Extract system info from data if available
	if metric.Data != nil {
		if os, ok := metric.Data["os"].(string); ok {
			systemInfo.OS = os
		}
		if arch, ok := metric.Data["architecture"].(string); ok {
			systemInfo.Architecture = arch
		}
		if kernel, ok := metric.Data["kernel"].(string); ok {
			systemInfo.Kernel = kernel
		}
		if uptime, ok := metric.Data["uptime"].(int64); ok {
			systemInfo.Uptime = uptime
		}
	}

	// Create server metrics based on metric type
	serverMetrics.Time = metric.Timestamp

	// Handle different metric types
	switch metric.Type {
	case "cpu_temperature":
		if temp, ok := metric.Value.(float64); ok {
			serverMetrics.CPU = temp
		}
	case "cpu_usage":
		if usage, ok := metric.Value.(float64); ok {
			serverMetrics.CPU = usage
		}
	case "memory_usage":
		if usage, ok := metric.Value.(float64); ok {
			serverMetrics.Memory = usage
		}
	case "disk_usage":
		if usage, ok := metric.Value.(float64); ok {
			serverMetrics.Disk = usage
		}
	case "network_usage":
		if usage, ok := metric.Value.(float64); ok {
			serverMetrics.Network = usage
		}
	case "metrics":
		// Handle unified metrics structure
		if metric.Data != nil {
			if metrics, ok := metric.Data["metrics"].(map[string]interface{}); ok {
				// Extract CPU metrics
				if cpu, ok := metrics["cpu"].(float64); ok {
					serverMetrics.CPU = cpu
				}
				if memory, ok := metrics["memory"].(float64); ok {
					serverMetrics.Memory = memory
				}
				if disk, ok := metrics["disk"].(float64); ok {
					serverMetrics.Disk = disk
				}
				if network, ok := metrics["network"].(float64); ok {
					serverMetrics.Network = network
				}
				// Extract CPU usage details
				if cpuUsage, ok := metrics["cpu_usage"].(map[string]interface{}); ok {
					usageTotal := getFloat64(cpuUsage, "usage_total")
					serverMetrics.CPU = usageTotal // Set CPU from cpu_usage.usage_total
					serverMetrics.CPUUsage = &websocket.CPUUsageInfo{
						UsageTotal:  usageTotal,
						UsageUser:   getFloat64(cpuUsage, "usage_user"),
						UsageSystem: getFloat64(cpuUsage, "usage_system"),
						UsageIdle:   getFloat64(cpuUsage, "usage_idle"),
						Cores:       getInt(cpuUsage, "cores"),
						Frequency:   getFloat64(cpuUsage, "frequency"),
					}
					// Extract load average if available
					if loadAvg, ok := cpuUsage["load_average"].(map[string]interface{}); ok {
						serverMetrics.CPUUsage.LoadAverage = &websocket.LoadAverageInfo{
							Load1Min:  getFloat64(loadAvg, "load_1min"),
							Load5Min:  getFloat64(loadAvg, "load_5min"),
							Load15Min: getFloat64(loadAvg, "load_15min"),
						}
					}
				}
			}
		}
	}

	// Create metrics data
	metricsData := websocket.MetricsData{
		ServerID: metric.ServerID,
		Metrics:  serverMetrics,
		System:   systemInfo,
	}

	return websocket.Message{
		Type:     websocket.MessageTypeMetrics,
		ServerID: metric.ServerID,
		Data: map[string]interface{}{
			"server_id": metricsData.ServerID,
			"metrics":   metricsData.Metrics,
			"system":    metricsData.System,
		},
		Timestamp: time.Now().Unix(),
	}
}

// ToWebSocketMetrics converts multiple metrics to WebSocket format
func (a *WebSocketAdapter) ToWebSocketMetrics(metrics []*Metric) []websocket.Message {
	messages := make([]websocket.Message, 0, len(metrics))

	for _, metric := range metrics {
		messages = append(messages, a.ToWebSocketMessage(metric))
	}

	return messages
}

// BatchMetricsByType groups metrics by type for efficient sending
func (a *WebSocketAdapter) BatchMetricsByType(metrics []*Metric) map[string][]*Metric {
	batches := make(map[string][]*Metric)

	for _, metric := range metrics {
		batches[metric.Type] = append(batches[metric.Type], metric)
	}

	return batches
}

// CreateSystemInfoMetric creates a system info metric
func (a *WebSocketAdapter) CreateSystemInfoMetric(serverID, serverKey, serverName, os, arch, kernel string, uptime int64) *Metric {
	return &Metric{
		ServerID:   serverID,
		ServerKey:  serverKey,
		ServerName: serverName,
		Type:       "system_info",
		Version:    "1.0",
		Data: map[string]interface{}{
			"os":           os,
			"architecture": arch,
			"kernel":       kernel,
			"uptime":       uptime,
			"hostname":     serverName,
		},
		Value:     nil,
		Timestamp: time.Now(),
		Tags:      make(map[string]string),
	}
}

// CreateContainerMetrics creates container metrics message
func (a *WebSocketAdapter) CreateContainerMetrics(serverID string, containers []map[string]interface{}) websocket.Message {
	return websocket.Message{
		Type:     "containers",
		ServerID: serverID,
		Data: map[string]interface{}{
			"containers": containers,
		},
		Timestamp: time.Now().Unix(),
	}
}

// Helper functions for type conversion
func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
	}
	return 0
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		if i, ok := val.(int); ok {
			return i
		}
		if f, ok := val.(float64); ok {
			return int(f)
		}
	}
	return 0
}
