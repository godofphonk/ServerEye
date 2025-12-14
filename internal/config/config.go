package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AgentConfig конфигурация агента
type AgentConfig struct {
	Server  ServerConfig  `yaml:"server"`
	API     APIConfig     `yaml:"api,omitempty"`
	Kafka   KafkaConfig   `yaml:"kafka,omitempty"`
	Metrics MetricsConfig `yaml:"metrics"`
	Logging LoggingConfig `yaml:"logging"`
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	SecretKey   string `yaml:"secret_key"`
}

// APIConfig конфигурация HTTP API
type APIConfig struct {
	BaseURL string `yaml:"base_url"`
	Timeout string `yaml:"timeout,omitempty"`
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

// KafkaConfig конфигурация Kafka
type KafkaConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Brokers      []string `yaml:"brokers"`
	TopicPrefix  string   `yaml:"topic_prefix"`
	Compression  string   `yaml:"compression"`
	MaxAttempts  int      `yaml:"max_attempts"`
	BatchSize    int      `yaml:"batch_size"`
	RequiredAcks int      `yaml:"required_acks"`
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

	// Debug: Print loaded Kafka brokers
	fmt.Printf("DEBUG: Loaded Kafka brokers: %v\n", config.Kafka.Brokers)

	// Валидация конфигурации
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("некорректная конфигурация: %v", err)
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

	// Проверяем, что есть либо Kafka, либо HTTP API конфигурация
	if !c.Kafka.Enabled && c.API.BaseURL == "" {
		return fmt.Errorf("должен быть включен Kafka либо указан базовый URL API")
	}

	return nil
}
