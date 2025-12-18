package types

import (
	"context"
	"time"
)

// Metric represents a system metric collected by agent
type Metric struct {
	ServerID   string                 `json:"server_id"`
	ServerKey  string                 `json:"server_key"`
	ServerName string                 `json:"server_name,omitempty"`
	Type       string                 `json:"type"`
	Version    string                 `json:"version"`
	Data       map[string]interface{} `json:"data"`
	Value      interface{}            `json:"value"` // For backward compatibility
	Timestamp  time.Time              `json:"timestamp"`
	Tags       map[string]string      `json:"tags,omitempty"`
}

// Publisher interface for sending metrics
type Publisher interface {
	Publish(ctx context.Context, metric *Metric) error
	PublishBatch(ctx context.Context, metrics []*Metric) error
	Close() error
	Name() string
}
