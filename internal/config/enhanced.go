package config

import (
	"fmt"
	"strings"
	"time"
)

// EnhancedAgentConfig extends the original configuration with additional validation fields
type EnhancedAgentConfig struct {
	Server    ServerConfig    `yaml:"server" json:"server" toml:"server"`
	API       APIConfig       `yaml:"api,omitempty" json:"api,omitempty" toml:"api,omitempty"`
	WebSocket WebSocketConfig `yaml:"websocket,omitempty" json:"websocket,omitempty" toml:"websocket,omitempty"`
	Metrics   MetricsConfig   `yaml:"metrics" json:"metrics" toml:"metrics"`
	Logging   LoggingConfig   `yaml:"logging" json:"logging" toml:"logging"`

	// New fields for enhanced configuration
	Features    FeaturesConfig    `yaml:"features,omitempty" json:"features,omitempty" toml:"features,omitempty"`
	Security    SecurityConfig    `yaml:"security,omitempty" json:"security,omitempty" toml:"security,omitempty"`
	Performance PerformanceConfig `yaml:"performance,omitempty" json:"performance,omitempty" toml:"performance,omitempty"`
}

// ConfigValidationRule represents a validation rule
type ConfigValidationRule struct {
	Field       string
	Required    bool
	MinLength   int
	MaxLength   int
	Pattern     string
	ValidValues []string
	Validator   func(interface{}) error
}

// ConfigValidator provides comprehensive configuration validation
type ConfigValidator struct {
	rules []ConfigValidationRule
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		rules: getDefaultValidationRules(),
	}
}

// getDefaultValidationRules returns default validation rules
func getDefaultValidationRules() []ConfigValidationRule {
	return []ConfigValidationRule{
		{
			Field:     "server.name",
			Required:  true,
			MinLength: 1,
			MaxLength: 100,
			Pattern:   `^[a-zA-Z0-9._-]+$`,
		},
		{
			Field:     "server.secret_key",
			Required:  true,
			MinLength: 10,
			MaxLength: 256,
		},
		{
			Field:    "api.base_url",
			Required: true,
			Pattern:  `^https?://.+`,
		},
		{
			Field:       "logging.level",
			Required:    true,
			ValidValues: []string{"debug", "info", "warn", "error"},
		},
		{
			Field:    "websocket.url",
			Required: false,
			Pattern:  `^wss?://.+`,
			Validator: func(value interface{}) error {
				if url, ok := value.(string); ok && url != "" {
					if !isWebSocketURL(url) {
						return fmt.Errorf("invalid WebSocket URL format")
					}
				}
				return nil
			},
		},
		{
			Field: "metrics.interval",
			Validator: func(value interface{}) error {
				if interval, ok := value.(string); ok && interval != "" {
					if _, err := time.ParseDuration(interval); err != nil {
						return fmt.Errorf("invalid duration format: %v", err)
					}
				}
				return nil
			},
		},
	}
}

// ValidateConfig validates the entire configuration
func (cv *ConfigValidator) ValidateConfig(config *EnhancedAgentConfig) error {
	for _, rule := range cv.rules {
		if err := cv.validateField(config, rule); err != nil {
			return fmt.Errorf("validation failed for field %s: %w", rule.Field, err)
		}
	}
	return nil
}

// validateField validates a single field against a rule
func (cv *ConfigValidator) validateField(config *EnhancedAgentConfig, rule ConfigValidationRule) error {
	value := cv.getFieldValue(config, rule.Field)

	// Check if required field is missing
	if rule.Required && cv.isEmpty(value) {
		return fmt.Errorf("required field is empty")
	}

	// Skip validation if field is empty and not required
	if cv.isEmpty(value) && !rule.Required {
		return nil
	}

	// Length validation
	if strValue, ok := value.(string); ok {
		if rule.MinLength > 0 && len(strValue) < rule.MinLength {
			return fmt.Errorf("minimum length %d required", rule.MinLength)
		}
		if rule.MaxLength > 0 && len(strValue) > rule.MaxLength {
			return fmt.Errorf("maximum length %d exceeded", rule.MaxLength)
		}
	}

	// Pattern validation
	if rule.Pattern != "" {
		if strValue, ok := value.(string); ok {
			if !matchesPattern(strValue, rule.Pattern) {
				return fmt.Errorf("does not match pattern %s", rule.Pattern)
			}
		}
	}

	// Valid values validation
	if len(rule.ValidValues) > 0 {
		if strValue, ok := value.(string); ok {
			if !contains(rule.ValidValues, strValue) {
				return fmt.Errorf("must be one of: %v", rule.ValidValues)
			}
		}
	}

	// Custom validator
	if rule.Validator != nil {
		if err := rule.Validator(value); err != nil {
			return err
		}
	}

	return nil
}

