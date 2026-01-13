# Quick Local Setup

Fast way to install and test ServerEye agent locally.

## One-Command Setup

```bash
# 1. Build
cd /home/gospodin/Рабочий стол/homeProjects/ServerEye
make build-agent

# 2. Install
sudo scripts/install-local.sh

# 3. Check status
sudo systemctl status servereye-agent
```

## Your Server Info

After installation, find your key:
```bash
sudo grep "secret_key:" /etc/servereye/config.yaml
```

Connect to Telegram bot @ServerEyeBot with `/add YOUR_KEY`.

## Debug

```bash
# View logs
sudo journalctl -u servereye-agent -f

# Restart service
sudo systemctl restart servereye-agent

# Run manually (debug mode)
sudo systemctl stop servereye-agent
sudo -u servereye /opt/servereye/servereye-agent -config /etc/servereye/config.yaml -log-level debug
```

## Common Commands

```bash
# Build new version
make build-agent

# Reinstall (keeps config)
sudo scripts/install-local.sh

# Stop service
sudo systemctl stop servereye-agent

# Start service
sudo systemctl start servereye-agent

# Edit config
sudo nano /etc/servereye/config.yaml

# Uninstall completely
sudo systemctl stop servereye-agent
sudo systemctl disable servereye-agent
sudo rm -rf /opt/servereye /etc/servereye /var/log/servereye
```

## Files Locations

- **Binary**: `/opt/servereye/servereye-agent`
- **Config**: `/etc/servereye/config.yaml`
- **Logs**: `/var/log/servereye/agent.log`
- **Service**: `/etc/systemd/system/servereye-agent.service`
