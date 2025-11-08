package publisher

import (
	"context"
	"time"
)

// Metric представляет метрику для отправки
type Metric struct {
	ServerID   string                 `json:"server_id"`
	ServerKey  string                 `json:"server_key"`
	ServerName string                 `json:"server_name,omitempty"`
	Type       string                 `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	Value      interface{}            `json:"value"`
	Tags       map[string]string      `json:"tags,omitempty"`
	Version    string                 `json:"version"`
}

// Publisher определяет интерфейс для публикации метрик
type Publisher interface {
	// Publish отправляет одну метрику
	Publish(ctx context.Context, metric *Metric) error

	// PublishBatch отправляет пакет метрик (optional, для оптимизации)
	PublishBatch(ctx context.Context, metrics []*Metric) error

	// Close закрывает соединение
	Close() error

	// Name возвращает имя publisher (для логирования)
	Name() string
}
