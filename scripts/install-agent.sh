#!/bin/bash

# ServerEye Agent Installation Script
# This script installs and configures ServerEye agent with automatic startup

set -e

AGENT_USER="servereye"
AGENT_DIR="/opt/servereye"
CONFIG_DIR="/etc/servereye"
LOG_DIR="/var/log/servereye"
SERVICE_FILE="/etc/systemd/system/servereye-agent.service"
AGENT_URL="https://raw.githubusercontent.com/godofphonk/ServerEye/master/downloads/servereye-linux-amd64"
BOT_URL="${SERVEREYE_BOT_URL:-http://192.168.0.105:8090}"  # Can be overridden with env var

echo "🚀 Installing ServerEye Agent..."

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "❌ This script must be run as root (use sudo)"
   exit 1
fi

# Create servereye user if doesn't exist
if ! id "$AGENT_USER" &>/dev/null; then
    echo "👤 Creating servereye user..."
    useradd -r -s /bin/false -d "$AGENT_DIR" "$AGENT_USER"
fi

# Create directories
echo "📁 Creating directories..."
mkdir -p "$AGENT_DIR" "$CONFIG_DIR" "$LOG_DIR"
chown "$AGENT_USER:$AGENT_USER" "$AGENT_DIR" "$LOG_DIR"
chmod 755 "$CONFIG_DIR"

# Download and install agent binary
echo "⬇️  Downloading ServerEye agent..."
wget -O "$AGENT_DIR/servereye-agent" "$AGENT_URL"
chmod +x "$AGENT_DIR/servereye-agent"
chown "$AGENT_USER:$AGENT_USER" "$AGENT_DIR/servereye-agent"

# Generate secret key and config
echo "🔑 Generating secret key..."
SECRET_KEY=$(openssl rand -hex 16 | sed 's/^/srv_/')
HOSTNAME=$(hostname)

# Create configuration file
echo "📝 Creating configuration..."
cat > "$CONFIG_DIR/config.yaml" << EOF
server:
  name: "$HOSTNAME"
  description: "ServerEye monitored server"
  secret_key: "$SECRET_KEY"

redis:
  address: "${REDIS_URL:-localhost:6379}"
  password: ""
  db: 0

metrics:
  cpu_temperature: true
  interval: "30s"

logging:
  level: "info"
  file: "$LOG_DIR/agent.log"
EOF

chown root:$AGENT_USER "$CONFIG_DIR/config.yaml"
chmod 640 "$CONFIG_DIR/config.yaml"

# Register key with bot
echo "🔄 Registering key with ServerEye bot..."
AGENT_VERSION="1.0.0"
OS_INFO=$(uname -s)" "$(uname -m)

JSON_PAYLOAD=$(cat << EOF
{
  "secret_key": "$SECRET_KEY",
  "agent_version": "$AGENT_VERSION",
  "os_info": "$OS_INFO",
  "hostname": "$HOSTNAME"
}
EOF
)

if curl -s -X POST "$BOT_URL/api/register-key" \
   -H "Content-Type: application/json" \
   -d "$JSON_PAYLOAD" > /dev/null; then
    echo "✅ Key registered with ServerEye bot!"
else
    echo "⚠️  Could not register key with bot (bot may be offline)"
    echo "   You can still use the key manually: $SECRET_KEY"
fi

# Install systemd service
echo "⚙️  Installing systemd service..."
cat > "$SERVICE_FILE" << 'EOF'
[Unit]
Description=ServerEye Agent - Server Monitoring Agent
After=network.target
Wants=network.target

[Service]
Type=simple
User=servereye
Group=servereye
WorkingDirectory=/opt/servereye
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
EOF

# Enable and start service
echo "🔄 Starting ServerEye agent service..."
systemctl daemon-reload
systemctl enable servereye-agent
systemctl start servereye-agent

# Wait a moment and check status
sleep 2
if systemctl is-active --quiet servereye-agent; then
    echo "✅ ServerEye Agent installed and started successfully!"
    echo ""
    echo "🔑 Your secret key: $SECRET_KEY"
    echo ""
    echo "📱 To connect to Telegram bot:"
    echo "1. Find @ServerEyeBot"
    echo "2. Send /start command"
    echo "3. Send: /add $SECRET_KEY"
    echo ""
    echo "📋 Service management:"
    echo "• Status: sudo systemctl status servereye-agent"
    echo "• Logs: sudo journalctl -u servereye-agent -f"
    echo "• Restart: sudo systemctl restart servereye-agent"
    echo ""
    echo "🎉 Installation complete! Your server is now monitored."
else
    echo "❌ Service failed to start. Check logs:"
    echo "sudo journalctl -u servereye-agent -n 20"
    exit 1
fi
