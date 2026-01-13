package config

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// ConfigInterface defines the interface for configuration management
type ConfigInterface interface {
	// Basic configuration access
	GetServerConfig() ServerConfig
	GetAPIConfig() APIConfig
	GetWebSocketConfig() WebSocketConfig
	GetMetricsConfig() MetricsConfig
	GetLoggingConfig() LoggingConfig
	GetFeaturesConfig() FeaturesConfig
	GetSecurityConfig() SecurityConfig
	GetPerformanceConfig() PerformanceConfig

	// Configuration management
	Reload() error
	GetEnvironment() Environment
	GetProfile() string
	GetConfigPath() string

	// Hot-reload support
	AddReloadCallback(callback func(*EnhancedAgentConfig))
	RemoveReloadCallback(callbackID string)

	// Validation
	Validate() error

	// Lifecycle
	Close() error
}

// ConfigProvider provides configuration with backward compatibility
type ConfigProvider struct {
	manager   *ConfigManager
	validator *ConfigValidator
}

// NewConfigProvider creates a new configuration provider
func NewConfigProvider(opts ConfigManagerOptions) (*ConfigProvider, error) {
	manager, err := NewConfigManager(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	validator := NewConfigValidator()

	return &ConfigProvider{
		manager:   manager,
		validator: validator,
	}, nil
}

// GetConfig returns the current configuration (thread-safe)
func (cp *ConfigProvider) GetConfig() *AgentConfig {
	cp.manager.mu.RLock()
	defer cp.manager.mu.RUnlock()

	// Return a copy to prevent external modifications
	configCopy := *cp.manager.config
	return &configCopy
}

// GetServerConfig returns server configuration
func (cp *ConfigProvider) GetServerConfig() ServerConfig {
	config := cp.GetConfig()
	return config.Server
}

// GetAPIConfig returns API configuration
func (cp *ConfigProvider) GetAPIConfig() APIConfig {
	config := cp.GetConfig()
	return config.API
}

// GetWebSocketConfig returns WebSocket configuration
func (cp *ConfigProvider) GetWebSocketConfig() WebSocketConfig {
	config := cp.manager.GetConfig()
	return config.WebSocket
}

// GetMetricsConfig returns metrics configuration
func (cp *ConfigProvider) GetMetricsConfig() MetricsConfig {
	config := cp.manager.GetConfig()
	return config.Metrics
}

// GetLoggingConfig returns logging configuration
func (cp *ConfigProvider) GetLoggingConfig() LoggingConfig {
	config := cp.manager.GetConfig()
	return config.Logging
}

// GetFeaturesConfig returns features configuration
func (cp *ConfigProvider) GetFeaturesConfig() FeaturesConfig {
	config := cp.manager.GetConfig()
	return config.Features
}

// GetSecurityConfig returns security configuration
func (cp *ConfigProvider) GetSecurityConfig() SecurityConfig {
	config := cp.manager.GetConfig()
	return config.Security
}

// GetPerformanceConfig returns performance configuration
func (cp *ConfigProvider) GetPerformanceConfig() PerformanceConfig {
	config := cp.manager.GetConfig()
	return config.Performance
}

// Reload reloads the configuration
func (cp *ConfigProvider) Reload() error {
	return cp.manager.ReloadConfig()
}

// GetEnvironment returns the current environment
func (cp *ConfigProvider) GetEnvironment() Environment {
	return cp.manager.GetEnvironment()
}

// GetProfile returns the current profile
func (cp *ConfigProvider) GetProfile() string {
	return cp.manager.GetProfile()
}

// GetConfigPath returns the configuration file path
func (cp *ConfigProvider) GetConfigPath() string {
	return cp.manager.configPath
}

// AddReloadCallback adds a reload callback
func (cp *ConfigProvider) AddReloadCallback(callback func(*AgentConfig)) {
	cp.manager.AddReloadCallback(callback)
}

// RemoveReloadCallback removes a reload callback (placeholder for future implementation)
func (cp *ConfigProvider) RemoveReloadCallback(callbackID string) {
	// TODO: Implement callback removal by ID
}

// Validate validates the current configuration
func (cp *ConfigProvider) Validate() error {
	config := cp.manager.GetConfig()

	// Convert to enhanced config for validation
	enhancedConfig := &EnhancedAgentConfig{
		Server:      config.Server,
		API:         config.API,
		WebSocket:   config.WebSocket,
		Metrics:     config.Metrics,
		Logging:     config.Logging,
		Features:    config.Features,
		Security:    config.Security,
		Performance: config.Performance,
	}

	// Run basic validation
	if err := cp.validator.ValidateConfig(enhancedConfig); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Run environment-specific validation
	if err := ValidateEnvironmentSpecificConfig(enhancedConfig, cp.manager.GetEnvironment()); err != nil {
		return fmt.Errorf("environment-specific validation failed: %w", err)
	}

	return nil
}

// Close closes the configuration provider
func (cp *ConfigProvider) Close() error {
	return cp.manager.Close()
}

// ConfigBuilder provides a fluent interface for building configuration
type ConfigBuilder struct {
	opts ConfigManagerOptions
}

// NewConfigBuilder creates a new configuration builder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		opts: ConfigManagerOptions{
			Environment:     Development,
			EnableHotReload: true,
		},
	}
}

