# Configuration Reference

This document provides detailed configuration options for ServerEye Agent.

## Configuration File Location

The agent reads configuration from:
- `/etc/servereye/config.yaml` (system-wide installation)
- `~/.servereye/config.yaml` (user installation)
- Custom path specified with `--config` flag

## Configuration Schema

```yaml
server:
  name: "Production Server"              # Server display name
  description: "ServerEye monitored server"  # Server description
  secret_key: "your-secret-key-here"    # Unique server identifier

api:
  base_url: "https://api.servereye.dev"  # API base URL
  timeout: "30s"                         # API request timeout
  retry_attempts: 3                      # Number of retry attempts

websocket:
  enabled: true                          # Enable WebSocket communication
  url: "wss://api.servereye.dev/ws"      # WebSocket URL
  reconnect_interval: "5s"                # Reconnection delay
  max_reconnect_attempts: 10              # Max reconnection attempts
  ping_interval: "30s"                    # WebSocket ping interval
  write_timeout: "10s"                   # WebSocket write timeout
  read_timeout: "10s"                    # WebSocket read timeout
  handshake_timeout: "10s"               # WebSocket handshake timeout
  buffer_size: 1000                      # WebSocket buffer size
  enable_compression: true               # Enable WebSocket compression
  metric_buffer_size: 100                # Metric buffer size
  metric_buffer_flush: "30s"             # Metric buffer flush interval
  command_queue_size: 100               # Command queue size
  command_timeout: "30s"                 # Command processing timeout

metrics:
  cpu_usage: true                        # CPU usage metrics
  memory_usage: true                     # Memory usage statistics
  disk_usage: true                       # Disk space usage and I/O
  cpu_temperature: true                  # CPU temperature from sensors
  interval: "30s"                        # Metrics collection interval
  network_interfaces:                    # Network interfaces to monitor
    - "eth0"
    - "wlan0"

logging:
  level: "info"                          # Logging level: debug, info, warn, error
  file: "/var/log/servereye/agent.log"   # Log file path

# Enhanced features
features:
  auto_updates: false                    # Automatic agent updates
  telemetry: true                        # Anonymous usage telemetry
  remote_commands: true                  # Allow remote command execution
  alerting: true                         # Enable alert notifications

security:
  allowed_ips: []                        # Whitelist of allowed IPs
  rate_limit_per_sec: 100                # Rate limit per second
  max_connections: 100                   # Maximum concurrent connections

performance:
  worker_count: 4                        # Number of worker goroutines
  queue_size: 1000                       # Size of internal queues
  batch_size: 100                        # Batch size for metric processing
  flush_interval: "30s"                   # Interval for flushing metrics
  connection_timeout: "10s"              # Connection timeout for API calls
```

## Environment Variables

All configuration options can be overridden with environment variables using the `SERVEREYE_` prefix:

```bash
# Server configuration
export SERVEREYE_SERVER_NAME="My Server"
export SERVEREYE_SERVER_SECRET_KEY="my-secret-key"

# API configuration
export SERVEREYE_API_BASE_URL="https://api.servereye.dev"
export SERVEREYE_API_TIMEOUT="30s"

# WebSocket configuration
export SERVEREYE_WS_ENABLED="true"
export SERVEREYE_WS_URL="wss://api.servereye.dev/ws"

# Metrics configuration
export SERVEREYE_METRICS_INTERVAL="30s"
export SERVEREYE_CPU_TEMPERATURE="true"

# Logging configuration
export SERVEREYE_LOG_LEVEL="info"
export SERVEREYE_LOG_FILE="/var/log/servereye/agent.log"

# Performance configuration
export SERVEREYE_WORKER_COUNT="4"
export SERVEREYE_QUEUE_SIZE="1000"
```

## Configuration Sections

### Server Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | "Production Server" | Server display name |
| `description` | string | "ServerEye monitored server" | Server description |
| `secret_key` | string | required | Unique server identifier |

### API Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `base_url` | string | "https://api.servereye.dev" | API base URL |
| `timeout` | duration | "30s" | API request timeout |
| `retry_attempts` | int | 3 | Number of retry attempts |

