#!/bin/bash

# ServerEye Quick Installation Script
set -e

echo "🚀 Installing ServerEye Agent..."

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    echo "❌ This script must be run as root (use sudo)"
    exit 1
fi

# Install dependencies
echo "📦 Installing dependencies..."
if command -v apt-get &> /dev/null; then
    apt-get update
    apt-get install -y git golang-go redis-server
    systemctl enable redis-server
    systemctl start redis-server
elif command -v yum &> /dev/null; then
    yum install -y git golang redis
    systemctl enable redis
    systemctl start redis
else
    echo "❌ Unsupported package manager"
    exit 1
fi

# Create directories
echo "📁 Creating directories..."
mkdir -p /opt/servereye
mkdir -p /etc/servereye

# Create user
echo "👤 Creating servereye user..."
if ! id servereye &>/dev/null; then
    useradd --system --shell /bin/false --home-dir /opt/servereye --no-create-home servereye
fi

# Clone and build
echo "🏗️ Building ServerEye..."
cd /tmp
rm -rf ServerEye
git clone https://github.com/godofphonk/ServerEye.git
cd ServerEye

export CGO_ENABLED=0
go build -o servereye-agent ./cmd/agent

# Install binary
cp servereye-agent /opt/servereye/
chmod +x /opt/servereye/servereye-agent

# Generate server key
SERVER_KEY="srv_$(openssl rand -hex 16)"

# Create config
echo "⚙️ Creating configuration..."
cat > /etc/servereye/config.yaml << EOF
server:
  name: "$(hostname)"
  secret_key: "$SERVER_KEY"

redis:
  address: "localhost:6379"
  password: ""
  db: 0

logging:
  level: "info"
  format: "json"
EOF

# Set permissions
chown -R servereye:servereye /etc/servereye
chown -R servereye:servereye /opt/servereye
chmod 600 /etc/servereye/config.yaml

# Create systemd service
echo "🔧 Creating systemd service..."
cat > /etc/systemd/system/servereye-agent.service << EOF
[Unit]
Description=ServerEye Agent
After=network.target redis.service
Wants=network.target

[Service]
Type=simple
User=servereye
Group=servereye
ExecStart=/opt/servereye/servereye-agent -config /etc/servereye/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
systemctl daemon-reload
systemctl enable servereye-agent
systemctl start servereye-agent

# Check status
sleep 2
if systemctl is-active --quiet servereye-agent; then
    echo "✅ ServerEye agent installed and started successfully!"
else
    echo "⚠️ ServerEye agent installed but may have issues starting"
    echo "Check logs with: journalctl -u servereye-agent -f"
fi

echo ""
echo "🎉 Installation completed!"
echo ""
echo "🔑 Your server key: $SERVER_KEY"
echo ""
echo "📱 Next steps:"
echo "1. Open Telegram and find your ServerEye bot"
echo "2. Send the server key: $SERVER_KEY"
echo "3. Try commands: /temp, /memory, /disk, /uptime"
echo ""
echo "🔧 Useful commands:"
echo "• Check status: systemctl status servereye-agent"
echo "• View logs: journalctl -u servereye-agent -f"
echo "• Restart: systemctl restart servereye-agent"
echo ""