// getFieldValue gets the value of a field from the config
func (cv *ConfigValidator) getFieldValue(config *EnhancedAgentConfig, field string) interface{} {
	switch field {
	case "server.name":
		return config.Server.Name
	case "server.secret_key":
		return config.Server.SecretKey
	case "server.server_id":
		return config.Server.ServerID
	case "api.base_url":
		return config.API.BaseURL
	case "api.api_key":
		return config.API.APIKey
	case "websocket.url":
		return config.WebSocket.URL
	case "websocket.enabled":
		return config.WebSocket.Enabled
	case "logging.level":
		return config.Logging.Level
	case "logging.file":
		return config.Logging.File
	case "metrics.interval":
		return config.Metrics.Interval
	case "metrics.cpu_temperature":
		return config.Metrics.CPUTemperature
	default:
		return nil
	}
}

// isEmpty checks if a value is empty
func (cv *ConfigValidator) isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	if str, ok := value.(string); ok {
		return str == ""
	}
	if b, ok := value.(bool); ok {
		return !b
	}
	if i, ok := value.(int); ok {
		return i == 0
	}
	return false
}

// isWebSocketURL checks if a string is a valid WebSocket URL
func isWebSocketURL(url string) bool {
	return len(url) >= 3 && (url[:3] == "ws:" || url[:4] == "wss:")
}

// matchesPattern checks if a string matches a regex pattern
func matchesPattern(value, pattern string) bool {
	// Simple pattern matching for common cases
	// In a real implementation, you would use regexp package
	switch pattern {
	case `^[a-zA-Z0-9._-]+$`:
		for _, r := range value {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '.' && r != '_' && r != '-' {
				return false
			}
		}
		return true
	case `^https?://.+`:
		return len(value) >= 7 && (value[:7] == "http://" || value[:8] == "https://")
	case `^wss?://.+`:
		return len(value) >= 3 && (value[:3] == "ws:" || value[:4] == "wss:")
	default:
		return true
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ValidateEnvironmentSpecificConfig performs environment-specific validation
func ValidateEnvironmentSpecificConfig(config *EnhancedAgentConfig, env Environment) error {
	switch env {
	case Production:
		// Production-specific validations
		if config.Logging.Level == "debug" {
			return fmt.Errorf("debug logging not recommended in production")
		}
		if config.WebSocket.Enabled && strings.HasPrefix(config.WebSocket.URL, "ws://") {
			return fmt.Errorf("unsecure WebSocket (ws://) not recommended in production")
		}
		if config.Security.EnableTLS && (config.Security.TLSCertFile == "" || config.Security.TLSKeyFile == "") {
			return fmt.Errorf("TLS certificate and key files required when TLS is enabled")
		}

	case Development:
		// Development-specific validations
		if config.Performance.WorkerCount > 10 {
			return fmt.Errorf("high worker count may impact development performance")
		}

	case Testing:
		// Testing-specific validations
		if config.Features.Telemetry {
			return fmt.Errorf("telemetry should be disabled in testing environment")
		}
	}

	return nil
}

// GetDefaultConfig returns a default configuration
func GetDefaultConfig() *EnhancedAgentConfig {
	return &EnhancedAgentConfig{
		Server: ServerConfig{
			Name:        "ServerEye Agent",
			Description: "Default ServerEye configuration",
		},
		API: APIConfig{
			BaseURL: "https://api.servereye.dev",
			Timeout: "30s",
		},
		WebSocket: WebSocketConfig{
			Enabled:              true,
			URL:                  "wss://api.servereye.dev/ws",
			ReconnectInterval:    "5s",
			MaxReconnectAttempts: 10,
			PingInterval:         "30s",
			WriteTimeout:         "10s",
			ReadTimeout:          "10s",
			HandshakeTimeout:     "10s",
			BufferSize:           1000,
			EnableCompression:    true,
			MetricBufferSize:     100,
			MetricBufferFlush:    "30s",
			CommandQueueSize:     100,
			CommandTimeout:       "30s",
		},
		Metrics: MetricsConfig{
			CPUUsage:       false,
			MemoryUsage:    true,
			DiskUsage:      false,
			CPUTemperature: true,
			Interval:       "30s",
		},
		Logging: LoggingConfig{
			Level: "info",
			File:  "/var/log/servereye/agent.log",
		},
		Features: FeaturesConfig{
			AutoUpdates:      false,
			Telemetry:        true,
			RemoteCommands:   true,
			Alerting:         true,
			DockerMonitoring: true,
		},
		Security: SecurityConfig{
			EnableTLS:       false,
			RateLimitPerSec: 10,
			MaxConnections:  100,
		},
		Performance: PerformanceConfig{
			WorkerCount:       4,
			QueueSize:         1000,
			BatchSize:         100,
			FlushInterval:     "30s",
			ConnectionTimeout: "10s",
		},
	}
}
