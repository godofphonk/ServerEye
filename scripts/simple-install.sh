#!/bin/bash
set -e

echo "Installing ServerEye Agent..."

# Check root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (use sudo)"
  exit 1
fi

# Install deps
apt-get update
apt-get install -y git golang-go redis-server

# Start Redis
systemctl enable redis-server
systemctl start redis-server

# Create user
useradd --system --shell /bin/false servereye || true

# Create dirs
mkdir -p /opt/servereye
mkdir -p /etc/servereye

# Build
cd /tmp
rm -rf ServerEye
git clone https://github.com/godofphonk/ServerEye.git
cd ServerEye
go build -o servereye-agent ./cmd/agent

# Install
cp servereye-agent /opt/servereye/
chmod +x /opt/servereye/servereye-agent

# Generate key
KEY="srv_$(openssl rand -hex 16)"

# Config
cat > /etc/servereye/config.yaml << 'EOF'
server:
  name: "test-server"
  secret_key: "REPLACE_KEY"
redis:
  address: "localhost:6379"
  password: ""
  db: 0
logging:
  level: "info"
  format: "json"
EOF

sed -i "s/REPLACE_KEY/$KEY/" /etc/servereye/config.yaml

# Permissions
chown -R servereye:servereye /opt/servereye
chown -R servereye:servereye /etc/servereye

# Service
cat > /etc/systemd/system/servereye-agent.service << 'EOF'
[Unit]
Description=ServerEye Agent
After=network.target

[Service]
Type=simple
User=servereye
ExecStart=/opt/servereye/servereye-agent -config /etc/servereye/config.yaml
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# Start
systemctl daemon-reload
systemctl enable servereye-agent
systemctl start servereye-agent

echo "Installation complete!"
echo "Your server key: $KEY"
echo "Send this key to your Telegram bot"
