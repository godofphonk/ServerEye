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
BOT_URL="${SERVEREYE_BOT_URL:-https://api.servereye.dev}"

echo "[*] Installing ServerEye Agent..."

# Check dependencies
echo "[*] Checking dependencies..."
for cmd in wget curl openssl systemctl sha256sum; do
    if ! command -v $cmd &> /dev/null; then
        echo "[ERROR] Required command '$cmd' not found. Please install it first."
        exit 1
    fi
done

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "[ERROR] This script must be run as root (use sudo)"
   exit 1
fi

# Create servereye user if doesn't exist
if ! id "$AGENT_USER" &>/dev/null; then
    echo "[*] Creating servereye user..."
    useradd -r -s /bin/false -d "$AGENT_DIR" "$AGENT_USER"
fi

# Add servereye user to docker group (if docker exists)
if command -v docker &> /dev/null; then
    echo "[*] Adding servereye user to docker group..."
    usermod -aG docker "$AGENT_USER" 2>/dev/null || echo "[WARNING] Could not add user to docker group (docker group may not exist)"
fi

# Clean up old user service if exists (check for user who called sudo)
REAL_USER="${SUDO_USER:-$USER}"
USER_HOME=$(eval echo "~$REAL_USER")
if [ -f "$USER_HOME/.config/systemd/user/servereye-agent.service" ]; then
    echo "[*] Removing old user service for $REAL_USER..."
    su - "$REAL_USER" -c "systemctl --user stop servereye-agent 2>/dev/null || true"
    su - "$REAL_USER" -c "systemctl --user disable servereye-agent 2>/dev/null || true"
    rm -f "$USER_HOME/.config/systemd/user/servereye-agent.service"
    su - "$REAL_USER" -c "systemctl --user daemon-reload 2>/dev/null || true"
fi

# Check if this is an update
UPDATE_MODE=false
if systemctl is-active --quiet servereye-agent 2>/dev/null; then
    UPDATE_MODE=true
    echo "[*] Existing installation detected - running in UPDATE mode"
    echo "[*] Stopping agent service..."
    systemctl stop servereye-agent
    sleep 1
fi

# Create directories
echo "[*] Creating directories..."
mkdir -p "$AGENT_DIR" "$CONFIG_DIR" "$LOG_DIR"
chown "$AGENT_USER:$AGENT_USER" "$AGENT_DIR" "$LOG_DIR"
chmod 755 "$CONFIG_DIR"

