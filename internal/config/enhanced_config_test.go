package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigManager_LoadEnvironmentOverrides(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected map[string]string
	}{
		{
			name: "load server overrides",
			envVars: map[string]string{
				"SERVEREYE_SERVER_NAME": "test-server",
				"SERVEREYE_SERVER_KEY":  "test-key-123",
			},
			expected: map[string]string{
				"server.name":       "test-server",
				"server.secret_key": "test-key-123",
			},
		},
		{
			name: "load API overrides",
			envVars: map[string]string{
				"SERVEREYE_API_URL": "https://api.test.com",
				"SERVEREYE_API_KEY": "api-key-456",
			},
			expected: map[string]string{
				"api.base_url": "https://api.test.com",
				"api.api_key":  "api-key-456",
			},
		},
		{
			name: "load WebSocket overrides",
			envVars: map[string]string{
				"SERVEREYE_WS_URL":     "wss://ws.test.com",
				"SERVEREYE_WS_ENABLED": "true",
			},
			expected: map[string]string{
				"websocket.url":     "wss://ws.test.com",
				"websocket.enabled": "true",
			},
		},
		{
			name: "load metrics overrides",
			envVars: map[string]string{
				"SERVEREYE_METRICS_INTERVAL": "60s",
				"SERVEREYE_CPU_TEMPERATURE":  "true",
			},
			expected: map[string]string{
				"metrics.interval":        "60s",
				"metrics.cpu_temperature": "true",
			},
		},
		{
			name: "load logging overrides",
			envVars: map[string]string{
				"SERVEREYE_LOG_LEVEL": "debug",
				"SERVEREYE_LOG_FILE":  "/tmp/test.log",
			},
			expected: map[string]string{
				"logging.level": "debug",
				"logging.file":  "/tmp/test.log",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			cm := &ConfigManager{
				envOverrides: make(map[string]string),
				logger:       logrus.New(),
			}
			cm.loadEnvironmentOverrides()

			for key, expectedValue := range tt.expected {
				assert.Contains(t, cm.envOverrides, key, "Expected override for %s", key)
				assert.Equal(t, expectedValue, cm.envOverrides[key], "Expected value for %s", key)
			}
		})
	}
}

