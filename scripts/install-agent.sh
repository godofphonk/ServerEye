#!/bin/bash

# ServerEye Agent Installation Script
# This script installs and configures ServerEye agent with automatic startup

set -e

AGENT_USER="servereye"
AGENT_DIR="/opt/servereye"
CONFIG_DIR="/etc/servereye"
LOG_DIR="/var/log/servereye"
SERVICE_FILE="/etc/systemd/system/servereye-agent.service"
AGENT_URL="https://github.com/godofphonk/ServerEye/releases/latest/download/servereye-agent-linux-amd64"
CHECKSUM_URL="https://github.com/godofphonk/ServerEye/releases/latest/download/checksums.txt"
BOT_URL="${SERVEREYE_BOT_URL:-https://api.servereye.dev}"  #  API endpoint

echo "ðŸš€ Installing ServerEye Agent..."

# Check dependencies
echo "ðŸ” Checking dependencies..."
for cmd in wget curl openssl systemctl sha256sum; do
    if ! command -v $cmd &> /dev/null; then
        echo "âŒ Required command '$cmd' not found. Please install it first."
        exit 1
    fi
done

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "âŒ This script must be run as root (use sudo)"
   exit 1
fi

# Create servereye user if doesn't exist
if ! id "$AGENT_USER" &>/dev/null; then
    echo "ðŸ‘¤ Creating servereye user..."
    useradd -r -s /bin/false -d "$AGENT_DIR" "$AGENT_USER"
fi

# Check if this is an update
UPDATE_MODE=false
if systemctl is-active --quiet servereye-agent 2>/dev/null; then
    UPDATE_MODE=true
    echo "ðŸ”„ Existing installation detected - running in UPDATE mode"
    echo "â¸ï¸  Stopping agent service..."
    systemctl stop servereye-agent
    sleep 1
fi

# Create directories
echo "ðŸ“ Creating directories..."
mkdir -p "$AGENT_DIR" "$CONFIG_DIR" "$LOG_DIR"
chown "$AGENT_USER:$AGENT_USER" "$AGENT_DIR" "$LOG_DIR"
chmod 755 "$CONFIG_DIR"

# Backup existing binary if updating
if [ "$UPDATE_MODE" = true ] && [ -f "$AGENT_DIR/servereye-agent" ]; then
    echo "ðŸ’¾ Backing up current binary..."
    cp "$AGENT_DIR/servereye-agent" "$AGENT_DIR/servereye-agent.backup"
fi

# Download and install agent binary
echo "â¬‡ï¸  Downloading ServerEye agent..."
wget -q -O "$AGENT_DIR/servereye-agent.new" "$AGENT_URL" || {
    echo "âŒ Failed to download agent binary"
    exit 1
}

# Get expected SHA256 from GitHub API
echo "ðŸ” Verifying binary integrity..."
echo "ðŸ“¡ Getting SHA256 from GitHub..."

GITHUB_API="https://api.github.com/repos/godofphonk/ServerEye/releases/latest"
RELEASE_DATA=$(curl -s "$GITHUB_API" || wget -qO- "$GITHUB_API")

if [ -z "$RELEASE_DATA" ]; then
    echo "âŒ Failed to fetch release information from GitHub"
    echo "   Cannot verify binary integrity without checksum"
    rm -f "$AGENT_DIR/servereye-agent.new"
    exit 1
fi

# Extract SHA256 from GitHub API response
# GitHub provides SHA256 in the browser_download_url response
EXPECTED_CHECKSUM=$(echo "$RELEASE_DATA" | grep -A 5 '"name": "servereye-agent-linux-amd64"' | grep '"label":' | sed 's/.*sha256:\([a-f0-9]*\).*/\1/' | head -n1)

# If not found in label, try to download from checksums.txt asset
if [ -z "$EXPECTED_CHECKSUM" ]; then
    echo "ðŸ“„ Fetching checksums file..."
    CHECKSUMS=$(curl -sL "$CHECKSUM_URL" || wget -qO- "$CHECKSUM_URL")
    if [ -n "$CHECKSUMS" ]; then
        EXPECTED_CHECKSUM=$(echo "$CHECKSUMS" | grep "servereye-agent-linux-amd64" | awk '{print $1}')
    fi
fi

if [ -z "$EXPECTED_CHECKSUM" ]; then
    echo "âŒ Could not retrieve SHA256 checksum from GitHub"
    echo "   This could indicate:"
    echo "   - Network connectivity issues"
    echo "   - GitHub API rate limit"
    echo "   - Release format changed"
    echo ""
    echo "ðŸ›¡ï¸  For security, installation requires checksum verification"
    rm -f "$AGENT_DIR/servereye-agent.new"
    exit 1
fi

# Calculate actual checksum
ACTUAL_CHECKSUM=$(sha256sum "$AGENT_DIR/servereye-agent.new" | awk '{print $1}')

