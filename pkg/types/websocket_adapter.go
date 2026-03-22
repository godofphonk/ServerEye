package types

import (
	"time"

	"github.com/godofphonk/ServerEye/pkg/protocol"
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
	// Handle heartbeat separately - no metrics data
	if metric.Type == "heartbeat" {
		return websocket.Message{
			Type:      websocket.MessageTypeHeartbeat,
			ServerID:  metric.ServerID,
			Data:      map[string]interface{}{},
			Timestamp: time.Now().Unix(),
		}
	}

	// For new metrics format, send data directly without conversion
	if metric.Type == "metrics" && metric.Data != nil {
		if metrics, ok := metric.Data["metrics"]; ok {
			return websocket.Message{
				Type:     websocket.MessageTypeMetrics,
				ServerID: metric.ServerID,
				Data: map[string]interface{}{
					"metrics": metrics,
				},
				Timestamp: time.Now().Unix(),
			}
		}
	}

	// Fallback to old format for other metric types (not heartbeat)
	serverMetrics := websocket.ServerMetrics{
		Time: metric.Timestamp,
	}
	systemInfo := a.extractSystemInfo(metric)

	// Handle different metric types
	a.processMetricType(metric, &serverMetrics)

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
			"metrics": metricsData.Metrics,
			"system":  metricsData.System,
		},
		Timestamp: time.Now().Unix(),
	}
}

// extractSystemInfo extracts system information from metric data
func (a *WebSocketAdapter) extractSystemInfo(metric *Metric) websocket.SystemInfo {
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

	return systemInfo
}

// processMetricType processes metric based on its type
func (a *WebSocketAdapter) processMetricType(metric *Metric, serverMetrics *websocket.ServerMetrics) {
	switch metric.Type {
	case "cpu_temperature", "cpu_usage":
		if value, ok := metric.Value.(float64); ok {
			serverMetrics.CPU = value
		}
	case "memory_usage":
		if value, ok := metric.Value.(float64); ok {
			serverMetrics.Memory = value
		}
	case "disk_usage":
		if value, ok := metric.Value.(float64); ok {
			serverMetrics.Disk = value
		}
	case "network_usage":
		if value, ok := metric.Value.(float64); ok {
			serverMetrics.Network = value
		}
	case "metrics":
		a.processUnifiedMetrics(metric, serverMetrics)
	}
}

// processUnifiedMetrics processes unified metrics structure
func (a *WebSocketAdapter) processUnifiedMetrics(metric *Metric, serverMetrics *websocket.ServerMetrics) {
	if metric.Data == nil {
		return
	}

	metrics, ok := metric.Data["metrics"].(map[string]interface{})
	if !ok {
		return
	}

	// Extract basic metrics
	a.extractBasicMetrics(metrics, serverMetrics)

	// Extract detailed metrics
	a.extractCPUUsageDetails(metrics, serverMetrics)
	a.extractOtherDetails(metrics, serverMetrics)
}

// extractBasicMetrics extracts basic metrics values
func (a *WebSocketAdapter) extractBasicMetrics(metrics map[string]interface{}, serverMetrics *websocket.ServerMetrics) {
	// Handle CPU - support both V2 struct and map formats
	if cpuUsageStruct, ok := metrics["cpu_usage"].(protocol.CPUUsage); ok {
		serverMetrics.CPU = cpuUsageStruct.UsageTotal
	} else if cpuUsage, ok := metrics["cpu_usage"].(map[string]interface{}); ok {
		if usageTotal, ok := cpuUsage["usage_total"].(float64); ok {
			serverMetrics.CPU = usageTotal
		}
	}

	// Handle Memory - support both V2 struct and map formats
	if memoryStruct, ok := metrics["memory"].(protocol.Memory); ok {
		serverMetrics.Memory = memoryStruct.UsedPercent
	} else if memory, ok := metrics["memory"].(map[string]interface{}); ok {
		if usedPercent, ok := memory["used_percent"].(float64); ok {
			serverMetrics.Memory = usedPercent
		}
	}

	// Handle Disk - support both V2 struct and map formats
	if disksStruct, ok := metrics["disks"].([]protocol.Disk); ok && len(disksStruct) > 0 {
		serverMetrics.Disk = disksStruct[0].UsedPercent
	} else if disks, ok := metrics["disks"].([]interface{}); ok && len(disks) > 0 {
		if firstDisk, ok := disks[0].(map[string]interface{}); ok {
			if usedPercent, ok := firstDisk["used_percent"].(float64); ok {
				serverMetrics.Disk = usedPercent
			}
		}
	}

	// Handle Network - support both V2 struct and map formats
	if networkStruct, ok := metrics["network"].(protocol.Network); ok {
		// Calculate total network from all interfaces
		var totalRx, totalTx float64
		for _, iface := range networkStruct.Interfaces {
			totalRx += float64(iface.RxBytes) / 1024 / 1024 // Convert to MB
			totalTx += float64(iface.TxBytes) / 1024 / 1024
		}
		serverMetrics.Network = totalRx + totalTx
	} else if network, ok := metrics["network"].(map[string]interface{}); ok {
		if totalRx, ok := network["total_rx_mbps"].(float64); ok {
			if totalTx, ok := network["total_tx_mbps"].(float64); ok {
				serverMetrics.Network = totalRx + totalTx
			}
		}
	}

	// Handle Temperature - support V2 struct format
	if tempStruct, ok := metrics["temperature"].(protocol.Temperature); ok {
		serverMetrics.Temperature = tempStruct.Highest
	}

	// Fallback to old format for backward compatibility
	if cpu, ok := metrics["cpu"].(float64); ok && serverMetrics.CPU == 0 {
		serverMetrics.CPU = cpu
	}
	if memory, ok := metrics["memory"].(float64); ok && serverMetrics.Memory == 0 {
		serverMetrics.Memory = memory
	}
	if disk, ok := metrics["disk"].(float64); ok && serverMetrics.Disk == 0 {
		serverMetrics.Disk = disk
	}
	if network, ok := metrics["network"].(float64); ok && serverMetrics.Network == 0 {
		serverMetrics.Network = network
	}
}

// extractCPUUsageDetails extracts CPU usage details
func (a *WebSocketAdapter) extractCPUUsageDetails(metrics map[string]interface{}, serverMetrics *websocket.ServerMetrics) {
	cpuUsage, ok := metrics["cpu_usage"].(map[string]interface{})
	if !ok {
		return
	}

	usageTotal := getFloat64(cpuUsage, "usage_total")
	serverMetrics.CPU = usageTotal
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

// extractOtherDetails extracts other detailed metrics
func (a *WebSocketAdapter) extractOtherDetails(metrics map[string]interface{}, serverMetrics *websocket.ServerMetrics) {
	details := map[string]interface{}{
		"memory_details":      &serverMetrics.MemoryDetails,
		"disk_details":        &serverMetrics.DiskDetails,
		"network_details":     &serverMetrics.NetworkDetails,
		"temperature_details": &serverMetrics.TemperatureDetails,
		"system_details":      &serverMetrics.SystemDetails,
	}

	for detailKey, target := range details {
		if detail, ok := metrics[detailKey]; ok {
			if ptr, ok := target.(*interface{}); ok {
				*ptr = detail
			}
		}
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
