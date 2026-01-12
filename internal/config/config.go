package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AgentConfig конфигурация агента
type AgentConfig struct {
	Server    ServerConfig    `yaml:"server"`
	API       APIConfig       `yaml:"api,omitempty"`
	WebSocket WebSocketConfig `yaml:"websocket,omitempty"`
	Metrics   MetricsConfig   `yaml:"metrics"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	SecretKey   string `yaml:"secret_key"`
	ServerID    string `yaml:"server_id,omitempty"`
}

// APIConfig конфигурация HTTP API
type APIConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Timeout string `yaml:"timeout,omitempty"`
}

// WebSocketConfig конфигурация WebSocket
type WebSocketConfig struct {
	Enabled              bool   `yaml:"enabled"`
	URL                  string `yaml:"url"`
	ReconnectInterval    string `yaml:"reconnect_interval,omitempty"`
	MaxReconnectAttempts int    `yaml:"max_reconnect_attempts,omitempty"`
	PingInterval         string `yaml:"ping_interval,omitempty"`
	WriteTimeout         string `yaml:"write_timeout,omitempty"`
	ReadTimeout          string `yaml:"read_timeout,omitempty"`
	HandshakeTimeout     string `yaml:"handshake_timeout,omitempty"`
	BufferSize           int    `yaml:"buffer_size,omitempty"`
	EnableCompression    bool   `yaml:"enable_compression,omitempty"`
	MetricBufferSize     int    `yaml:"metric_buffer_size,omitempty"`
	MetricBufferFlush    string `yaml:"metric_buffer_flush,omitempty"`
	CommandQueueSize     int    `yaml:"command_queue_size,omitempty"`
	CommandTimeout       string `yaml:"command_timeout,omitempty"`
}

// MetricsConfig конфигурация метрик
type MetricsConfig struct {
	CPUUsage       bool   `yaml:"cpu_usage"`
	MemoryUsage    bool   `yaml:"memory_usage"`
	DiskUsage      bool   `yaml:"disk_usage"`
	CPUTemperature bool   `yaml:"cpu_temperature"`
	Interval       string `yaml:"interval"`
}

// LoggingConfig конфигурация логирования
type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// LoadAgentConfig загружает конфигурацию агента
func LoadAgentConfig(filepath string) (*AgentConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл конфигурации: %v", err)
	}

	var config AgentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("не удалось парсить конфигурацию: %v", err)
	}

	// Валидация конфигурации
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("некорректная конфигурация: %v", err)
	}

	// Устанавливаем дефолтные значения
	if config.Metrics.Interval == "" {
		config.Metrics.Interval = "30s"
	}

	return &config, nil
}

// validate валидирует конфигурацию агента
func (c *AgentConfig) validate() error {
	if c.Server.Name == "" {
		return fmt.Errorf("имя сервера не может быть пустым")
	}
	if c.Server.SecretKey == "" {
		return fmt.Errorf("секретный ключ не может быть пустым")
	}

	// Проверяем, что указан базовый URL API
	if c.API.BaseURL == "" {
		return fmt.Errorf("должен быть указан базовый URL API")
	}

	return nil
}