### WebSocket Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | true | Enable WebSocket communication |
| `url` | string | "wss://api.servereye.dev/ws" | WebSocket URL |
| `reconnect_interval` | duration | "5s" | Reconnection delay |
| `max_reconnect_attempts` | int | 10 | Max reconnection attempts |
| `ping_interval` | duration | "30s" | WebSocket ping interval |
| `write_timeout` | duration | "10s" | WebSocket write timeout |
| `read_timeout` | duration | "10s" | WebSocket read timeout |
| `handshake_timeout` | duration | "10s" | WebSocket handshake timeout |
| `buffer_size` | int | 1000 | WebSocket buffer size |
| `enable_compression` | bool | true | Enable WebSocket compression |
| `metric_buffer_size` | int | 100 | Metric buffer size |
| `metric_buffer_flush` | duration | "30s" | Metric buffer flush interval |
| `command_queue_size` | int | 100 | Command queue size |
| `command_timeout` | duration | "30s" | Command processing timeout |

### Metrics Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `cpu_usage` | bool | true | Enable CPU usage metrics |
| `memory_usage` | bool | true | Enable memory usage metrics |
| `disk_usage` | bool | true | Enable disk usage metrics |
| `cpu_temperature` | bool | true | Enable CPU temperature metrics |
| `interval` | duration | "30s" | Metrics collection interval |
| `network_interfaces` | []string | ["eth0"] | Network interfaces to monitor |

### Logging Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `level` | string | "info" | Logging level (debug, info, warn, error) |
| `file` | string | "/var/log/servereye/agent.log" | Log file path |

### Features Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `auto_updates` | bool | false | Enable automatic agent updates |
| `telemetry` | bool | true | Enable anonymous usage telemetry |
| `remote_commands` | bool | true | Allow remote command execution |
| `alerting` | bool | true | Enable alert notifications |

### Security Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `allowed_ips` | []string | [] | Whitelist of allowed IPs |
| `rate_limit_per_sec` | int | 100 | Rate limit per second |
| `max_connections` | int | 100 | Maximum concurrent connections |

### Performance Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `worker_count` | int | 4 | Number of worker goroutines |
| `queue_size` | int | 1000 | Size of internal queues |
| `batch_size` | int | 100 | Batch size for metric processing |
| `flush_interval` | duration | "30s" | Interval for flushing metrics |
| `connection_timeout` | duration | "10s" | Connection timeout for API calls |

## Configuration Validation

The agent validates configuration on startup and will:

1. **Check required fields**: `secret_key` is required
2. **Validate URLs**: API and WebSocket URLs must be valid
3. **Validate durations**: Time intervals must be valid duration strings
4. **Validate ranges**: Numeric values must be within reasonable ranges
5. **Check file permissions**: Log file must be writable

## Hot Reload

The agent supports configuration hot reload when running as a service:

```bash
# Reload configuration without restart
sudo systemctl reload servereye-agent
```

The agent watches the configuration file and automatically applies changes to:
- Logging level and file
- Metrics collection intervals
- Performance tuning parameters
- WebSocket settings

Changes to `secret_key` or `server.name` require a full restart.

## Environment-Specific Configurations

### Development

```yaml
logging:
  level: "debug"
  file: "./logs/agent.log"

performance:
  worker_count: 2
  queue_size: 500
  batch_size: 50
  flush_interval: "10s"
```

### Production

```yaml
logging:
  level: "warn"
  file: "/var/log/servereye/agent.log"

performance:
  worker_count: 8
  queue_size: 2000
  batch_size: 200
  flush_interval: "15s"

security:
  max_connections: 200
```

### Testing

```yaml
logging:
  level: "error"
  file: "/tmp/test-agent.log"

performance:
  worker_count: 1
  queue_size: 100
  batch_size: 10
  flush_interval: "1s"
```

## Troubleshooting

### Common Configuration Issues

1. **Invalid secret key**: Ensure `secret_key` is unique and matches API registration
2. **File permissions**: Check that the agent can write to log file path
3. **Network connectivity**: Verify WebSocket URL is accessible
4. **Invalid intervals**: Use Go duration format (e.g., "30s", "1m", "1h")

### Debug Configuration

```bash
# Test configuration syntax
servereye-agent --config /etc/servereye/config.yaml --log-level=debug

# Show effective configuration
servereye-agent --config /etc/servereye/config.yaml --log-level=debug --dry-run
```
