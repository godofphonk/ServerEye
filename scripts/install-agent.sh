#!/bin/bash

# ServerEye Agent Installation Script
# This script installs and configures ServerEye agent with automatic startup

set -e

AGENT_USER="servereye"
AGENT_DIR="/opt/servereye"
CONFIG_DIR="/etc/servereye"
LOG_DIR="/var/log/servereye"
SERVICE_FILE="/etc/systemd/system/servereye-agent.service"
AGENT_URL="https://raw.githubusercontent.com/godofphonk/ServerEye/master/downloads/servereye-agent-linux"
CHECKSUM_URL="https://raw.githubusercontent.com/godofphonk/ServerEye/master/downloads/SHA256SUMS"
EXPECTED_CHECKSUM="979aafae68c93f1af80f04238b692dad7828d26d14a79de34e8b68e7c0ded651"
BOT_URL="${SERVEREYE_BOT_URL:-https://api.servereye.dev}"  #  API endpoint

echo "🚀 Installing ServerEye Agent..."

# Check dependencies
echo "🔍 Checking dependencies..."
for cmd in wget curl openssl systemctl sha256sum; do
    if ! command -v $cmd &> /dev/null; then
        echo "❌ Required command '$cmd' not found. Please install it first."
        exit 1
    fi
done

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

# Verify checksum
echo "🔐 Verifying binary integrity..."
ACTUAL_CHECKSUM=$(sha256sum "$AGENT_DIR/servereye-agent" | awk '{print $1}')
if [ "$ACTUAL_CHECKSUM" != "$EXPECTED_CHECKSUM" ]; then
    echo "❌ Checksum verification failed!"
    echo "   Expected: $EXPECTED_CHECKSUM"
    echo "   Got:      $ACTUAL_CHECKSUM"
    echo ""
    echo "⚠️  This could indicate:"
    echo "   - Binary was tampered with (MITM attack)"
    echo "   - Download was corrupted"
    echo "   - Binary version mismatch"
    echo ""
    echo "🛡️  For security, installation has been aborted."
    rm -f "$AGENT_DIR/servereye-agent"
    exit 1
fi
echo "✅ Binary integrity verified!"

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

api:
  base_url: "${API_URL:-https://api.servereye.dev}"
  timeout: "30s"

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
    echo "1. Find @ServerEyeBot in Telegram"
    echo "2. Send /start command"
    echo "3. Send: /add $SECRET_KEY"
    echo ""
    echo "🎯 Available commands after connection:"
    echo "• /temp - Get CPU temperature"
    echo "• /memory - Get memory usage"
    echo "• /disk - Get disk usage"
    echo "• /containers - List Docker containers"
    echo "• /status - Get server status"
    echo ""
    echo "📋 Service management:"
    echo "• Status: sudo systemctl status servereye-agent"
    echo "• Logs: sudo journalctl -u servereye-agent -f"
    echo "• Restart: sudo systemctl restart servereye-agent"
    echo ""
    echo "🎉 Installation complete! Your server is now monitored."
    echo ""
    echo "ℹ️  Note: If bot registration failed, you can manually add the key later."
else
    echo "❌ Service failed to start. Check logs:"
    echo "sudo journalctl -u servereye-agent -n 20"
    exit 1
fi