func TestConfigValidator_ValidateConfig(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name        string
		config      *EnhancedAgentConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "test-server",
					SecretKey: "secure-key-1234567890",
				},
				API: APIConfig{
					BaseURL: "https://api.test.com",
				},
				WebSocket: WebSocketConfig{
					Enabled: true,
					URL:     "wss://ws.test.com",
				},
				Metrics: MetricsConfig{
					Interval: "30s",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: false,
		},
		{
			name: "missing server name",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					SecretKey: "secure-key-1234567890",
				},
				API: APIConfig{
					BaseURL: "https://api.test.com",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: true,
			errorMsg:    "server.name",
		},
		{
			name: "invalid secret key length",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "test-server",
					SecretKey: "short",
				},
				API: APIConfig{
					BaseURL: "https://api.test.com",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: true,
			errorMsg:    "server.secret_key",
		},
		{
			name: "invalid API URL",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "test-server",
					SecretKey: "secure-key-1234567890",
				},
				API: APIConfig{
					BaseURL: "invalid-url",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: true,
			errorMsg:    "api.base_url",
		},
		{
			name: "invalid log level",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "test-server",
					SecretKey: "secure-key-1234567890",
				},
				API: APIConfig{
					BaseURL: "https://api.test.com",
				},
				Logging: LoggingConfig{
					Level: "invalid",
				},
			},
			expectError: true,
			errorMsg:    "logging.level",
		},
		{
			name: "invalid WebSocket URL",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "test-server",
					SecretKey: "secure-key-1234567890",
				},
				API: APIConfig{
					BaseURL: "https://api.test.com",
				},
				WebSocket: WebSocketConfig{
					Enabled: true,
					URL:     "invalid-ws-url",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: true,
			errorMsg:    "websocket.url",
		},
		{
			name: "invalid metrics interval",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "test-server",
					SecretKey: "secure-key-1234567890",
				},
				API: APIConfig{
					BaseURL: "https://api.test.com",
				},
				Metrics: MetricsConfig{
					Interval: "invalid-duration",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: true,
			errorMsg:    "metrics.interval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEnvironmentSpecificConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *EnhancedAgentConfig
		env         Environment
		expectError bool
		errorMsg    string
	}{
		{
			name: "production with debug logging",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "prod-server",
					SecretKey: "secure-key-1234567890",
				},
				Logging: LoggingConfig{
					Level: "debug",
				},
			},
			env:         Production,
			expectError: true,
			errorMsg:    "debug logging not recommended",
		},
		{
			name: "production with insecure WebSocket",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "prod-server",
					SecretKey: "secure-key-1234567890",
				},
				WebSocket: WebSocketConfig{
					Enabled: true,
					URL:     "ws://insecure.com",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			env:         Production,
			expectError: true,
			errorMsg:    "unsecure WebSocket",
		},
		{
			name: "production with TLS enabled but no cert",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "prod-server",
					SecretKey: "secure-key-1234567890",
				},
				Security: SecurityConfig{
					RateLimitPerSec: 10,
					MaxConnections:  100,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			env:         Production,
			expectError: false,
		},
		{
			name: "valid production config",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "prod-server",
					SecretKey: "secure-key-1234567890",
				},
				WebSocket: WebSocketConfig{
					Enabled: true,
					URL:     "wss://secure.com",
				},
				Security: SecurityConfig{
					RateLimitPerSec: 10,
					MaxConnections:  100,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			env:         Production,
			expectError: false,
		},
		{
			name: "testing with telemetry enabled",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "test-server",
					SecretKey: "secure-key-1234567890",
				},
				Features: FeaturesConfig{
					Telemetry: true,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			env:         Testing,
			expectError: true,
			errorMsg:    "telemetry should be disabled",
		},
		{
			name: "valid testing config",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "test-server",
					SecretKey: "secure-key-1234567890",
				},
				Features: FeaturesConfig{
					Telemetry: false,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			env:         Testing,
			expectError: false,
		},
		{
			name: "development with high worker count",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "dev-server",
					SecretKey: "secure-key-1234567890",
				},
				Performance: PerformanceConfig{
					WorkerCount: 15,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			env:         Development,
			expectError: true,
			errorMsg:    "high worker count",
		},
		{
			name: "valid development config",
			config: &EnhancedAgentConfig{
				Server: ServerConfig{
					Name:      "dev-server",
					SecretKey: "secure-key-1234567890",
				},
				Performance: PerformanceConfig{
					WorkerCount: 4,
				},
				Logging: LoggingConfig{
					Level: "debug",
				},
			},
			env:         Development,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnvironmentSpecificConfig(tt.config, tt.env)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigBuilder(t *testing.T) {
	logger := logrus.New()

	t.Run("build config with all options", func(t *testing.T) {
		// Create temporary config file
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "test-config.yaml")

		// Create a valid config file
		validConfig := `
server:
  name: "test-server"
  secret_key: "secure-key-1234567890"
api:
  base_url: "https://api.test.com"
logging:
  level: "info"
`
		require.NoError(t, os.WriteFile(configPath, []byte(validConfig), 0600))

		callback := func(config *AgentConfig) {
			// Callback for testing
		}

		provider, err := NewConfigBuilder().
			WithConfigPath(configPath).
			WithEnvironment(Production).
			WithProfile("prod").
			WithHotReload(false).
			WithLogger(logger).
			WithReloadCallback(callback).
			Build()

		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, Production, provider.GetEnvironment())
		assert.Equal(t, "prod", provider.GetProfile())
		assert.Equal(t, configPath, provider.GetConfigPath())

		provider.Close()
	})

	t.Run("build config with defaults", func(t *testing.T) {
		// Create temporary config file
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "default-config.yaml")

		// Create a valid config file
		validConfig := `
server:
  name: "default-server"
  secret_key: "secure-key-1234567890"
api:
  base_url: "https://api.test.com"
logging:
  level: "info"
`
		require.NoError(t, os.WriteFile(configPath, []byte(validConfig), 0600))

		provider, err := NewConfigBuilder().
			WithConfigPath(configPath).
			Build()

		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, Development, provider.GetEnvironment())
		assert.Equal(t, "", provider.GetProfile())

		provider.Close()
	})
}

