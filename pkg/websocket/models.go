package websocket

import (
	"time"
)

// CPUUsageInfo represents detailed CPU usage statistics
type CPUUsageInfo struct {
	UsageTotal  float64          `json:"usage_total"`
	UsageUser   float64          `json:"usage_user"`
	UsageSystem float64          `json:"usage_system"`
	UsageIdle   float64          `json:"usage_idle"`
	LoadAverage *LoadAverageInfo `json:"load_average"`
	Cores       int              `json:"cores"`
	Frequency   float64          `json:"frequency"`
}

// LoadAverageInfo represents system load averages
type LoadAverageInfo struct {
	Load1Min  float64 `json:"load_1min"`
	Load5Min  float64 `json:"load_5min"`
	Load15Min float64 `json:"load_15min"`
}

// Message represents WebSocket message structure
type Message struct {
	Type      string                 `json:"type"`
	ServerID  string                 `json:"server_id,omitempty"`
	ServerKey string                 `json:"server_key,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp int64                  `json:"timestamp,omitempty"`
}

// Message types
const (
	MessageTypeAuth        = "auth"
	MessageTypeAuthSuccess = "auth_success"
	MessageTypeError       = "error"
	MessageTypeMetrics     = "metrics"
	MessageTypeHeartbeat   = "heartbeat"
	MessageTypeCommand     = "command"
	MessageTypeSubscribe   = "subscribe"
)

// AuthMessage represents authentication message
type AuthMessage struct {
	Type      string `json:"type"`
	ServerID  string `json:"server_id"`
	ServerKey string `json:"server_key"`
}

// AuthSuccessMessage represents successful authentication response
type AuthSuccessMessage struct {
	Type     string `json:"type"`
	ServerID string `json:"server_id"`
}

// ErrorMessage represents error message
type ErrorMessage struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}

// MetricsData represents metrics payload
type MetricsData struct {
	ServerID string        `json:"server_id"`
	Metrics  ServerMetrics `json:"metrics"`
	System   SystemInfo    `json:"system"`
}

// ServerMetrics represents server performance metrics
type ServerMetrics struct {
	CPU            float64       `json:"cpu"`                       // CPU usage percentage (0-100)
	Memory         float64       `json:"memory"`                    // Memory usage percentage (0-100)
	Disk           float64       `json:"disk"`                      // Disk usage percentage (0-100)
	Network        float64       `json:"network"`                   // Network usage in MB/s
	CPUUsage       *CPUUsageInfo `json:"cpu_usage,omitempty"`       // Detailed CPU usage statistics
	MemoryDetails  interface{}   `json:"memory_details,omitempty"`  // Detailed memory statistics
	DiskDetails    interface{}   `json:"disk_details,omitempty"`    // Detailed disk statistics
	NetworkDetails interface{}   `json:"network_details,omitempty"` // Detailed network statistics
	Time           time.Time     `json:"time"`                      // Timestamp when metrics were collected
}

// SystemInfo represents system information
type SystemInfo struct {
	OS           string `json:"os"`           // Operating system name
	Architecture string `json:"architecture"` // System architecture
	Kernel       string `json:"kernel"`       // Kernel version
	Uptime       int64  `json:"uptime"`       // System uptime in seconds
	Hostname     string `json:"hostname"`     // Server hostname
}

// HeartbeatMessage represents heartbeat message
type HeartbeatMessage struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
}

// CommandMessage represents command message from server
type CommandMessage struct {
	Type      string                 `json:"type"`
	RequestID string                 `json:"request_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
}

// CommandResponse represents response to command
type CommandResponse struct {
	RequestID string      `json:"request_id"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp"`
}
