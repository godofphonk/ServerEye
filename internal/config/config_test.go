package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAgentConfig_Valid(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "agent.yaml")

	validConfig := `
server:
  name: "TestServer"
  description: "Test server description"
  secret_key: "srv_0123456789abcdef0123456789abcdef"

api:
  base_url: "https://api.example.com"
  timeout: "30s"

metrics:
  cpu_temperature: true
  interval: "30s"

logging:
  level: "info"
  file: "/var/log/agent.log"
`

	if err := os.WriteFile(configPath, []byte(validConfig), 0600); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	config, err := LoadAgentConfig(configPath)
	if err != nil {
		t.Fatalf("LoadAgentConfig() error = %v", err)
	}

	// Verify config was loaded correctly
	if config.Server.Name != "TestServer" {
		t.Errorf("Server.Name = %v, want TestServer", config.Server.Name)
	}
	if config.Server.SecretKey != "srv_0123456789abcdef0123456789abcdef" {
		t.Errorf("Server.SecretKey = %v, want srv_0123456789abcdef0123456789abcdef", config.Server.SecretKey)
	}
	if config.API.BaseURL != "https://api.example.com" {
		t.Errorf("API.BaseURL = %v, want https://api.example.com", config.API.BaseURL)
	}
	if !config.Metrics.CPUTemperature {
		t.Error("Metrics.CPUTemperature should be true")
	}
}

func TestLoadAgentConfig_WithRedis(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "agent.yaml")

	validConfig := `
server:
  name: "TestServer"
  secret_key: "srv_abc123def456abc123def456abc12345"

redis:
  address: "localhost:6379"
  password: ""
  db: 0

metrics:
  cpu_temperature: false
  interval: "60s"

logging:
  level: "debug"
  file: "/var/log/agent.log"
`

	if err := os.WriteFile(configPath, []byte(validConfig), 0600); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	config, err := LoadAgentConfig(configPath)
	if err != nil {
		t.Fatalf("LoadAgentConfig() error = %v", err)
	}

	if config.Redis.Address != "localhost:6379" {
		t.Errorf("Redis.Address = %v, want localhost:6379", config.Redis.Address)
	}
}

func TestLoadAgentConfig_MissingFile(t *testing.T) {
	_, err := LoadAgentConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for missing config file")
	}
}

func TestLoadAgentConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")

	invalidYAML := `
server:
  name: "Test
  this is invalid yaml
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	_, err := LoadAgentConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestAgentConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  AgentConfig
		wantErr bool
	}{
		{
			name: "valid config with API",
			config: AgentConfig{
				Server: ServerConfig{
					Name:      "TestServer",
					SecretKey: "srv_test123",
				},
				API: APIConfig{
					BaseURL: "https://api.example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with Redis",
			config: AgentConfig{
				Server: ServerConfig{
					Name:      "TestServer",
					SecretKey: "srv_test123",
				},
				Redis: RedisConfig{
					Address: "localhost:6379",
				},
			},
			wantErr: false,
		},
		{
			name: "missing server name",
			config: AgentConfig{
				Server: ServerConfig{
					SecretKey: "srv_test123",
				},
				API: APIConfig{
					BaseURL: "https://api.example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "missing secret key",
			config: AgentConfig{
				Server: ServerConfig{
					Name: "TestServer",
				},
				API: APIConfig{
					BaseURL: "https://api.example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "missing both Redis and API",
			config: AgentConfig{
				Server: ServerConfig{
					Name:      "TestServer",
					SecretKey: "srv_test123",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadBotConfig_Valid(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bot.yaml")

	validConfig := `
telegram:
  token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"

redis:
  address: "localhost:6379"
  password: ""
  db: 0

database:
  url: "postgres://user:pass@localhost/dbname"

logging:
  level: "info"
  file: "/var/log/bot.log"
`

	if err := os.WriteFile(configPath, []byte(validConfig), 0600); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	config, err := LoadBotConfig(configPath)
	if err != nil {
		t.Fatalf("LoadBotConfig() error = %v", err)
	}

	if config.Telegram.Token != "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11" {
		t.Errorf("Telegram.Token = %v", config.Telegram.Token)
	}
	if config.Redis.Address != "localhost:6379" {
		t.Errorf("Redis.Address = %v, want localhost:6379", config.Redis.Address)
	}
	if config.Database.URL != "postgres://user:pass@localhost/dbname" {
		t.Errorf("Database.URL = %v", config.Database.URL)
	}
}

func TestBotConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  BotConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: BotConfig{
				Telegram: TelegramConfig{Token: "test_token"},
				Redis:    RedisConfig{Address: "localhost:6379"},
				Database: DatabaseConfig{URL: "postgres://localhost/db"},
			},
			wantErr: false,
		},
		{
			name: "missing telegram token",
			config: BotConfig{
				Redis:    RedisConfig{Address: "localhost:6379"},
				Database: DatabaseConfig{URL: "postgres://localhost/db"},
			},
			wantErr: true,
		},
		{
			name: "missing redis address",
			config: BotConfig{
				Telegram: TelegramConfig{Token: "test_token"},
				Database: DatabaseConfig{URL: "postgres://localhost/db"},
			},
			wantErr: true,
		},
		{
			name: "missing database url",
			config: BotConfig{
				Telegram: TelegramConfig{Token: "test_token"},
				Redis:    RedisConfig{Address: "localhost:6379"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadBotConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")

	invalidYAML := `
telegram:
  token: "broken
redis:
  - invalid structure
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	_, err := LoadBotConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}
