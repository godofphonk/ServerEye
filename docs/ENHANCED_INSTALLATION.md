# Enhanced Installation Scripts

ServerEye теперь включает обновленные скрипты установки с поддержкой новой системы конфигурации.

## Скрипты

### 1. `install-agent.sh` - Production Installation
Основной скрипт для production установки.

### 2. `install-local.sh` - Local Development Installation
Скрипт для локальной разработки с использованием локально собранного бинарника.

## Новые возможности

### 🔧 Enhanced Configuration Features
- **Environment variable overrides** - Поддержка переопределения через переменные окружения
- **Hot-reload configuration changes** - Автоматическая перезагрузка конфигурации
- **Environment-specific configurations** - Конфигурации для разных окружений
- **Comprehensive validation** - Расширенная валидация конфигурации

### 📁 Структура файлов конфигурации

```
/etc/servereye/
├── config.yaml              # Основная конфигурация с environment variables
├── agent.env                 # Environment variables (создается установщиком)
└── local.env                 # Local overrides (опционально)
```

## Использование

### Production Installation

```bash
# Basic installation
sudo ./scripts/install-agent.sh

# With environment specification
sudo SERVEREYE_ENVIRONMENT=production ./scripts/install-agent.sh

# With custom backend URL
sudo SERVEREYE_BACKEND_URL=https://api.servereye.dev ./scripts/install-agent.sh
```

### Local Development Installation

```bash
# Build the agent first
make build-agent

# Install local build
sudo ./scripts/install-local.sh

# With custom environment
sudo SERVEREYE_ENVIRONMENT=development ./scripts/install-local.sh
```

## Environment Variables

### Основные переменные

```bash
# Server configuration
SERVEREYE_SERVER_KEY="your-secret-key"
SERVEREYE_SERVER_ID="server-id"

# API configuration  
SERVEREYE_API_URL="https://api.servereye.dev"
SERVEREYE_API_KEY="your-api-key"

# WebSocket configuration
SERVEREYE_WS_URL="wss://api.servereye.dev/ws"
SERVEREYE_WS_ENABLED="true"

# Metrics configuration
SERVEREYE_METRICS_INTERVAL="30s"
SERVEREYE_CPU_TEMPERATURE="true"

# Logging configuration
SERVEREYE_LOG_LEVEL="info"
SERVEREYE_LOG_FILE="/var/log/servereye/agent.log"

# Environment specification
SERVEREYE_ENVIRONMENT="production"
```

### Специфичные для разработки

```bash
# Local development overrides
SERVEREYE_WS_URL="ws://localhost:8080/ws"
SERVEREYE_LOG_LEVEL="debug"
SERVEREYE_METRICS_INTERVAL="10s"
SERVEREYE_ENVIRONMENT="development"
```

## Environment-Specific Configurations

### Production (`config.production.yaml`)
- Оптимизировано для production
- Enhanced security settings
- Performance tuning
- Стабильные интервалы метрик

### Development (`config.development.yaml`)
- Debug logging
- Local endpoints
- Быстрые интервалы для тестирования
- Verbose output

### Testing (`config.testing.yaml`)
- Минимальная функциональность
- Error-only logging
- Отключена телеметрия
- Быстрые интервалы

## Конфигурация systemd

### Enhanced Service Features

```ini
[Unit]
Description=ServerEye Agent - Server Monitoring Agent
After=network.target

[Service]
Type=simple
User=servereye
Group=servereye
WorkingDirectory=/opt/servereye
EnvironmentFile=/etc/servereye/agent.env
EnvironmentFile=-/etc/servereye/local.env
ExecStart=/opt/servereye/servereye-agent -config /etc/servereye/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/servereye /etc/servereye

[Install]
WantedBy=multi-user.target
```

### Environment Files Priority

1. `/etc/servereye/agent.env` - Создается установщиком
2. `/etc/servereye/local.env` - Local overrides (опционально)
3. Systemd environment variables

## Управление конфигурацией

### Редактирование конфигурации

```bash
# Редактировать основную конфигурацию
sudo nano /etc/servereye/config.yaml

# Создать local overrides
sudo nano /etc/servereye/local.env
```

### Перезагрузка конфигурации

