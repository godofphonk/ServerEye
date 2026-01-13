# Enhanced Configuration System

ServerEye теперь поддерживает улучшенную систему конфигурации с множественными форматами, environment override, hot-reload и comprehensive validation.

## Features

### 🚀 Multiple Format Support
- **YAML** - Основной формат, человеко-читаемый
- **JSON** - Для программной генерации
- **TOML** - Альтернативный читаемый формат

### 🔄 Hot-Reload
Автоматическая перезагрузка конфигурации при изменении файлов:
- Мониторинг основных конфигурационных файлов
- Environment-specific файлы
- Profile-specific файлы
- Graceful reload с уведомлениями

### 🌍 Environment Override
Поддержка environment-specific конфигураций:
- `config.yaml` - Базовая конфигурация
- `config.production.yaml` - Production override
- `config.development.yaml` - Development override
- `config.testing.yaml` - Testing override

### 🔧 Environment Variables
Полный override через environment variables:
```bash
# Server configuration
export SERVEREYE_SERVER_NAME="prod-server-01"
export SERVEREYE_SERVER_KEY="your-secret-key"

# API configuration
export SERVEREYE_API_URL="https://api.servereye.dev"
export SERVEREYE_API_KEY="your-api-key"

# WebSocket configuration
export SERVEREYE_WS_URL="wss://api.servereye.dev/ws"
export SERVEREYE_WS_ENABLED="true"

# Metrics configuration
export SERVEREYE_METRICS_INTERVAL="30s"
export SERVEREYE_CPU_TEMPERATURE="true"

# Logging configuration
export SERVEREYE_LOG_LEVEL="info"
export SERVEREYE_LOG_FILE="/var/log/servereye/agent.log"
```

### ✅ Comprehensive Validation
Валидация конфигурации с environment-specific правилами:
- Базовая валидация полей
- Environment-specific проверки
- Production security requirements
- Development performance warnings

## Usage

### Basic Usage
```go
import "github.com/godofphonk/ServerEye/internal/config"

// Create configuration provider
provider, err := config.NewConfigBuilder().
    WithConfigPath("/etc/servereye/config.yaml").
    WithEnvironment(config.Production).
    WithHotReload(true).
    Build()
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

// Get configuration
serverConfig := provider.GetServerConfig()
apiConfig := provider.GetAPIConfig()
```

### Factory Methods
```go
factory := config.NewConfigFactory()

// Production configuration
prodProvider, err := factory.CreateProductionConfig(
    "/etc/servereye/config.yaml", 
    logger,
)

// Development configuration
devProvider, err := factory.CreateDevelopmentConfig(
    "/etc/servereye/config.yaml", 
    logger,
)

// Testing configuration
testProvider, err := factory.CreateTestingConfig(
    "/etc/servereye/config.yaml", 
    logger,
)
```

### Configuration Reload Callbacks
```go
provider.AddReloadCallback(func(newConfig *config.AgentConfig) {
    log.Info("Configuration reloaded!")
    // Reinitialize components if needed
})
```

## Configuration Structure

### Enhanced Fields
Новые поля в конфигурации:

```yaml
# Features control
features:
  auto_updates: false
  telemetry: true
  remote_commands: true
  alerting: true
  docker_monitoring: true

# Security settings
security:
  enable_tls: false
  tls_cert_file: ""
  tls_key_file: ""
  allowed_ips: []
  rate_limit_per_sec: 10
  max_connections: 100

# Performance tuning
performance:
  worker_count: 4
  queue_size: 1000
  batch_size: 100
  flush_interval: "30s"
  connection_timeout: "10s"
```

## Environment-Specific Behavior

