package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AgentConfig конфигурация агента
type AgentConfig struct {
	Server  ServerConfig  `yaml:"server"`
	Redis   RedisConfig   `yaml:"redis,omitempty"`
	API     APIConfig     `yaml:"api,omitempty"`
	Metrics MetricsConfig `yaml:"metrics"`
	Logging LoggingConfig `yaml:"logging"`
}

// BotConfig конфигурация бота
type BotConfig struct {
	Telegram TelegramConfig `yaml:"telegram"`
	Redis    RedisConfig    `yaml:"redis"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	SecretKey   string `yaml:"secret_key"`
}

// RedisConfig конфигурация Redis
type RedisConfig struct {
	Address  string `yaml:"address"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// APIConfig конфигурация HTTP API
type APIConfig struct {
	BaseURL string `yaml:"base_url"`
	Timeout string `yaml:"timeout,omitempty"`
}

// MetricsConfig конфигурация метрик
type MetricsConfig struct {
	CPUTemperature bool   `yaml:"cpu_temperature"`
	Interval       string `yaml:"interval"`
}

// LoggingConfig конфигурация логирования
type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// TelegramConfig конфигурация Telegram бота
type TelegramConfig struct {
	Token string `yaml:"token"`
}

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	URL string `yaml:"url"`
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

	return &config, nil
}

// LoadBotConfig загружает конфигурацию бота
func LoadBotConfig(filepath string) (*BotConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл конфигурации: %v", err)
	}

	var config BotConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("не удалось парсить конфигурацию: %v", err)
	}

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
	
	// Проверяем, что есть либо Redis, либо HTTP API конфигурация
	if c.Redis.Address == "" && c.API.BaseURL == "" {
		return fmt.Errorf("должен быть указан либо адрес Redis, либо базовый URL API")
	}
	
	return nil
}

// validate валидирует конфигурацию бота
func (c *BotConfig) validate() error {
	if c.Telegram.Token == "" {
		return fmt.Errorf("токен Telegram бота не может быть пустым")
	}
	if c.Redis.Address == "" {
		return fmt.Errorf("адрес Redis не может быть пустым")
	}
	if c.Database.URL == "" {
		return fmt.Errorf("URL базы данных не может быть пустым")
	}
	return nil
}
