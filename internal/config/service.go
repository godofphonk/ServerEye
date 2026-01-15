package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ConfigService provides configuration management
type ConfigService struct{}

// NewConfigService creates a new configuration service
func NewConfigService() *ConfigService {
	return &ConfigService{}
}

// LoadAgentConfig loads agent configuration from file
func (c *ConfigService) LoadAgentConfig(filepath string) (*AgentConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл конфигурации: %v", err)
	}

	var config AgentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("не удалось парсить конфигурацию: %v", err)
	}

	// Валидация конфигурации
	if err := c.Validate(&config); err != nil {
		return nil, fmt.Errorf("некорректная конфигурация: %v", err)
	}

	// Устанавливаем дефолтные значения
	if config.Metrics.Interval == "" {
		config.Metrics.Interval = "30s"
	}

	return &config, nil
}

// Validate validates the agent configuration
func (c *ConfigService) Validate(cfg *AgentConfig) error {
	if cfg.Server.Name == "" {
		return fmt.Errorf("имя сервера не может быть пустым")
	}
	if cfg.Server.SecretKey == "" {
		return fmt.Errorf("секретный ключ не может быть пустым")
	}

	// Проверяем, что указан базовый URL API
	if cfg.API.BaseURL == "" {
		return fmt.Errorf("должен быть указан базовый URL API")
	}

	return nil
}
