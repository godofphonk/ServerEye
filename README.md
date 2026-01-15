# ServerEye Agent

<div align="center">

![Go Report Card](https://goreportcard.com/badge/github.com/godofphonk/ServerEye)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)
![Release](https://img.shields.io/github/v/release/godofphonk/ServerEye?style=for-the-badge&logo=github)

**server monitoring**

[Installation](#-installation) • [Configuration](#-configuration) • [Metrics](#-metrics) • [Security](#-security)

</div>

## Overview

ServerEye Agent is a lightweight, high-performance monitoring agent designed for everyone from individual developers to large enterprises. It collects comprehensive system metrics including CPU usage and temperature, memory consumption, disk I/O, network statistics, and system information, transmitting them to the ServerEye API via secure WebSocket connections.

### Key Features

- 🚀 **Real-time Metrics** - CPU, memory, disk, network, and temperature monitoring
- � **Telegram Integration** - Monitor servers via @ServerEyeBot with instant notifications
- �🔐 **Secure Communication** - WebSocket with TLS and authentication
- ⚡ **High Performance** - Minimal resource footprint and efficient data transmission
- 🔧 **Flexible Configuration** - YAML-based config with hot reload support
- 📊 **Production Ready** - Structured logging, graceful shutdown, and reliable operation

## 🚀 Quick Start

### Prerequisites

- Linux (x86_64 or ARM64)
- Systemd (for service installation)

### Installation

#### 🔄 Automatic (Recommended)

```bash
wget -qO- https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install-agent.sh | sudo bash
```

#### 📦 Manual Download

```bash
# Download latest release
wget https://github.com/godofphonk/ServerEye/releases/download/v1.1.0/servereye-agent-linux-amd64
chmod +x servereye-agent-linux-amd64
sudo mv servereye-agent-linux-amd64 /usr/local/bin/servereye-agent

# Verify installation
servereye-agent --version
```

#### ✅ Verification

```bash
# Download checksums
wget https://github.com/godofphonk/ServerEye/releases/latest/download/checksums.txt

# Verify integrity
sha256sum -c checksums.txt
# Expected: servereye-agent-linux-amd64: OK
```


### 📱 Connect to Telegram Bot

```bash
After installation, connect your server to Telegram:
1. Find @ServerEyeBot in Telegram
2. Send: /start
3. Send: /add YOUR_SECRET_KEY(key_123123)

Your secret key is shown at the end of installation
Or find it in: /etc/servereye/config.yaml
```

### Manual Service Management

```bash
# Check service status
sudo systemctl status servereye-agent

# Restart service
sudo systemctl restart servereye-agent

# View logs
sudo journalctl -u servereye-agent -f

# Stop service
sudo systemctl stop servereye-agent
```

## ⚙️ Configuration

The agent configuration is stored in `/etc/servereye/config.yaml` (system) or `~/.servereye/config.yaml` (user).

### Basic Configuration

```yaml
server:
  name: "Production Server"
  description: "ServerEye monitored server"
  secret_key: "srv_your_generated_secret_key"
  server_id: "your-server-id"

api:
  base_url: "https://api.servereye.dev"
  api_key: "your-api-key"
  timeout: "30s"

websocket:
  enabled: true
  url: "wss://api.servereye.dev/ws"
  reconnect_interval: "5s"
  max_reconnect_attempts: 10
  ping_interval: "30s"
  write_timeout: "10s"
  read_timeout: "10s"
  handshake_timeout: "10s"
  buffer_size: 1000
  enable_compression: true
  metric_buffer_size: 100
  metric_buffer_flush: "30s"
  command_queue_size: 100
  command_timeout: "30s"

metrics:
  cpu_usage: true          # CPU utilization percentages (user, system, idle, total)
  memory_usage: true       # Memory usage statistics (used, available, cached, swap)
  disk_usage: true         # Disk space usage and I/O statistics
  cpu_temperature: true    # CPU temperature from thermal sensors
  interval: "30s"          # Metrics collection interval

logging:
  level: "info"
  file: "/var/log/servereye/agent.log"

# Enhanced features
features:
  auto_updates: false      # Automatic agent updates (disabled by default)
  telemetry: true         # Anonymous usage telemetry and statistics
  remote_commands: true   # Allow remote command execution via API
  alerting: true         # Enable alert notifications

security:
  allowed_ips: []          # Whitelist of allowed IP addresses
  rate_limit_per_sec: 10   # Rate limit requests per second
  max_connections: 100     # Maximum concurrent connections

performance:
  worker_count: 4          # Number of worker goroutines for processing
  queue_size: 1000         # Size of internal queues
  batch_size: 100          # Batch size for metric processing
  flush_interval: "30s"    # Interval for flushing metrics
  connection_timeout: "10s" # Connection timeout for API calls
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVEREYE_API_URL` | API endpoint for registration | - |
| `SERVEREYE_LOG_LEVEL` | Logging level | `info` |
| `SERVEREYE_CONFIG_PATH` | Custom config path | `/etc/servereye/config.yaml` |

## 📊 Metrics

### System Metrics

- **CPU**: Usage, load average, frequency, temperature
- **Memory**: Usage, available, cached, swap
- **Disk**: Usage, I/O stats, mount points
- **Network**: Interface stats, connections, bandwidth

### Temperature Sensors

- CPU thermal zones
- Coretemp sensors
- Hardware monitoring chips

## 🔐 Security

### Authentication

- **Secret Key**: Unique 32-character key per agent
- **WebSocket Auth**: Token-based authentication
- **API Registration**: Automatic key registration

### Data Protection

- Encrypted WebSocket connections (wss://)
- Token-based authentication
- API key validation
- Rate limiting at application level
- No sensitive data in logs
- Configurable data retention
- Secure key storage

## 🛠️ Management

### Service Management

```bash
# Start/Stop/Restart
sudo systemctl start servereye-agent
sudo systemctl stop servereye-agent
sudo systemctl restart servereye-agent

# Enable/Disable
sudo systemctl enable servereye-agent
sudo systemctl disable servereye-agent

# View logs
sudo journalctl -u servereye-agent -f
```

### Configuration Updates

```bash
# Test configuration
servereye-agent --config /path/to/config.yaml --log-level=debug

# Reload without restart (hot reload)
sudo systemctl reload servereye-agent
```

### Uninstallation

```bash
# Complete removal (recommended)
sudo systemctl stop servereye-agent
sudo systemctl disable servereye-agent
sudo rm -f /etc/systemd/system/servereye-agent.service
sudo systemctl daemon-reload
sudo rm -rf /opt/servereye /etc/servereye /var/log/servereye /home/servereye
sudo userdel servereye 2>/dev/null || true
```

##  Performance

### Resource Usage

| Metric | Typical Usage | Maximum |
|--------|---------------|---------|
| CPU | < 1% | < 2% |
| Memory | < 50MB | < 100MB |
| Network | < 20KB/min | < 250KB/min |
| Disk | Logs only | Logs grow over time* |

### Optimization

- Efficient metric collection with configurable intervals
- Batched data transmission
- Connection pooling and reuse
- Minimal system call overhead

> **Note**: Disk usage is only for log files. Logs grow over time as the agent writes to `/var/log/servereye/agent.log` without rotation. Monitor log file size and clean up periodically.

## 🚨 Troubleshooting

### Common Issues

#### Agent not starting

```bash
# Check configuration
sudo servereye-agent --config /etc/servereye/config.yaml --log-level=debug

# Verify permissions
ls -la /etc/servereye/
ls -la /var/log/servereye/
```



#### High resource usage

```bash
# Monitor agent resources
top -p $(pgrep servereye-agent)
iotop -p $(pgrep servereye-agent)

# Check metrics collection frequency
grep "interval" /etc/servereye/config.yaml
```

### Debug Mode

```bash
# Run with debug logging
servereye-agent --log-level=debug

# Enable detailed metrics collection
servereye-agent --log-level=debug --config /path/to/debug-config.yaml
```

### Log Analysis

```bash
# View recent logs
sudo journalctl -u servereye-agent --since "1 hour ago"

# Filter by log level
sudo journalctl -u servereye-agent -p err

# Follow logs in real-time
sudo journalctl -u servereye-agent -f

# Check log file directly
sudo tail -f /var/log/servereye/agent.log
```

## 📱 Telegram Bot Integration

The easiest way to monitor your server and view logs is through the **ServerEye Telegram Bot**.

### Setup

1. **Find the bot**: Search for `@ServerEyeBot` in Telegram
2. **Start the bot**: Send `/start` command
3. **Add your server**: Send `/add YOUR_SECRET_KEY`

> **Your secret key** is displayed during installation and can be found in `/etc/servereye/config.yaml`

### Available Commands

| Command | Description |
|---------|-------------|
| `/temp` | Get CPU temperature |
| `/memory` | Get memory usage statistics |
| `/disk` | Get disk space and I/O stats |
| `/status` | Get complete server status |
| `/logs` | View recent agent logs* |
| `/help` | Show all available commands |

### Real-time Monitoring

The Telegram bot provides:
- **📊 Live metrics** - Real-time system statistics
- **📋 Log viewing** - Recent agent logs and errors
- **🚨 Alerts** - Automatic notifications for issues
- **📈 History** - Historical performance data
- **⚡ Instant commands** - Quick server management

### Benefits

- **No SSH required** - Monitor from anywhere
- **Mobile friendly** - Works on Telegram mobile/desktop
- **Real-time updates** - Instant metric notifications
- **Easy troubleshooting** - Quick access to logs and errors
- **Multiple servers** - Monitor all your servers in one place

> **Note**: The `/logs` command shows the most recent log entries. For complete log analysis, use `sudo journalctl -u servereye-agent -f` on the server.

## 📚 API Reference

### Command Line Options

| Option | Description |
|--------|-------------|
| `--config` | Path to configuration file |
| `--log-level` | Logging level (debug, info, warn, error) |
| `--install` | Install and configure agent |
| `--uninstall` | Complete agent removal |
| `--version` | Show version information |

### Configuration Schema

See [Configuration Reference](docs/configuration.md) for detailed configuration options.


## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.


---

<div align="center">

[![ServerEye](https://img.shields.io/badge/ServerEye-blue?style=flat-square)](https://servereye.com)

</div>
