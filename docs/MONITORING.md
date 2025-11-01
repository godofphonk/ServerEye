# 📊 Monitoring & Observability

This document describes ServerEye's built-in monitoring capabilities, metrics collection, and observability features.

## Table of Contents

- [System Metrics](#system-metrics)
- [Application Metrics](#application-metrics)
- [Health Checks](#health-checks)
- [Logging](#logging)
- [Alerting](#alerting)
- [Troubleshooting](#troubleshooting)

## System Metrics

### CPU Temperature

**Collection Method:**
- Read from `/sys/class/thermal/thermal_zone*/temp`
- Supports multiple thermal zones
- Updates every 30 seconds (configurable)

**Metrics:**
- Current temperature per zone (°C)
- Average across all zones
- Maximum temperature
- Thermal throttling status

**Example Response:**
```
🌡️ CPU Temperature

Zone 0: 45°C
Zone 1: 47°C
Zone 2: 43°C

Average: 45°C
Status: ✅ Normal
```

**Alert Thresholds:**
- Normal: < 70°C
- Warning: 70-85°C
- Critical: > 85°C

### Memory Usage

**Collection Method:**
- Uses `gopsutil` library
- Real-time memory statistics
- Includes swap information

**Metrics:**
- Total RAM (GB)
- Used RAM (GB and %)
- Available RAM (GB)
- Free RAM (GB)
- Buffers/Cache (MB)
- Swap usage

**Example Response:**
```
💾 Memory Usage

Total: 16.0 GB
Used: 8.5 GB (53%)
Available: 7.5 GB
Free: 2.3 GB

Buffers: 512 MB
Cached: 4.2 GB
```

**Alert Thresholds:**
- Normal: < 80%
- Warning: 80-90%
- Critical: > 90%

### Disk Usage

**Collection Method:**
- `df` command output parsing
- All mounted filesystems
- Excludes tmpfs, devtmpfs, and system mounts

**Metrics:**
- Filesystem type
- Mount point
- Total size
- Used space (GB and %)
- Available space (GB)
- Inode usage

**Example Response:**
```
💿 Disk Usage

/ (ext4)
  Total: 500 GB
  Used: 320 GB (64%)
  Available: 180 GB

/home (ext4)
  Total: 1000 GB
  Used: 650 GB (65%)
  Available: 350 GB
```

**Alert Thresholds:**
- Normal: < 80%
- Warning: 80-90%
- Critical: > 90%

### System Uptime

**Collection Method:**
- Read from `/proc/uptime`
- Boot time from system API

**Metrics:**
- Uptime (days, hours, minutes)
- Boot time (timestamp)
- Last reboot reason (if available)

**Example Response:**
```
⏱️ System Uptime

Uptime: 45 days, 12 hours, 34 minutes
Boot Time: 2024-09-15 08:23:41
Last Reboot: Normal shutdown
```

### Process Information

**Collection Method:**
- `ps` command with custom format
- Sorted by CPU or memory usage

**Metrics:**
- Process ID (PID)
- Process name
- CPU usage (%)
- Memory usage (MB and %)
- Uptime
- User

**Example Response:**
```
📊 Top Processes (by CPU)

1. docker (PID 1234)
   CPU: 45% | Memory: 2.5 GB

2. postgres (PID 5678)
   CPU: 12% | Memory: 1.8 GB

3. redis (PID 9012)
   CPU: 8% | Memory: 512 MB
```

### Docker Containers

**Collection Method:**
- Docker CLI via `docker ps -a`
- JSON format parsing

**Metrics:**
- Container ID (short)
- Container name
- Image
- Status (running, stopped, etc.)
- State (up, exited, restarting)
- Ports
- Uptime

**Example Response:**
```
🐳 Docker Containers

✅ nginx (b4c5d6e7f8g9)
   Image: nginx:latest
   Status: Up 2 days
   Ports: 0.0.0.0:80->80/tcp

⏸️ mysql (a1b2c3d4e5f6)
   Image: mysql:8.0
   Status: Exited (0) 1 hour ago
```

## Application Metrics

### Bot Metrics

**Collected Metrics:**

```go
type BotMetrics struct {
    // Command metrics
    CommandsReceived   int64
    CommandsSuccessful int64
    CommandsFailed     int64
    
    // User metrics
    ActiveUsers        int64
    TotalUsers         int64
    
    // Server metrics
    RegisteredServers  int64
    ActiveServers      int64
    
    // Performance metrics
    AverageResponseTime time.Duration
    P95ResponseTime    time.Duration
    P99ResponseTime    time.Duration
    
    // Error metrics
    ErrorRate          float64
    TimeoutRate        float64
}
```

**Endpoints:**

```bash
# Prometheus format
curl http://localhost:8080/metrics

# JSON format
curl http://localhost:8080/api/metrics
```

**Sample Output:**
```
# HELP servereye_commands_total Total commands received
# TYPE servereye_commands_total counter
servereye_commands_total{status="success"} 1543
servereye_commands_total{status="failed"} 12

# HELP servereye_response_time_seconds Command response time
# TYPE servereye_response_time_seconds histogram
servereye_response_time_seconds_bucket{le="0.1"} 1234
servereye_response_time_seconds_bucket{le="0.5"} 1523
servereye_response_time_seconds_bucket{le="1.0"} 1543
```

### Agent Metrics

**Collected Metrics:**

```go
type AgentMetrics struct {
    // Connection metrics
    LastHeartbeat      time.Time
    HeartbeatInterval  time.Duration
    ConnectionUptime   time.Duration
    
    // Command metrics
    CommandsProcessed  int64
    CommandsSucceeded  int64
    CommandsFailed     int64
    
    // Performance metrics
    AverageProcessTime time.Duration
    MetricsCollected   int64
}
```

**Viewing Metrics:**

```bash
# Agent logs
sudo journalctl -u servereye-agent -f

# Heartbeat status
grep "heartbeat" /var/log/servereye/agent.log
```

## Health Checks

### Bot Health Endpoint

```bash
GET /health
```

**Response (Healthy):**
```json
{
  "status": "healthy",
  "timestamp": "2024-11-01T15:30:00Z",
  "checks": {
    "database": "ok",
    "redis": "ok",
    "telegram": "ok"
  },
  "uptime": "45d12h34m",
  "version": "1.0.0"
}
```

**Response (Unhealthy):**
```json
{
  "status": "unhealthy",
  "timestamp": "2024-11-01T15:30:00Z",
  "checks": {
    "database": "ok",
    "redis": "error",
    "telegram": "ok"
  },
  "errors": [
    "Redis connection timeout"
  ],
  "uptime": "45d12h34m",
  "version": "1.0.0"
}
```

### Agent Health

**Heartbeat Mechanism:**

Agent sends heartbeat every 30 seconds:

```yaml
heartbeat:
  interval: 30s
  timeout: 10s
  retry_count: 3
```

**Health Indicators:**
- Last successful heartbeat
- Command processing latency
- Error rate
- System resource usage

**Check Health:**

```bash
# Via bot
/status

# Via systemd
sudo systemctl status servereye-agent

# Via logs
sudo journalctl -u servereye-agent -n 20
```

## Logging

### Log Levels

| Level | Description | Usage |
|-------|-------------|-------|
| DEBUG | Detailed information | Development only |
| INFO | General information | Normal operations |
| WARN | Warning messages | Potential issues |
| ERROR | Error messages | Failed operations |
| FATAL | Fatal errors | System shutdown |

### Structured Logging

**Example Log Entry:**

```json
{
  "level": "info",
  "time": "2024-11-01T15:30:00Z",
  "msg": "Command executed successfully",
  "user_id": 123456789,
  "command": "/temp",
  "server_id": "srv_abc123",
  "duration": "0.234s",
  "success": true
}
```

### Log Configuration

**Bot:**

```go
logger := logrus.New()
logger.SetLevel(logrus.InfoLevel)
logger.SetFormatter(&logrus.JSONFormatter{})
logger.SetOutput(os.Stdout)
```

**Agent:**

```yaml
logging:
  level: "info"
  format: "json"
  file: "/var/log/servereye/agent.log"
  max_size: "100MB"
  max_age: "30d"
  max_backups: 5
```

### Log Rotation

```bash
# Configure logrotate
sudo cat > /etc/logrotate.d/servereye << EOF
/var/log/servereye/*.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    create 0640 servereye servereye
    postrotate
        systemctl reload servereye-agent > /dev/null 2>&1 || true
    endscript
}
EOF
```

### Sensitive Data Redaction

**Automatically Redacted:**
- Server keys (replaced with `srv_***`)
- Telegram tokens (replaced with `***`)
- Database passwords
- API keys

**Example:**

```
Before: Server key: srv_a1b2c3d4e5f6
After:  Server key: srv_***
```

## Alerting

### Built-in Alerts

**Temperature Alerts:**
```
🚨 Critical Temperature Alert!

Server: production-web-01
Zone 0: 92°C (Critical!)

Action Required:
1. Check cooling system
2. Reduce load
3. Check for dust buildup
```

**Disk Space Alerts:**
```
⚠️ Disk Space Warning

Server: production-db-01
Filesystem: /var/lib/postgresql
Used: 87% (435 GB / 500 GB)

Recommended Actions:
1. Clean old logs
2. Archive old data
3. Expand disk space
```

**Memory Alerts:**
```
⚠️ High Memory Usage

Server: production-api-01
Used: 14.2 GB / 16.0 GB (89%)

Top Processes:
1. node (8.5 GB)
2. postgres (3.2 GB)
3. redis (1.8 GB)
```

### Custom Alert Rules

**Coming Soon:**
- User-defined thresholds
- Alert schedules
- Multiple notification channels
- Alert grouping and deduplication

## Troubleshooting

### Common Issues

#### 1. High Response Time

**Symptoms:**
- Commands take > 5 seconds
- Timeout errors

**Diagnostics:**
```bash
# Check bot logs
docker logs servereye-bot | grep "timeout"

# Check Redis latency
redis-cli --latency

# Check database connections
psql -c "SELECT count(*) FROM pg_stat_activity"
```

**Solutions:**
- Increase Redis/PostgreSQL resources
- Add connection pooling
- Scale horizontally

#### 2. Agent Disconnected

**Symptoms:**
- Commands return "Agent offline"
- No heartbeats in logs

**Diagnostics:**
```bash
# Check agent status
sudo systemctl status servereye-agent

# Check agent logs
sudo journalctl -u servereye-agent -n 50

# Check network
curl https://api.servereye.dev/health
```

**Solutions:**
- Restart agent: `sudo systemctl restart servereye-agent`
- Check network connectivity
- Verify server key is registered

#### 3. High Memory Usage

**Symptoms:**
- Bot using > 500 MB RAM
- OOM killer triggered

**Diagnostics:**
```bash
# Check memory profile
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Check goroutines
curl http://localhost:6060/debug/pprof/goroutine?debug=2
```

**Solutions:**
- Fix goroutine leaks
- Add memory limits
- Implement caching strategies

### Debug Mode

**Enable Debug Logging:**

**Bot:**
```bash
docker run -e LOG_LEVEL=debug servereye-bot
```

**Agent:**
```yaml
logging:
  level: "debug"
```

**Debug Endpoints:**

```bash
# Goroutine dump
curl http://localhost:6060/debug/pprof/goroutine?debug=2

# Heap dump
curl http://localhost:6060/debug/pprof/heap > heap.prof

# CPU profile
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
```

---

**See also:**
- [Architecture](ARCHITECTURE.md)
- [Development Guide](DEVELOPMENT.md)
- [Security Considerations](SECURITY.md)