# Check version if updating
if [ "$UPDATE_MODE" = true ] && [ -f "$AGENT_DIR/servereye-agent" ]; then
    echo "[*] Checking installed version..."
    INSTALLED_VERSION=$("$AGENT_DIR/servereye-agent" --version 2>/dev/null | grep -oP 'version \K[0-9.]+' || echo "unknown")
    
    # Get latest version from GitHub
    LATEST_VERSION=$(curl -sL https://api.github.com/repos/godofphonk/ServerEye/releases/latest | grep -oP '"tag_name": "\K[^"]+' | sed 's/^v//' || echo "unknown")
    
    if [ "$INSTALLED_VERSION" != "unknown" ] && [ "$LATEST_VERSION" != "unknown" ] && [ "$INSTALLED_VERSION" = "$LATEST_VERSION" ]; then
        echo "[OK] You already have the latest version ($INSTALLED_VERSION)!"
        echo ""
        
        # Show existing key
        if [ -f "$CONFIG_DIR/config.yaml" ]; then
            SECRET_KEY=$(grep 'secret_key:' "$CONFIG_DIR/config.yaml" | awk '{print $2}' | tr -d '"')
            echo "Your secret key: $SECRET_KEY"
            echo ""
            echo "To connect to Telegram bot:"
            echo "1. Find @ServerEyeBot in Telegram"
            echo "2. Send /start command"
            echo "3. Send: /add $SECRET_KEY"
            echo ""
        fi
        
        echo "Service status:"
        systemctl status servereye-agent --no-pager -l
        exit 0
    fi
    
    if [ "$INSTALLED_VERSION" != "unknown" ] && [ "$LATEST_VERSION" != "unknown" ]; then
        echo "[*] Updating from version $INSTALLED_VERSION to $LATEST_VERSION..."
    fi
    
    echo "[*] Backing up current binary..."
    cp "$AGENT_DIR/servereye-agent" "$AGENT_DIR/servereye-agent.backup"
fi

# Download and install agent binary
echo "[*] Downloading ServerEye agent..."
wget -q -O "$AGENT_DIR/servereye-agent.new" "$AGENT_URL" || {
    echo "[ERROR] Failed to download agent binary"
    exit 1
}

# Get expected SHA256 from checksums.txt
echo "[*] Verifying binary integrity..."
echo "[*] Downloading checksums..."

CHECKSUMS=$(curl -sL "$CHECKSUM_URL" 2>/dev/null || wget -qO- "$CHECKSUM_URL" 2>/dev/null)

if [ -z "$CHECKSUMS" ]; then
    echo "[ERROR] Failed to download checksums file"
    echo "   Cannot verify binary integrity without checksum"
    rm -f "$AGENT_DIR/servereye-agent.new"
    exit 1
fi

# Extract SHA256 for our binary
EXPECTED_CHECKSUM=$(echo "$CHECKSUMS" | grep "servereye-agent-linux-amd64" | awk '{print $1}')

if [ -z "$EXPECTED_CHECKSUM" ]; then
    echo "[ERROR] Could not retrieve SHA256 checksum from GitHub"
    echo "   This could indicate:"
    echo "   - Network connectivity issues"
    echo "   - GitHub API rate limit"
    echo "   - Release format changed"
    echo ""
    echo "[SECURITY] For security, installation requires checksum verification"
    rm -f "$AGENT_DIR/servereye-agent.new"
    exit 1
fi

# Calculate actual checksum
ACTUAL_CHECKSUM=$(sha256sum "$AGENT_DIR/servereye-agent.new" | awk '{print $1}')

if [ "$ACTUAL_CHECKSUM" != "$EXPECTED_CHECKSUM" ]; then
    echo "[ERROR] SHA256 checksum verification FAILED!"
    echo ""
    echo "   Expected: $EXPECTED_CHECKSUM"
    echo "   Got:      $ACTUAL_CHECKSUM"
    echo ""
    echo "[WARNING] This could indicate:"
    echo "   - Binary was tampered with (MITM attack)"
    echo "   - Download was corrupted"
    echo "   - Network issues during download"
    echo ""
    echo "[SECURITY] For security, installation has been aborted."
    echo "   Please try again or contact support."
    rm -f "$AGENT_DIR/servereye-agent.new"
    exit 1
fi

echo "[OK] SHA256 checksum verified successfully!"
echo "   Checksum: ${ACTUAL_CHECKSUM:0:16}..."

# Move new binary to final location
mv "$AGENT_DIR/servereye-agent.new" "$AGENT_DIR/servereye-agent"
chmod +x "$AGENT_DIR/servereye-agent"
chown "$AGENT_USER:$AGENT_USER" "$AGENT_DIR/servereye-agent"

# Configuration handling
if [ "$UPDATE_MODE" = true ] && [ -f "$CONFIG_DIR/config.yaml" ]; then
    echo "[*] Keeping existing configuration..."
    SECRET_KEY=$(grep 'secret_key:' "$CONFIG_DIR/config.yaml" | awk '{print $2}' | tr -d '"')
else
    # Generate secret key and config for new installation
    echo "[*] Generating secret key..."
    SECRET_KEY=$(openssl rand -hex 16 | sed 's/^/srv_/')
    HOSTNAME=$(hostname)

    # Create configuration file
    echo "[*] Creating configuration..."
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
    echo "[*] Registering key with ServerEye bot..."
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
        echo "[OK] Key registered with ServerEye bot!"
    else
        echo "[WARNING] Could not register key with bot (bot may be offline)"
        echo "   You can still use the key manually: $SECRET_KEY"
    fi
fi

# Install systemd service
echo "[*] Installing systemd service..."
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
    echo "[*] Restarting ServerEye agent service..."
    systemctl start servereye-agent
else
    echo "[*] Starting ServerEye agent service..."
    systemctl start servereye-agent
fi

# Wait and check status
sleep 2
if systemctl is-active --quiet servereye-agent; then
    if [ "$UPDATE_MODE" = true ]; then
        echo "[OK] ServerEye Agent updated successfully!"
        echo ""
        echo "Your secret key: $SECRET_KEY"
        echo ""
        echo "What was updated:"
        echo "  - Agent binary updated to latest version"
        echo "  - Configuration preserved"
        echo "  - Service restarted"
        echo "  - Previous version backed up"
        echo ""
        echo "Service management:"
        echo "  - Status: sudo systemctl status servereye-agent"
        echo "  - Logs: sudo journalctl -u servereye-agent -f"
        echo ""
        echo "Update complete!"
    else
        echo "[OK] ServerEye Agent installed and started successfully!"
        echo ""
        echo "Your secret key: $SECRET_KEY"
        echo ""
        echo "To connect to Telegram bot:"
        echo "1. Find @ServerEyeBot in Telegram"
        echo "2. Send /start command"
        echo "3. Send: /add $SECRET_KEY"
        echo ""
        echo "Available commands after connection:"
        echo "  - /temp - Get CPU temperature"
        echo "  - /memory - Get memory usage"
        echo "  - /disk - Get disk usage"
        echo "  - /containers - List Docker containers"
        echo "  - /status - Get server status"
        echo ""
        echo "Service management:"
        echo "  - Status: sudo systemctl status servereye-agent"
        echo "  - Logs: sudo journalctl -u servereye-agent -f"
        echo ""
        echo "Installation complete!"
    fi
else
    echo "[ERROR] Service failed to start. Check logs:"
    echo "sudo journalctl -u servereye-agent -n 20"
    if [ "$UPDATE_MODE" = true ]; then
        echo ""
        echo "To rollback:"
        echo "sudo systemctl stop servereye-agent"
        echo "sudo cp $AGENT_DIR/servereye-agent.backup $AGENT_DIR/servereye-agent"
        echo "sudo systemctl start servereye-agent"
    fi
    exit 1
fi