// WithConfigPath sets the configuration file path
func (cb *ConfigBuilder) WithConfigPath(path string) *ConfigBuilder {
	cb.opts.ConfigPath = path
	return cb
}

// WithEnvironment sets the environment
func (cb *ConfigBuilder) WithEnvironment(env Environment) *ConfigBuilder {
	cb.opts.Environment = env
	return cb
}

// WithProfile sets the profile
func (cb *ConfigBuilder) WithProfile(profile string) *ConfigBuilder {
	cb.opts.Profile = profile
	return cb
}

// WithHotReload enables or disables hot-reload
func (cb *ConfigBuilder) WithHotReload(enabled bool) *ConfigBuilder {
	cb.opts.EnableHotReload = enabled
	return cb
}

// WithLogger sets the logger
func (cb *ConfigBuilder) WithLogger(logger *logrus.Logger) *ConfigBuilder {
	cb.opts.Logger = logger
	return cb
}

// WithReloadCallback adds a reload callback
func (cb *ConfigBuilder) WithReloadCallback(callback func(*AgentConfig)) *ConfigBuilder {
	cb.opts.ReloadCallbacks = append(cb.opts.ReloadCallbacks, callback)
	return cb
}

// Build creates the configuration provider
func (cb *ConfigBuilder) Build() (*ConfigProvider, error) {
	return NewConfigProvider(cb.opts)
}

// ConfigFactory provides factory methods for common configurations
type ConfigFactory struct{}

// NewConfigFactory creates a new configuration factory
func NewConfigFactory() *ConfigFactory {
	return &ConfigFactory{}
}

// CreateProductionConfig creates a production configuration
func (cf *ConfigFactory) CreateProductionConfig(configPath string, logger *logrus.Logger) (*ConfigProvider, error) {
	return NewConfigBuilder().
		WithConfigPath(configPath).
		WithEnvironment(Production).
		WithHotReload(false). // Typically disabled in production
		WithLogger(logger).
		Build()
}

// CreateDevelopmentConfig creates a development configuration
func (cf *ConfigFactory) CreateDevelopmentConfig(configPath string, logger *logrus.Logger) (*ConfigProvider, error) {
	return NewConfigBuilder().
		WithConfigPath(configPath).
		WithEnvironment(Development).
		WithProfile("dev").
		WithHotReload(true).
		WithLogger(logger).
		Build()
}

// CreateTestingConfig creates a testing configuration
func (cf *ConfigFactory) CreateTestingConfig(configPath string, logger *logrus.Logger) (*ConfigProvider, error) {
	return NewConfigBuilder().
		WithConfigPath(configPath).
		WithEnvironment(Testing).
		WithProfile("test").
		WithHotReload(false).
		WithLogger(logger).
		Build()
}

// LoadAgentConfigWithEnhancements loads the original AgentConfig with enhancements
// This provides backward compatibility with the existing codebase
func LoadAgentConfigWithEnhancements(filepath string) (*AgentConfig, error) {
	// Use the enhanced config manager
	provider, err := NewConfigBuilder().
		WithConfigPath(filepath).
		WithEnvironment(DetermineEnvironmentFromPath(filepath)).
		Build()
	if err != nil {
		return nil, err
	}
	defer provider.Close()

	// Get enhanced configuration
	enhancedConfig := provider.manager.GetConfig()

	// Convert back to original AgentConfig for backward compatibility
	config := &AgentConfig{
		Server:    enhancedConfig.Server,
		API:       enhancedConfig.API,
		WebSocket: enhancedConfig.WebSocket,
		Metrics:   enhancedConfig.Metrics,
		Logging:   enhancedConfig.Logging,
	}

	return config, nil
}

// DetermineEnvironmentFromPath determines environment from config file path
func DetermineEnvironmentFromPath(path string) Environment {
	lowerPath := strings.ToLower(path)
	if strings.Contains(lowerPath, "prod") || strings.Contains(lowerPath, "production") {
		return Production
	}
	if strings.Contains(lowerPath, "test") || strings.Contains(lowerPath, "testing") {
		return Testing
	}
	if strings.Contains(lowerPath, "stage") || strings.Contains(lowerPath, "staging") {
		return Staging
	}
	if strings.Contains(lowerPath, "dev") || strings.Contains(lowerPath, "development") {
		return Development
	}
	return Development
}

// ConfigMigration provides migration utilities for configuration formats
type ConfigMigration struct{}

// MigrateFromLegacy migrates from legacy configuration format
func (cm *ConfigMigration) MigrateFromLegacy(legacyConfig *AgentConfig) *EnhancedAgentConfig {
	enhanced := &EnhancedAgentConfig{
		Server:    legacyConfig.Server,
		API:       legacyConfig.API,
		WebSocket: legacyConfig.WebSocket,
		Metrics:   legacyConfig.Metrics,
		Logging:   legacyConfig.Logging,

		// Set sensible defaults for new fields
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

	return enhanced
}

// ExportToLegacy exports enhanced configuration to legacy format
func (cm *ConfigMigration) ExportToLegacy(enhanced *EnhancedAgentConfig) *AgentConfig {
	return &AgentConfig{
		Server:    enhanced.Server,
		API:       enhanced.API,
		WebSocket: enhanced.WebSocket,
		Metrics:   enhanced.Metrics,
		Logging:   enhanced.Logging,
	}
}