if [ "$ACTUAL_CHECKSUM" != "$EXPECTED_CHECKSUM" ]; then
    echo "âŒ SHA256 checksum verification FAILED!"
    echo ""
    echo "   Expected: $EXPECTED_CHECKSUM"
    echo "   Got:      $ACTUAL_CHECKSUM"
    echo ""
    echo "âš ï¸  This could indicate:"
    echo "   - Binary was tampered with (MITM attack)"
    echo "   - Download was corrupted"
    echo "   - Network issues during download"
    echo ""
    echo "ðŸ›¡ï¸  For security, installation has been aborted."
    echo "   Please try again or contact support."
    rm -f "$AGENT_DIR/servereye-agent.new"
    exit 1
fi

echo "âœ… SHA256 checksum verified successfully!"
echo "   Checksum: ${ACTUAL_CHECKSUM:0:16}..."

# Move new binary to final location
mv "$AGENT_DIR/servereye-agent.new" "$AGENT_DIR/servereye-agent"
chmod +x "$AGENT_DIR/servereye-agent"
chown "$AGENT_USER:$AGENT_USER" "$AGENT_DIR/servereye-agent"

# Configuration handling
if [ "$UPDATE_MODE" = true ] && [ -f "$CONFIG_DIR/config.yaml" ]; then
    echo "ðŸ“ Keeping existing configuration..."
    # Extract existing secret key for display
    SECRET_KEY=$(grep 'secret_key:' "$CONFIG_DIR/config.yaml" | awk '{print $2}' | tr -d '"')
else
    # Generate secret key and config for new installation
    echo "ðŸ”‘ Generating secret key..."
    SECRET_KEY=$(openssl rand -hex 16 | sed 's/^/srv_/')
    HOSTNAME=$(hostname)

    # Create configuration file
    echo "ðŸ“ Creating configuration..."
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

    # Register key with bot (only for new installations)
    echo "ðŸ”„ Registering key with ServerEye bot..."
    AGENT_VERSION="1.0.0"
    OS_INFO=$(uname -s)" "$(uname -m)
    HOSTNAME=$(hostname)

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
        echo "âœ… Key registered with ServerEye bot!"
    else
        echo "âš ï¸  Could not register key with bot (bot may be offline)"
        echo "   You can still use the key manually: $SECRET_KEY"
    fi
fi

# Install systemd service
echo "âš™ï¸  Installing systemd service..."
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
systemctl daemon-reload
systemctl enable servereye-agent

if [ "$UPDATE_MODE" = true ]; then
    echo "ðŸ”„ Restarting ServerEye agent service..."
    systemctl start servereye-agent
else
    echo "ðŸ”„ Starting ServerEye agent service..."
    systemctl start servereye-agent
fi

# Wait a moment and check status
sleep 2
if systemctl is-active --quiet servereye-agent; then
    if [ "$UPDATE_MODE" = true ]; then
        echo "âœ… ServerEye Agent updated successfully!"
        echo ""
        echo "ðŸ”‘ Your secret key: $SECRET_KEY"
        echo ""
        echo "ðŸ“‹ What was updated:"
        echo "â€¢ Agent binary updated to latest version"
        echo "â€¢ Configuration preserved"
        echo "â€¢ Service restarted"
        echo "â€¢ Previous version backed up to: $AGENT_DIR/servereye-agent.backup"
        echo ""
        echo "ðŸ“‹ Service management:"
        echo "â€¢ Status: sudo systemctl status servereye-agent"
        echo "â€¢ Logs: sudo journalctl -u servereye-agent -f"
        echo "â€¢ Rollback: sudo cp $AGENT_DIR/servereye-agent.backup $AGENT_DIR/servereye-agent && sudo systemctl restart servereye-agent"
        echo ""
        echo "ðŸŽ‰ Update complete! Your server is now running the latest version."
    else
        echo "âœ… ServerEye Agent installed and started successfully!"
        echo ""
        echo "ðŸ”‘ Your secret key: $SECRET_KEY"
        echo ""
        echo "ðŸ“± To connect to Telegram bot:"
        echo "1. Find @ServerEyeBot in Telegram"
        echo "2. Send /start command"
        echo "3. Send: /add $SECRET_KEY"
        echo ""
        echo "ðŸŽ¯ Available commands after connection:"
        echo "â€¢ /temp - Get CPU temperature"
        echo "â€¢ /memory - Get memory usage"
        echo "â€¢ /disk - Get disk usage"
        echo "â€¢ /containers - List Docker containers"
        echo "â€¢ /status - Get server status"
        echo ""
        echo "ðŸ“‹ Service management:"
        echo "â€¢ Status: sudo systemctl status servereye-agent"
        echo "â€¢ Logs: sudo journalctl -u servereye-agent -f"
        echo "â€¢ Restart: sudo systemctl restart servereye-agent"
        echo ""
        echo "ðŸŽ‰ Installation complete! Your server is now monitored."
        echo ""
        echo "â„¹ï¸  Note: If bot registration failed, you can manually add the key later."
    fi
else
    echo "âŒ Service failed to start. Check logs:"
    echo "sudo journalctl -u servereye-agent -n 20"
    if [ "$UPDATE_MODE" = true ]; then
        echo ""
        echo "ðŸ”„ To rollback to previous version:"
        echo "sudo systemctl stop servereye-agent"
        echo "sudo cp $AGENT_DIR/servereye-agent.backup $AGENT_DIR/servereye-agent"
        echo "sudo systemctl start servereye-agent"
    fi
    exit 1
fi
