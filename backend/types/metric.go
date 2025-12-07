package types

import (
	"context"
	"time"
)

// Metric represents a single metric data point
type Metric struct {
	ServerID   string            `json:"server_id"`
	ServerKey  string            `json:"server_key"`
	Type       string            `json:"type"`
	Value      float64           `json:"value"`
	Tags       map[string]string `json:"tags,omitempty"`
	Version    string            `json:"version"`
	Timestamp  time.Time         `json:"timestamp"`
}

// Publisher interface for sending metrics
type Publisher interface {
	Publish(ctx context.Context, metric *Metric) error
	PublishBatch(ctx context.Context, metrics []*Metric) error
	Close() error
	Name() string
}