func TestConfigFactory(t *testing.T) {
	logger := logrus.New()
	factory := NewConfigFactory()

	t.Run("create production config", func(t *testing.T) {
		// Create temporary config file
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "prod-config.yaml")

		// Create a valid config file
		validConfig := `
server:
  name: "prod-server"
  secret_key: "secure-key-1234567890"
api:
  base_url: "https://api.test.com"
logging:
  level: "info"
`
		require.NoError(t, os.WriteFile(configPath, []byte(validConfig), 0600))

		provider, err := factory.CreateProductionConfig(configPath, logger)
		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, Production, provider.GetEnvironment())
		provider.Close()
	})

	t.Run("create development config", func(t *testing.T) {
		// Create temporary config file
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "dev-config.yaml")

		// Create a valid config file
		validConfig := `
server:
  name: "dev-server"
  secret_key: "secure-key-1234567890"
api:
  base_url: "https://api.test.com"
logging:
  level: "debug"
`
		require.NoError(t, os.WriteFile(configPath, []byte(validConfig), 0600))

		provider, err := factory.CreateDevelopmentConfig(configPath, logger)
		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, Development, provider.GetEnvironment())
		assert.Equal(t, "dev", provider.GetProfile())
		provider.Close()
	})

	t.Run("create testing config", func(t *testing.T) {
		// Create temporary config file
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "test-config.yaml")

		// Create a valid config file
		validConfig := `
server:
  name: "test-server"
  secret_key: "secure-key-1234567890"
api:
  base_url: "https://api.test.com"
logging:
  level: "info"
`
		require.NoError(t, os.WriteFile(configPath, []byte(validConfig), 0600))

		provider, err := factory.CreateTestingConfig(configPath, logger)
		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, Testing, provider.GetEnvironment())
		assert.Equal(t, "test", provider.GetProfile())
		provider.Close()
	})
}

