# Local Development Guide

This guide explains how to develop and test ServerEye agent locally without using GitHub releases.

## Prerequisites

- Go 1.21+ installed
- Docker (optional, for container metrics)
- Systemd (for service management)
- sudo access for installation

## Quick Start

### 1. Build the Agent

```bash
cd /home/gospodin/Рабочий стол/homeProjects/ServerEye
make build-agent
```

This creates the binary at `build/servereye-agent`.

### 2. Install Locally

Use the local installation script instead of the production one:

```bash
sudo scripts/install-local.sh
```

This script:
- Uses your locally built binary
- Registers with the ServerEye API
- Gets a real server key from the backend
- Creates proper systemd service
- Sets up configuration with WebSocket support

### 3. Check Status

```bash
sudo systemctl status servereye-agent
sudo journalctl -u servereye-agent -f
```

## Development Workflow

### Making Changes

1. Edit source code
2. Build locally: `make build-agent`
3. Reinstall: `sudo scripts/install-local.sh`
4. Test: Check logs and service status

### Testing Different Configurations

The local installer creates configuration at `/etc/servereye/config.yaml`. You can modify it:

```bash
sudo nano /etc/servereye/config.yaml
sudo systemctl restart servereye-agent
```

### Debug Mode

Run agent directly with debug logging:

```bash
# Stop the service first
sudo systemctl stop servereye-agent

# Run manually
sudo -u servereye /opt/servereye/servereye-agent -config /etc/servereye/config.yaml -log-level debug
```

## Configuration Options

### Enable/Disable Metrics

Edit `/etc/servereye/config.yaml`:

```yaml
metrics:
  cpu_usage: true          # CPU utilization
  memory_usage: true       # Memory usage
  disk_usage: true         # Disk usage
  cpu_temperature: true    # CPU temperature
  interval: "30s"         # Collection interval
```

### WebSocket Settings

```yaml
websocket:
  enabled: true
  url: "wss://api.servereye.dev/ws"
  reconnect_interval: "5s"
  max_reconnect_attempts: 10
  ping_interval: "30s"
```

### Docker Integration

If Docker is available, the agent will collect container metrics automatically. Make sure the `servereye` user has Docker access:

```bash
sudo usermod -aG docker servereye
```

## Troubleshooting

### Common Issues

1. **WebSocket Authentication Failed**
   - Check if server key is valid in `/etc/servereye/config.yaml`
   - Verify API endpoint is reachable

2. **Permission Denied**
   - Ensure running with sudo for installation
   - Check file permissions: `ls -la /opt/servereye/`

3. **Service Won't Start**
   - Check logs: `sudo journalctl -u servereye-agent -n 50`
   - Verify configuration: `servereye-agent -config /etc/servereye/config.yaml -log-level debug`

4. **Docker Metrics Missing**
   - Add user to docker group: `sudo usermod -aG docker servereye`
   - Restart service: `sudo systemctl restart servereye-agent`

### Log Locations

- **Systemd logs**: `sudo journalctl -u servereye-agent -f`
- **Agent logs**: `/var/log/servereye/agent.log`
- **Configuration**: `/etc/servereye/config.yaml`

### Manual Testing

Test individual components:

```bash
# Test configuration loading
sudo -u servereye /opt/servereye/servereye-agent -config /etc/servereye/config.yaml -log-level debug

# Test version
sudo -u servereye /opt/servereye/servereye-agent --version

# Test installation (dry run)
sudo scripts/install-local.sh  # Will detect existing installation
```

## API Integration

The local installer automatically:
1. Registers your server with `https://api.servereye.dev/RegisterKey`
2. Gets a unique `server_id` and `server_key`
3. Configures WebSocket connection with proper credentials

You can verify registration:
```bash
curl -X POST https://api.servereye.dev/RegisterKey \
  -H "Content-Type: application/json" \
  -H "X-API-Key: sPnMkMxyxIcjq1kJD7FOtEjUrHxvSmEU" \
  -d '{"hostname":"test","operating_system":"linux","agent_version":"1.0.0"}'
```

## Building for Production

When ready for production:

```bash
# Build optimized binaries
make release

# Copy to downloads
make downloads

# Use production installer
sudo bash scripts/install-agent.sh
```

## File Structure After Installation

```
/opt/servereye/
├── servereye-agent          # Main binary
└── servereye-agent.backup   # Backup of previous version

/etc/servereye/
├── config.yaml             # Main configuration
└── agent.env               # Environment variables

/var/log/servereye/
└── agent.log               # Agent logs

/etc/systemd/system/
└── servereye-agent.service # Systemd service file
```

## Testing WebSocket Connection

Monitor WebSocket traffic:

```bash
# Check connection status
sudo journalctl -u servereye-agent | grep -i websocket

# View authentication
sudo journalctl -u servereye-agent | grep -i "authentication"

# Monitor metrics sending
sudo journalctl -u servereye-agent | grep -i "metric"
```

## Development Tips

1. **Use Git Branches**: Create feature branches for development
2. **Test Locally**: Always test with `install-local.sh` before production
3. **Monitor Logs**: Keep an eye on systemd logs during development
4. **Backup Config**: Save working configurations before major changes
5. **Version Tracking**: Use `make version` to track build versions

## Uninstalling

Complete removal:

```bash
sudo systemctl stop servereye-agent
sudo systemctl disable servereye-agent
sudo rm -f /etc/systemd/system/servereye-agent.service
sudo systemctl daemon-reload
sudo rm -rf /opt/servereye /etc/servereye /var/log/servereye
sudo userdel servereye  # Optional
```

This removes all traces of the local installation.
