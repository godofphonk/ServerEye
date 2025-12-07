#!/bin/bash
# Cloudflare Tunnel Setup for ServerEye Key Database
# Run this on your server after deploying the database

set -e

echo "🌐 Setting up Cloudflare Tunnel for PostgreSQL..."

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check if cloudflared is installed
if ! command -v cloudflared &> /dev/null; then
    echo -e "${YELLOW}Installing cloudflared...${NC}"
    wget -q https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
    sudo dpkg -i cloudflared-linux-amd64.deb
    rm cloudflared-linux-amd64.deb
fi

# Get database password from .env
cd /opt/servereye
if [ ! -f .env ]; then
    echo -e "${RED}.env not found in /opt/servereye${NC}"
    exit 1
fi

source .env
DB_PASSWORD="${KEYS_DB_PASSWORD}"

# Create tunnel
echo -e "${YELLOW}Creating Cloudflare tunnel...${NC}"
cloudflared tunnel login

# Generate tunnel name and create
TUNNEL_NAME="servereye-db-$(date +%s)"
cloudflared tunnel create "$TUNNEL_NAME"

# Get tunnel credentials file path
TUNNEL_UUID=$(cloudflared tunnel list | grep "$TUNNEL_NAME" | awk '{print $1}')
CONFIG_PATH="/root/.cloudflared/${TUNNEL_UUID}.json"

# Create config directory and config file
mkdir -p /root/.cloudflared
cat > /root/.cloudflared/config.yml << EOF
tunnel: $TUNNEL_NAME
credentials-file: $CONFIG_PATH

ingress:
  - hostname: db.yourdomain.com
    service: tcp://localhost:5433
  - service: http_status:404
EOF

# Route DNS (replace with your domain)
echo -e "${YELLOW}Please replace 'yourdomain.com' with your actual domain${NC}"
echo "Routing DNS for tunnel..."
cloudflared tunnel route dns "$TUNNEL_NAME" db.yourdomain.com

# Create systemd service
cat > /etc/systemd/system/cloudflared.service << EOF
[Unit]
Description=cloudflared
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/cloudflared tunnel run $TUNNEL_NAME
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
systemctl enable cloudflared
systemctl start cloudflared

echo ""
echo -e "${GREEN}=== Cloudflare Tunnel Setup Complete ===${NC}"
echo "Tunnel name: $TUNNEL_NAME"
echo "Public URL: https://db.yourdomain.com"
echo ""
echo "Update your agent installation with:"
echo "export SERVEREYE_DB_URL=\"postgres://servereye_keys:$DB_PASSWORD@db.yourdomain.com:5432/PgRegisteredKeys?sslmode=require\""
echo ""
echo -e "${GREEN}Tunnel is running as a service!${NC}"
