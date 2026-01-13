package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// ConfigFormat represents supported configuration formats
type ConfigFormat string

const (
	FormatYAML ConfigFormat = "yaml"
	FormatJSON ConfigFormat = "json"
	FormatTOML ConfigFormat = "toml"
)

// Environment profile types
type Environment string

const (
	Development Environment = "development"
	Testing     Environment = "testing"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

// ConfigManager manages configuration with hot-reload and environment support
type ConfigManager struct {
	mu               sync.RWMutex
	config           *AgentConfig
	environment      Environment
	profile          string
	configPath       string
	watcher          *fsnotify.Watcher
	logger           *logrus.Logger
	reloadCallbacks  []func(*AgentConfig)
	hotReloadEnabled bool
	envOverrides     map[string]string
}

// ConfigManagerOptions for creating a new ConfigManager
type ConfigManagerOptions struct {
	ConfigPath      string
	Environment     Environment
	Profile         string
	EnableHotReload bool
	Logger          *logrus.Logger
	ReloadCallbacks []func(*AgentConfig)
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(opts ConfigManagerOptions) (*ConfigManager, error) {
	if opts.Logger == nil {
		opts.Logger = logrus.New()
	}

	cm := &ConfigManager{
		environment:      opts.Environment,
		profile:          opts.Profile,
		configPath:       opts.ConfigPath,
		logger:           opts.Logger,
		reloadCallbacks:  opts.ReloadCallbacks,
		hotReloadEnabled: opts.EnableHotReload,
		envOverrides:     make(map[string]string),
	}

	// Load environment overrides
	cm.loadEnvironmentOverrides()

	// Load initial configuration
	if err := cm.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	// Setup hot-reload if enabled
	if opts.EnableHotReload {
		if err := cm.setupWatcher(); err != nil {
			cm.logger.WithError(err).Warn("Failed to setup config watcher, hot-reload disabled")
		}
	}

	return cm, nil
}

// loadEnvironmentOverrides loads environment variable overrides
func (cm *ConfigManager) loadEnvironmentOverrides() {
	// Server configuration
	if serverName := os.Getenv("SERVEREYE_SERVER_NAME"); serverName != "" {
		cm.envOverrides["server.name"] = serverName
	}
	if serverKey := os.Getenv("SERVEREYE_SERVER_KEY"); serverKey != "" {
		cm.envOverrides["server.secret_key"] = serverKey
	}
	if serverID := os.Getenv("SERVEREYE_SERVER_ID"); serverID != "" {
		cm.envOverrides["server.server_id"] = serverID
	}

	// API configuration
	if apiURL := os.Getenv("SERVEREYE_API_URL"); apiURL != "" {
		cm.envOverrides["api.base_url"] = apiURL
	}
	if apiKey := os.Getenv("SERVEREYE_API_KEY"); apiKey != "" {
		cm.envOverrides["api.api_key"] = apiKey
	}

	// WebSocket configuration
	if wsURL := os.Getenv("SERVEREYE_WS_URL"); wsURL != "" {
		cm.envOverrides["websocket.url"] = wsURL
	}
	if wsEnabled := os.Getenv("SERVEREYE_WS_ENABLED"); wsEnabled != "" {
		cm.envOverrides["websocket.enabled"] = wsEnabled
	}

	// Metrics configuration
	if metricsInterval := os.Getenv("SERVEREYE_METRICS_INTERVAL"); metricsInterval != "" {
		cm.envOverrides["metrics.interval"] = metricsInterval
	}
	if cpuTemp := os.Getenv("SERVEREYE_CPU_TEMPERATURE"); cpuTemp != "" {
		cm.envOverrides["metrics.cpu_temperature"] = cpuTemp
	}

	// Logging configuration
	if logLevel := os.Getenv("SERVEREYE_LOG_LEVEL"); logLevel != "" {
		cm.envOverrides["logging.level"] = logLevel
	}
	if logFile := os.Getenv("SERVEREYE_LOG_FILE"); logFile != "" {
		cm.envOverrides["logging.file"] = logFile
	}

	cm.logger.WithField("overrides", len(cm.envOverrides)).Debug("Loaded environment overrides")
}

// loadConfig loads configuration from file with environment and profile support
func (cm *ConfigManager) loadConfig() error {
	// Determine config format
	format := cm.detectConfigFormat()

	// Load base configuration
	var config AgentConfig
	var err error

	switch format {
	case FormatYAML:
		config, err = cm.loadYAMLConfig()
	case FormatJSON:
		config, err = cm.loadJSONConfig()
	case FormatTOML:
		config, err = cm.loadTOMLConfig()
	default:
		return fmt.Errorf("unsupported config format: %s", format)
	}

	if err != nil {
		return err
	}

	// Apply environment-specific overrides
	if cm.environment != "" {
		if err := cm.applyEnvironmentOverrides(&config); err != nil {
			cm.logger.WithError(err).Warn("Failed to apply environment overrides")
		}
	}

	// Apply profile-specific overrides
	if cm.profile != "" {
		if err := cm.applyProfileOverrides(&config); err != nil {
			cm.logger.WithError(err).Warn("Failed to apply profile overrides")
		}
	}

	// Apply environment variable overrides
	cm.applyEnvVariableOverrides(&config)

	// Validate configuration
	if err := cm.validateConfig(&config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	cm.mu.Lock()
	cm.config = &config
	cm.mu.Unlock()

	cm.logger.WithFields(logrus.Fields{
		"format":      format,
		"environment": cm.environment,
		"profile":     cm.profile,
	}).Info("Configuration loaded successfully")

	return nil
}

// detectConfigFormat detects the configuration file format
func (cm *ConfigManager) detectConfigFormat() ConfigFormat {
	ext := strings.ToLower(filepath.Ext(cm.configPath))
	switch ext {
	case ".yaml", ".yml":
		return FormatYAML
	case ".json":
		return FormatJSON
	case ".toml":
		return FormatTOML
	default:
		return FormatYAML // Default to YAML
	}
}

// loadYAMLConfig loads configuration from YAML file
func (cm *ConfigManager) loadYAMLConfig() (AgentConfig, error) {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return AgentConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AgentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return AgentConfig{}, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return config, nil
}

// loadJSONConfig loads configuration from JSON file
func (cm *ConfigManager) loadJSONConfig() (AgentConfig, error) {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return AgentConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AgentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return AgentConfig{}, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	return config, nil
}

// loadTOMLConfig loads configuration from TOML file
func (cm *ConfigManager) loadTOMLConfig() (AgentConfig, error) {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return AgentConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AgentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return AgentConfig{}, fmt.Errorf("failed to parse TOML config: %w", err)
	}

	return config, nil
}

// applyEnvironmentOverrides applies environment-specific configuration overrides
func (cm *ConfigManager) applyEnvironmentOverrides(config *AgentConfig) error {
	envConfigPath := cm.getEnvironmentConfigPath()
	if _, err := os.Stat(envConfigPath); os.IsNotExist(err) {
		return nil // No environment-specific config
	}

	data, err := os.ReadFile(envConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read environment config: %w", err)
	}

	var envConfig AgentConfig
	if err := yaml.Unmarshal(data, &envConfig); err != nil {
		return fmt.Errorf("failed to parse environment config: %w", err)
	}

	// Merge environment config
	cm.mergeConfig(config, envConfig)

	cm.logger.WithField("environment", cm.environment).Debug("Applied environment overrides")
	return nil
}

// applyProfileOverrides applies profile-specific configuration overrides
func (cm *ConfigManager) applyProfileOverrides(config *AgentConfig) error {
	profileConfigPath := cm.getProfileConfigPath()
	if _, err := os.Stat(profileConfigPath); os.IsNotExist(err) {
		return nil // No profile-specific config
	}

	data, err := os.ReadFile(profileConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read profile config: %w", err)
	}

	var profileConfig AgentConfig
	if err := yaml.Unmarshal(data, &profileConfig); err != nil {
		return fmt.Errorf("failed to parse profile config: %w", err)
	}

	// Merge profile config
	cm.mergeConfig(config, profileConfig)

	cm.logger.WithField("profile", cm.profile).Debug("Applied profile overrides")
	return nil
}

// applyEnvVariableOverrides applies environment variable overrides
func (cm *ConfigManager) applyEnvVariableOverrides(config *AgentConfig) {
	// Apply server overrides
	if name, ok := cm.envOverrides["server.name"]; ok {
		config.Server.Name = name
	}
	if key, ok := cm.envOverrides["server.secret_key"]; ok {
		config.Server.SecretKey = key
	}
	if id, ok := cm.envOverrides["server.server_id"]; ok {
		config.Server.ServerID = id
	}

	// Apply API overrides
	if url, ok := cm.envOverrides["api.base_url"]; ok {
		config.API.BaseURL = url
	}
	if key, ok := cm.envOverrides["api.api_key"]; ok {
		config.API.APIKey = key
	}

	// Apply WebSocket overrides
	if url, ok := cm.envOverrides["websocket.url"]; ok {
		config.WebSocket.URL = url
	}
	if enabled, ok := cm.envOverrides["websocket.enabled"]; ok {
		config.WebSocket.Enabled = enabled == "true" || enabled == "1"
	}

	// Apply metrics overrides
	if interval, ok := cm.envOverrides["metrics.interval"]; ok {
		config.Metrics.Interval = interval
	}
	if cpuTemp, ok := cm.envOverrides["metrics.cpu_temperature"]; ok {
		config.Metrics.CPUTemperature = cpuTemp == "true" || cpuTemp == "1"
	}

	// Apply logging overrides
	if level, ok := cm.envOverrides["logging.level"]; ok {
		config.Logging.Level = level
	}
	if file, ok := cm.envOverrides["logging.file"]; ok {
		config.Logging.File = file
	}

	if len(cm.envOverrides) > 0 {
		cm.logger.WithField("count", len(cm.envOverrides)).Debug("Applied environment variable overrides")
	}
}

// mergeConfig merges source config into destination config
func (cm *ConfigManager) mergeConfig(dest *AgentConfig, src AgentConfig) {
	if src.Server.Name != "" {
		dest.Server.Name = src.Server.Name
	}
	if src.Server.Description != "" {
		dest.Server.Description = src.Server.Description
	}
	if src.Server.SecretKey != "" {
		dest.Server.SecretKey = src.Server.SecretKey
	}
	if src.Server.ServerID != "" {
		dest.Server.ServerID = src.Server.ServerID
	}

	if src.API.BaseURL != "" {
		dest.API.BaseURL = src.API.BaseURL
	}
	if src.API.APIKey != "" {
		dest.API.APIKey = src.API.APIKey
	}
	if src.API.Timeout != "" {
		dest.API.Timeout = src.API.Timeout
	}

	if src.WebSocket.Enabled {
		dest.WebSocket.Enabled = src.WebSocket.Enabled
	}
	if src.WebSocket.URL != "" {
		dest.WebSocket.URL = src.WebSocket.URL
	}
	if src.WebSocket.ReconnectInterval != "" {
		dest.WebSocket.ReconnectInterval = src.WebSocket.ReconnectInterval
	}
	if src.WebSocket.MaxReconnectAttempts != 0 {
		dest.WebSocket.MaxReconnectAttempts = src.WebSocket.MaxReconnectAttempts
	}
	if src.WebSocket.PingInterval != "" {
		dest.WebSocket.PingInterval = src.WebSocket.PingInterval
	}
	if src.WebSocket.WriteTimeout != "" {
		dest.WebSocket.WriteTimeout = src.WebSocket.WriteTimeout
	}
	if src.WebSocket.ReadTimeout != "" {
		dest.WebSocket.ReadTimeout = src.WebSocket.ReadTimeout
	}
	if src.WebSocket.HandshakeTimeout != "" {
		dest.WebSocket.HandshakeTimeout = src.WebSocket.HandshakeTimeout
	}
	if src.WebSocket.BufferSize != 0 {
		dest.WebSocket.BufferSize = src.WebSocket.BufferSize
	}
	if src.WebSocket.EnableCompression {
		dest.WebSocket.EnableCompression = src.WebSocket.EnableCompression
	}
	if src.WebSocket.MetricBufferSize != 0 {
		dest.WebSocket.MetricBufferSize = src.WebSocket.MetricBufferSize
	}
	if src.WebSocket.MetricBufferFlush != "" {
		dest.WebSocket.MetricBufferFlush = src.WebSocket.MetricBufferFlush
	}
	if src.WebSocket.CommandQueueSize != 0 {
		dest.WebSocket.CommandQueueSize = src.WebSocket.CommandQueueSize
	}
	if src.WebSocket.CommandTimeout != "" {
		dest.WebSocket.CommandTimeout = src.WebSocket.CommandTimeout
	}

	if src.Metrics.CPUUsage {
		dest.Metrics.CPUUsage = src.Metrics.CPUUsage
	}
	if src.Metrics.MemoryUsage {
		dest.Metrics.MemoryUsage = src.Metrics.MemoryUsage
	}
	if src.Metrics.DiskUsage {
		dest.Metrics.DiskUsage = src.Metrics.DiskUsage
	}
	if src.Metrics.CPUTemperature {
		dest.Metrics.CPUTemperature = src.Metrics.CPUTemperature
	}
	if src.Metrics.Interval != "" {
		dest.Metrics.Interval = src.Metrics.Interval
	}

	if src.Logging.Level != "" {
		dest.Logging.Level = src.Logging.Level
	}
	if src.Logging.File != "" {
		dest.Logging.File = src.Logging.File
	}
}

// getEnvironmentConfigPath returns the path for environment-specific config
func (cm *ConfigManager) getEnvironmentConfigPath() string {
	dir := filepath.Dir(cm.configPath)
	ext := filepath.Ext(cm.configPath)
	base := strings.TrimSuffix(filepath.Base(cm.configPath), ext)
	return filepath.Join(dir, fmt.Sprintf("%s.%s%s", base, cm.environment, ext))
}

// getProfileConfigPath returns the path for profile-specific config
func (cm *ConfigManager) getProfileConfigPath() string {
	dir := filepath.Dir(cm.configPath)
	ext := filepath.Ext(cm.configPath)
	base := strings.TrimSuffix(filepath.Base(cm.configPath), ext)
	return filepath.Join(dir, fmt.Sprintf("%s.%s%s", base, cm.profile, ext))
}

// validateConfig performs comprehensive configuration validation
func (cm *ConfigManager) validateConfig(config *AgentConfig) error {
	// Server configuration validation
	if config.Server.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if len(config.Server.Name) > 100 {
		return fmt.Errorf("server name too long (max 100 characters)")
	}
	if config.Server.SecretKey == "" {
		return fmt.Errorf("server secret key is required")
	}
	if len(config.Server.SecretKey) < 10 {
		return fmt.Errorf("server secret key too short (min 10 characters)")
	}

	// API configuration validation
	if config.API.BaseURL == "" {
		return fmt.Errorf("API base URL is required")
	}
	if !strings.HasPrefix(config.API.BaseURL, "http://") && !strings.HasPrefix(config.API.BaseURL, "https://") {
		return fmt.Errorf("API base URL must start with http:// or https://")
	}

	// WebSocket configuration validation
	if config.WebSocket.Enabled {
		if config.WebSocket.URL == "" {
			return fmt.Errorf("WebSocket URL is required when WebSocket is enabled")
		}
		if !strings.HasPrefix(config.WebSocket.URL, "ws://") && !strings.HasPrefix(config.WebSocket.URL, "wss://") {
			return fmt.Errorf("WebSocket URL must start with ws:// or wss://")
		}
		if config.WebSocket.MaxReconnectAttempts < 0 || config.WebSocket.MaxReconnectAttempts > 100 {
			return fmt.Errorf("WebSocket max reconnect attempts must be between 0 and 100")
		}
		if config.WebSocket.BufferSize < 64 || config.WebSocket.BufferSize > 10000 {
			return fmt.Errorf("WebSocket buffer size must be between 64 and 10000")
		}
	}

	// Metrics configuration validation
	if config.Metrics.Interval == "" {
		config.Metrics.Interval = "30s" // Default
	} else {
		if _, err := time.ParseDuration(config.Metrics.Interval); err != nil {
			return fmt.Errorf("invalid metrics interval format: %v", err)
		}
	}

	// Logging configuration validation
	validLogLevels := []string{"debug", "info", "warn", "error"}
	validLevel := false
	for _, level := range validLogLevels {
		if config.Logging.Level == level {
			validLevel = true
			break
		}
	}
	if !validLevel {
		return fmt.Errorf("invalid log level: %s (must be one of: %s)", config.Logging.Level, strings.Join(validLogLevels, ", "))
	}

	// Environment-specific validation
	if cm.environment == Production {
		if config.Logging.Level == "debug" {
			return fmt.Errorf("debug logging not recommended in production environment")
		}
		if config.WebSocket.Enabled && strings.HasPrefix(config.WebSocket.URL, "ws://") {
			return fmt.Errorf("unsecure WebSocket (ws://) not recommended in production environment")
		}
	}

	return nil
}

// setupWatcher sets up file system watcher for hot-reload
func (cm *ConfigManager) setupWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	cm.watcher = watcher

	// Watch main config file
	if err := cm.watcher.Add(cm.configPath); err != nil {
		return fmt.Errorf("failed to watch config file: %w", err)
	}

	// Watch environment-specific config file
	if cm.environment != "" {
		envConfigPath := cm.getEnvironmentConfigPath()
		if _, err := os.Stat(envConfigPath); err == nil {
			if err := cm.watcher.Add(envConfigPath); err != nil {
				cm.logger.WithError(err).Warnf("Failed to watch environment config: %s", envConfigPath)
			}
		}
	}

	// Watch profile-specific config file
	if cm.profile != "" {
		profileConfigPath := cm.getProfileConfigPath()
		if _, err := os.Stat(profileConfigPath); err == nil {
			if err := cm.watcher.Add(profileConfigPath); err != nil {
				cm.logger.WithError(err).Warnf("Failed to watch profile config: %s", profileConfigPath)
			}
		}
	}

	// Start watching goroutine
	go cm.watchConfig()

	cm.logger.Info("Config watcher enabled for hot-reload")
	return nil
}

// watchConfig watches for configuration file changes
func (cm *ConfigManager) watchConfig() {
	for {
		select {
		case event, ok := <-cm.watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				cm.logger.WithField("file", event.Name).Debug("Config file changed, reloading...")

				// Debounce rapid file changes
				time.Sleep(100 * time.Millisecond)

				if err := cm.loadConfig(); err != nil {
					cm.logger.WithError(err).Error("Failed to reload configuration")
					continue
				}

				// Call reload callbacks
				for _, callback := range cm.reloadCallbacks {
					if callback != nil {
						callback(cm.GetConfig())
					}
				}

				cm.logger.Info("Configuration reloaded successfully")
			}

		case err, ok := <-cm.watcher.Errors:
			if !ok {
				return
			}
			cm.logger.WithError(err).Error("Config watcher error")
		}
	}
}

// GetConfig returns the current configuration (thread-safe)
func (cm *ConfigManager) GetConfig() *AgentConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy to prevent external modifications
	configCopy := *cm.config
	return &configCopy
}

// ReloadConfig manually reloads the configuration
func (cm *ConfigManager) ReloadConfig() error {
	return cm.loadConfig()
}

// Close closes the config manager and cleans up resources
func (cm *ConfigManager) Close() error {
	if cm.watcher != nil {
		return cm.watcher.Close()
	}
	return nil
}

// AddReloadCallback adds a callback to be called on configuration reload
func (cm *ConfigManager) AddReloadCallback(callback func(*AgentConfig)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.reloadCallbacks = append(cm.reloadCallbacks, callback)
}

// GetEnvironment returns the current environment
func (cm *ConfigManager) GetEnvironment() Environment {
	return cm.environment
}

// GetProfile returns the current profile
func (cm *ConfigManager) GetProfile() string {
	return cm.profile
}