### Production
- Запрещен debug logging
- Требуется secure WebSocket (wss://)
- Обязательные TLS сертификаты при включенном TLS
- Enhanced security settings

### Development
- Разрешен debug logging
- Поддержка insecure WebSocket (ws://)
- Ограничения на worker count для производительности
- Verbose logging

### Testing
- Отключена телеметрия
- Минимальные интервалы метрик
- Error-only logging
- Отключены ненужные features

## Migration

### From Legacy Configuration
```go
migration := &config.ConfigMigration{}

// Migrate legacy config to enhanced
enhanced := migration.MigrateFromLegacy(legacyConfig)

// Export enhanced to legacy format
legacy := migration.ExportToLegacy(enhancedConfig)
```

### Backward Compatibility
```go
// Load with enhanced features but maintain compatibility
config, err := config.LoadAgentConfigWithEnhancements("/path/to/config.yaml")
```

## File Structure

```
/etc/servereye/
├── config.yaml              # Base configuration
├── config.production.yaml    # Production overrides
├── config.development.yaml   # Development overrides
├── config.testing.yaml       # Testing overrides
└── config.dev.yaml          # Profile-specific overrides
```

## Validation Rules

### Required Fields
- `server.name` - Имя сервера (1-100 символов)
- `server.secret_key` - Секретный ключ (минимум 10 символов)
- `api.base_url` - URL API (http:// или https://)
- `logging.level` - Уровень логирования (debug/info/warn/error)

### Optional Fields with Validation
- `websocket.url` - WebSocket URL (ws:// или wss://)
- `metrics.interval` - Интервал сбора метрик (valid duration)
- `performance.worker_count` - Количество workers (1-20)

### Environment-Specific Validation
Production environment:
- Debug logging запрещен
- Insecure WebSocket запрещен
- TLS требует сертификатов

Development environment:
- Worker count ограничен (≤10)
- Рекомендуется debug logging

Testing environment:
- Телеметрия должна быть отключена
- Минимальные интервалы

## Best Practices

### 1. Use Environment Variables for Secrets
```yaml
server:
  secret_key: "${SERVEREYE_SERVER_KEY}"
api:
  api_key: "${SERVEREYE_API_KEY}"
```

### 2. Environment-Specific Configurations
Разделяйте конфигурации по окружениям:
- Базовая конфигурация в `config.yaml`
- Environment-specific overrides
- Profile-specific настройки

### 3. Enable Hot-Reload in Development
```go
provider, err := config.NewConfigBuilder().
    WithConfigPath("/etc/servereye/config.yaml").
    WithEnvironment(config.Development).
    WithHotReload(true).  // Enable in development
    Build()
```

### 4. Use Factory Methods for Standard Setups
```go
// Production with security
prodProvider, err := factory.CreateProductionConfig(configPath, logger)

// Development with debugging
devProvider, err := factory.CreateDevelopmentConfig(configPath, logger)
```

### 5. Validate Configuration on Startup
```go
if err := provider.Validate(); err != nil {
    log.Fatalf("Configuration validation failed: %v", err)
}
```

## Examples

### Complete Production Setup
```bash
# Environment variables
export SERVEREYE_SERVER_KEY="prod-secure-key-12345678901234567890"
export SERVEREYE_API_KEY="prod-api-key-12345678901234567890"

# Configuration files
/etc/servereye/config.yaml              # Base config
/etc/servereye/config.production.yaml    # Production overrides

# Start agent
./servereye-agent --config /etc/servereye/config.yaml
```

### Development with Hot-Reload
```bash
# Development configuration
export SERVEREYE_SERVER_KEY="dev-key-12345678901234567890"
export SERVEREYE_LOG_LEVEL="debug"

# Start with hot-reload enabled
./servereye-agent --config ./configs/config.development.yaml

# Edit configuration files and see automatic reload
vim ./configs/config.development.yaml
```

## Troubleshooting

### Configuration Not Loading
1. Проверьте права доступа к файлам
2. Валидируйте YAML/JSON/TOML синтаксис
3. Проверьте environment variables
4. Используйте `provider.Validate()` для диагностики

### Hot-Reload Not Working
1. Убедитесь что hot-reload включен
2. Проверьте что файлы отслеживаются
3. Проверьте логи для ошибок watcher

### Validation Errors
1. Проверьте required поля
2. Убедитесь что форматы данных корректны
3. Проверьте environment-specific правила
4. Используйте详细的 error messages

Эта улучшенная система конфигурации обеспечивает enterprise-level гибкость, безопасность и надежность для ServerEye агента.