```bash
# Перезапустить сервис для применения изменений
sudo systemctl restart servereye-agent

# Проверить статус
sudo systemctl status servereye-agent

# Просмотреть логи
sudo journalctl -u servereye-agent -f
```

### Environment Overrides без перезапуска

```bash
# Установить environment variable через systemd
sudo systemctl set-environment SERVEREYE_LOG_LEVEL=debug
sudo systemctl restart servereye-agent

# Или создать local.env файл
echo "SERVEREYE_LOG_LEVEL=debug" | sudo tee -a /etc/servereye/local.env
sudo systemctl restart servereye-agent
```

## Примеры использования

### 1. Production с custom backend

```bash
sudo SERVEREYE_BACKEND_URL=https://custom.api.com \
     SERVEREYE_API_KEY=custom-key \
     SERVEREYE_ENVIRONMENT=production \
     ./scripts/install-agent.sh
```

### 2. Development с hot-reload

```bash
# Build и установка
make build-agent
sudo ./scripts/install-local.sh

# Редактировать конфигурацию
sudo nano /etc/servereye/config.yaml

# Быстрый reload
sudo systemctl restart servereye-agent
```

### 3. Testing configuration

```bash
# Создать тестовую конфигурацию
sudo SERVEREYE_ENVIRONMENT=testing \
     SERVEREYE_LOG_LEVEL=error \
     SERVEREYE_METRICS_INTERVAL=1s \
     ./scripts/install-agent.sh
```

## Migration со старых версий

### Backward Compatibility

Старые конфигурации продолжают работать. Новая система автоматически:

1. **Сохраняет существующие конфигурации** при обновлении
2. **Добавляет environment variables** для гибкости
3. **Поддерживает старый формат** YAML

### Manual Migration

```bash
# Экспортировать существующую конфигурацию
sudo cp /etc/servereye/config.yaml /etc/servereye/config.yaml.backup

# Добавить environment variables
sudo nano /etc/servereye/local.env

# Перезапустить
sudo systemctl restart servereye-agent
```

## Troubleshooting

### Configuration Issues

```bash
# Проверить конфигурацию
sudo /opt/servereye/servereye-agent -config /etc/servereye/config.yaml -validate

# Проверить environment variables
systemctl show servereye-agent | grep SERVEREYE

# Просмотреть логи конфигурации
sudo journalctl -u servereye-agent | grep -i config
```

### Service Issues

```bash
# Проверить статус сервиса
sudo systemctl status servereye-agent

# Просмотреть логи
sudo journalctl -u servereye-agent -n 50

# Перезапустить с verbose логами
sudo SERVEREYE_LOG_LEVEL=debug systemctl restart servereye-agent
```

### Environment Variable Issues

```bash
# Проверить загруженные переменные
sudo systemctl show servereye-agent | grep Environment

# Тестировать переменные
sudo -u servereye env | grep SERVEREYE

# Проверить файлы окружения
sudo cat /etc/servereye/agent.env
sudo cat /etc/servereye/local.env
```

## Best Practices

### 1. Используйте Environment Variables для секретов
```bash
# Хорошо: использовать переменные окружения
secret_key: "${SERVEREYE_SERVER_KEY}"

# Плохо: хардкодить секреты
secret_key: "hardcoded-secret-key"
```

### 2. Создавайте Environment-Specific конфигурации
```bash
# Production
sudo SERVEREYE_ENVIRONMENT=production ./scripts/install-agent.sh

# Development  
sudo SERVEREYE_ENVIRONMENT=development ./scripts/install-local.sh
```

### 3. Используйте Local Overrides для разработки
```bash
# /etc/servereye/local.env
SERVEREYE_LOG_LEVEL=debug
SERVEREYE_METRICS_INTERVAL=5s
SERVEREYE_WS_URL=ws://localhost:8080/ws
```

### 4. Валидируйте конфигурацию перед применением
```bash
# Проверить синтаксис
sudo /opt/servereye/servereye-agent -config /etc/servereye/config.yaml -validate

# Проверить логи на ошибки валидации
sudo journalctl -u servereye-agent | grep -i error
```

Эти обновленные скрипты обеспечивают enterprise-level гибкость и надежность для ServerEye агента с полной поддержкой новой системы конфигурации.
