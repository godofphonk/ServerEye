package protocol

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MessageType defines the type of message
type MessageType string

const (
	// Commands from bot to agent
	TypeGetCPUTemp       MessageType = "get_cpu_temp"
	TypeGetSystemInfo    MessageType = "get_system_info"
	TypeGetContainers    MessageType = "get_containers"
	TypeStartContainer   MessageType = "start_container"
	TypeStopContainer    MessageType = "stop_container"
	TypeRestartContainer MessageType = "restart_container"
	TypePing             MessageType = "ping"

	// Responses from agent to bot
	TypeCPUTempResponse       MessageType = "cpu_temp_response"
	TypeSystemInfoResponse    MessageType = "system_info_response"
	TypeContainersResponse    MessageType = "containers_response"
	TypeContainerActionResponse MessageType = "container_action_response"
	TypePong                  MessageType = "pong"
	TypeErrorResponse         MessageType = "error_response"
)

// Message represents a base protocol message
type Message struct {
	ID        string      `json:"id"`
	Type      MessageType `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Version   string      `json:"version"`
	Payload   interface{} `json:"payload"`
}

// NewMessage creates a new message
func NewMessage(msgType MessageType, payload interface{}) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Type:      msgType,
		Timestamp: time.Now(),
		Version:   "1.0",
		Payload:   payload,
	}
}

// ToJSON serializes message to JSON
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON deserializes message from JSON
func FromJSON(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

// CPUTempPayload represents CPU temperature data
type CPUTempPayload struct {
	Temperature float64 `json:"temperature"`
	Unit        string  `json:"unit"`
	Sensor      string  `json:"sensor"`
}

// SystemInfoPayload represents system information data
type SystemInfoPayload struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Uptime   string `json:"uptime"`
}

// ErrorPayload represents error information
type ErrorPayload struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// PongPayload represents pong response data
type PongPayload struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
}

// ContainerInfo represents Docker container information
type ContainerInfo struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Image  string            `json:"image"`
	Status string            `json:"status"`
	State  string            `json:"state"`
	Ports  []string          `json:"ports"`
	Labels map[string]string `json:"labels,omitempty"`
}

// ContainersPayload represents Docker containers data
type ContainersPayload struct {
	Containers []ContainerInfo `json:"containers"`
	Total      int             `json:"total"`
}

// ContainerActionPayload represents container action request
type ContainerActionPayload struct {
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	Action        string `json:"action"` // "start", "stop", "restart"
}

// ContainerActionResponse represents container action result
type ContainerActionResponse struct {
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	Action        string `json:"action"`
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	NewState      string `json:"new_state,omitempty"`
}

// Error codes
const (
	ErrorSensorNotFound     = "SENSOR_NOT_FOUND"
	ErrorPermissionDenied   = "PERMISSION_DENIED"
	ErrorCommandTimeout     = "COMMAND_TIMEOUT"
	ErrorInvalidCommand     = "INVALID_COMMAND"
	ErrorContainerNotFound  = "CONTAINER_NOT_FOUND"
	ErrorContainerAction    = "CONTAINER_ACTION_FAILED"
	ErrorDockerUnavailable  = "DOCKER_UNAVAILABLE"
)
