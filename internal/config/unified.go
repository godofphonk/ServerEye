package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// UnifiedConfig - единая структура конфигурации с backward compatibility
type UnifiedConfig struct {
	Server      ServerConfig      `yaml:"server" json:"server" toml:"server"`
	API         APIConfig         `yaml:"api,omitempty" json:"api,omitempty" toml:"api,omitempty"`
	WebSocket   WebSocketConfig   `yaml:"websocket,omitempty" json:"websocket,omitempty" toml:"websocket,omitempty"`
	Metrics     MetricsConfig     `yaml:"metrics" json:"metrics" toml:"metrics"`
	Logging     LoggingConfig     `yaml:"logging" json:"logging" toml:"logging"`
	Features    FeaturesConfig    `yaml:"features,omitempty" json:"features,omitempty" toml:"features,omitempty"`
	Security    SecurityConfig    `yaml:"security,omitempty" json:"security,omitempty" toml:"security,omitempty"`
	Performance PerformanceConfig `yaml:"performance,omitempty" json:"performance,omitempty" toml:"performance,omitempty"`

	// Internal fields
	environment string `yaml:"-" json:"-" toml:"-"`
	configPath  string `yaml:"-" json:"-" toml:"-"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *UnifiedConfig {
	return &UnifiedConfig{
		Server: ServerConfig{
			Name:        "ServerEye Agent",
			Description: "ServerEye monitoring agent",
		},
		API: APIConfig{
			BaseURL: "https://api.servereye.com",
			Timeout: "30s",
		},
		WebSocket: WebSocketConfig{
			Enabled:              true,
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
			CPUUsage:       true,
			MemoryUsage:    true,
			DiskUsage:      true,
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

// LoadUnifiedConfig loads configuration from file with fallback to defaults
func LoadUnifiedConfig(filepath string) (*UnifiedConfig, error) {
	config := DefaultConfig()
	config.configPath = filepath

	// Try to load from file
	if data, err := os.ReadFile(filepath); err == nil {
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Apply environment-specific overrides
	config.applyEnvironmentOverrides()

	return config, nil
}

// Validate validates the configuration
func (c *UnifiedConfig) Validate() error {
	// Required fields
	if c.Server.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if c.Server.SecretKey == "" {
		return fmt.Errorf("server secret key is required")
	}
	if c.API.BaseURL == "" {
		return fmt.Errorf("API base URL is required")
	}

	// Validate URLs
	if !isValidURL(c.API.BaseURL) {
		return fmt.Errorf("invalid API base URL: %s", c.API.BaseURL)
	}

	if c.WebSocket.URL != "" && !isValidWebSocketURL(c.WebSocket.URL) {
		return fmt.Errorf("invalid WebSocket URL: %s", c.WebSocket.URL)
	}

	// Validate durations
	if _, err := time.ParseDuration(c.Metrics.Interval); err != nil {
		return fmt.Errorf("invalid metrics interval: %s", c.Metrics.Interval)
	}

	// Validate WebSocket durations if WebSocket is enabled
	if c.WebSocket.Enabled {
		durations := []string{
			c.WebSocket.ReconnectInterval,
			c.WebSocket.PingInterval,
			c.WebSocket.WriteTimeout,
			c.WebSocket.ReadTimeout,
			c.WebSocket.HandshakeTimeout,
			c.WebSocket.MetricBufferFlush,
			c.WebSocket.CommandTimeout,
		}
		for _, dur := range durations {
			if dur != "" {
				if _, err := time.ParseDuration(dur); err != nil {
					return fmt.Errorf("invalid WebSocket duration: %s", dur)
				}
			}
		}
	}

	// Validate performance settings
	if c.Performance.WorkerCount <= 0 {
		return fmt.Errorf("worker count must be positive")
	}
	if c.Performance.QueueSize <= 0 {
		return fmt.Errorf("queue size must be positive")
	}
	if c.Performance.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}

	return nil
}

// applyEnvironmentOverrides applies environment-specific configuration
func (c *UnifiedConfig) applyEnvironmentOverrides() {
	env := c.environment
	if env == "" {
		env = determineEnvironmentFromPath(c.configPath)
		c.environment = env
	}

	switch env {
	case "production":
		c.Logging.Level = "warn"
		c.Features.Telemetry = false
		c.Features.AutoUpdates = false
		c.Performance.WorkerCount = 8
		c.Security.EnableTLS = true
	case "staging":
		c.Logging.Level = "info"
		c.Features.Telemetry = true
		c.Features.AutoUpdates = false
		c.Performance.WorkerCount = 4
	case "testing":
		c.Logging.Level = "debug"
		c.Features.Telemetry = false
		c.Metrics.Interval = "1s"
		c.Performance.WorkerCount = 2
	case "development":
		c.Logging.Level = "debug"
		c.Features.Telemetry = true
		c.Performance.WorkerCount = 2
	}
}

// GetWebSocketURL returns the WebSocket URL with fallback
func (c *UnifiedConfig) GetWebSocketURL() string {
	if c.WebSocket.URL != "" {
		return c.WebSocket.URL
	}
	// Fallback to API URL with WebSocket protocol
	if strings.HasPrefix(c.API.BaseURL, "http://") {
		return "ws" + c.API.BaseURL[4:] + "/ws"
	}
	if strings.HasPrefix(c.API.BaseURL, "https://") {
		return "wss" + c.API.BaseURL[5:] + "/ws"
	}
	return "ws://localhost:8080/ws"
}

// GetMetricsInterval returns the parsed metrics interval
func (c *UnifiedConfig) GetMetricsInterval() time.Duration {
	if interval, err := time.ParseDuration(c.Metrics.Interval); err == nil {
		return interval
	}
	return 30 * time.Second // fallback
}

// ToAgentConfig converts UnifiedConfig to legacy AgentConfig for backward compatibility
func (c *UnifiedConfig) ToAgentConfig() *AgentConfig {
	return &AgentConfig{
		Server:      c.Server,
		API:         c.API,
		WebSocket:   c.WebSocket,
		Metrics:     c.Metrics,
		Logging:     c.Logging,
		Features:    c.Features,
		Security:    c.Security,
		Performance: c.Performance,
	}
}

// FromAgentConfig creates UnifiedConfig from legacy AgentConfig
func FromAgentConfig(agentConfig *AgentConfig) *UnifiedConfig {
	config := DefaultConfig()
	config.Server = agentConfig.Server
	config.API = agentConfig.API
	config.WebSocket = agentConfig.WebSocket
	config.Metrics = agentConfig.Metrics
	config.Logging = agentConfig.Logging
	config.Features = agentConfig.Features
	config.Security = agentConfig.Security
	config.Performance = agentConfig.Performance
	return config
}

// Helper functions
func isValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func isValidWebSocketURL(url string) bool {
	return strings.HasPrefix(url, "ws://") || strings.HasPrefix(url, "wss://")
}

func determineEnvironmentFromPath(path string) string {
	lowerPath := strings.ToLower(path)
	if strings.Contains(lowerPath, "prod") || strings.Contains(lowerPath, "production") {
		return "production"
	}
	if strings.Contains(lowerPath, "test") || strings.Contains(lowerPath, "testing") {
		return "testing"
	}
	if strings.Contains(lowerPath, "stage") || strings.Contains(lowerPath, "staging") {
		return "staging"
	}
	if strings.Contains(lowerPath, "dev") || strings.Contains(lowerPath, "development") {
		return "development"
	}
	return "development"
}