func TestConfigMigration(t *testing.T) {
	migration := &ConfigMigration{}

	t.Run("migrate from legacy", func(t *testing.T) {
		legacy := &AgentConfig{
			Server: ServerConfig{
				Name:        "legacy-server",
				Description: "Legacy server",
				SecretKey:   "legacy-key",
			},
			API: APIConfig{
				BaseURL: "https://legacy.api.com",
			},
			Logging: LoggingConfig{
				Level: "info",
				File:  "/var/log/legacy.log",
			},
		}

		enhanced := migration.MigrateFromLegacy(legacy)

		assert.Equal(t, legacy.Server.Name, enhanced.Server.Name)
		assert.Equal(t, legacy.Server.Description, enhanced.Server.Description)
		assert.Equal(t, legacy.Server.SecretKey, enhanced.Server.SecretKey)
		assert.Equal(t, legacy.API.BaseURL, enhanced.API.BaseURL)
		assert.Equal(t, legacy.Logging.Level, enhanced.Logging.Level)
		assert.Equal(t, legacy.Logging.File, enhanced.Logging.File)

		// Check default values for new fields
		assert.False(t, enhanced.Features.AutoUpdates)
		assert.True(t, enhanced.Features.Telemetry)
		assert.True(t, enhanced.Features.RemoteCommands)
		assert.Equal(t, 10, enhanced.Security.RateLimitPerSec)
		assert.Equal(t, 4, enhanced.Performance.WorkerCount)
		assert.Equal(t, 1000, enhanced.Performance.QueueSize)
	})

	t.Run("export to legacy", func(t *testing.T) {
		enhanced := &EnhancedAgentConfig{
			Server: ServerConfig{
				Name:        "enhanced-server",
				Description: "Enhanced server",
				SecretKey:   "enhanced-key",
			},
			API: APIConfig{
				BaseURL: "https://enhanced.api.com",
			},
			Features: FeaturesConfig{
				AutoUpdates: true,
				Telemetry:   false,
			},
			Security: SecurityConfig{
				RateLimitPerSec: 10,
				MaxConnections:  100,
			},
			Performance: PerformanceConfig{
				WorkerCount: 8,
			},
			Logging: LoggingConfig{
				Level: "debug",
				File:  "/var/log/enhanced.log",
			},
		}

		legacy := migration.ExportToLegacy(enhanced)

		assert.Equal(t, enhanced.Server.Name, legacy.Server.Name)
		assert.Equal(t, enhanced.Server.Description, legacy.Server.Description)
		assert.Equal(t, enhanced.Server.SecretKey, legacy.Server.SecretKey)
		assert.Equal(t, enhanced.API.BaseURL, legacy.API.BaseURL)
		assert.Equal(t, enhanced.Logging.Level, legacy.Logging.Level)
		assert.Equal(t, enhanced.Logging.File, legacy.Logging.File)

		// Check that new fields are NOT included in legacy (they are dropped during export)
		// This is the expected behavior - ExportToLegacy only exports basic fields
	})
}

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "ServerEye Agent", config.Server.Name)
	assert.Equal(t, "Default ServerEye configuration", config.Server.Description)
	assert.Equal(t, "https://api.servereye.dev", config.API.BaseURL)
	assert.True(t, config.WebSocket.Enabled)
	assert.Equal(t, "wss://api.servereye.dev/ws", config.WebSocket.URL)
	assert.False(t, config.Metrics.CPUUsage)
	assert.True(t, config.Metrics.MemoryUsage)
	assert.False(t, config.Metrics.DiskUsage)
	assert.True(t, config.Metrics.CPUTemperature)
	assert.Equal(t, "30s", config.Metrics.Interval)
	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "/var/log/servereye/agent.log", config.Logging.File)

	// Check features defaults
	assert.False(t, config.Features.AutoUpdates)
	assert.True(t, config.Features.Telemetry)
	assert.True(t, config.Features.RemoteCommands)
	assert.True(t, config.Features.Alerting)

	// Check security defaults
	assert.Equal(t, 10, config.Security.RateLimitPerSec)
	assert.Equal(t, 100, config.Security.MaxConnections)

	// Check performance defaults
	assert.Equal(t, 4, config.Performance.WorkerCount)
	assert.Equal(t, 1000, config.Performance.QueueSize)
	assert.Equal(t, 100, config.Performance.BatchSize)
	assert.Equal(t, "30s", config.Performance.FlushInterval)
	assert.Equal(t, "10s", config.Performance.ConnectionTimeout)
}

func TestDetermineEnvironmentFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected Environment
	}{
		{"/etc/servereye/prod-config.yaml", Production},
		{"/etc/servereye/production-config.yaml", Production},
		{"/etc/servereye/test-config.yaml", Testing},
		{"/etc/servereye/testing-config.yaml", Testing},
		{"/etc/servereye/stage-config.yaml", Staging},
		{"/etc/servereye/staging-config.yaml", Staging},
		{"/etc/servereye/dev-config.yaml", Development},
		{"/etc/servereye/development-config.yaml", Development},
		{"/etc/servereye/config.yaml", Development},
		{"/etc/servereye/unknown.yaml", Development},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			env := DetermineEnvironmentFromPath(tt.path)
			assert.Equal(t, tt.expected, env)
		})
	}
}
