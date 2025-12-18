package kafka

import "time"

// MetricMessage represents a system metric sent by agent
type MetricMessage struct {
	AgentID   string                 `json:"agent_id"`
	Hostname  string                 `json:"hostname"`
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// CommandMessage represents a command sent to agent
type CommandMessage struct {
	AgentID string                 `json:"agent_id"`
	Command string                 `json:"command"`
	Params  map[string]interface{} `json:"params"`
}

// ResponseMessage represents agent response to command
type ResponseMessage struct {
	AgentID   string                 `json:"agent_id"`
	CommandID string                 `json:"command_id"`
	Success   bool                   `json:"success"`
	Data      map[string]interface{} `json:"data"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Topic names
const (
	MetricsTopic   = "servereye.metrics"
	CommandsTopic  = "servereye.commands"
	ResponsesTopic = "servereye.responses"
)
