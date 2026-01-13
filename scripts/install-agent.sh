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
AGENT_ENV_FILE="$CONFIG_DIR/agent.env"
# Backend API configuration
DEFAULT_BACKEND_URL="https://api.servereye.dev"
BACKEND_URL="${SERVEREYE_BACKEND_URL:-$DEFAULT_BACKEND_URL}"
API_KEY="${SERVEREYE_API_KEY:-sPnMkMxyxIcjq1kJD7FOtEjUrHxvSmEU}"

ensure_api_env() {
    # Load existing env file if present
    if [ -f "$AGENT_ENV_FILE" ]; then
        # shellcheck disable=SC1090
        source "$AGENT_ENV_FILE"
    fi

    if [ -z "$BACKEND_URL" ]; then
        if [ -t 0 ]; then
            echo "[*] BACKEND_URL not set."
            read -r -p "Enter BACKEND_URL (ServerEye Backend API endpoint): " user_backend_url
            BACKEND_URL="$user_backend_url"
        fi
    fi

    if [ -z "$BACKEND_URL" ]; then
        cat <<EOF
[ERROR] BACKEND_URL is required for key registration.
  - Option 1: export it before running the installer
      BACKEND_URL=https://api.servereye.dev bash install-agent.sh
  - Option 2: rerun this installer from an interactive shell and enter the value when prompted
EOF
        exit 1
    fi

    mkdir -p "$CONFIG_DIR"
    cat > "$AGENT_ENV_FILE" <<EOF
BACKEND_URL="$BACKEND_URL"
API_KEY="$API_KEY"
DATABASE_URL="$DATABASE_URL"
EOF
    chmod 600 "$AGENT_ENV_FILE"
    echo "[*] Stored configuration in $AGENT_ENV_FILE"
}

register_server_with_api() {
    local agent_version="$1"
    local operating_system="$2"
    local hostname="$3"

    if [ -z "$BACKEND_URL" ]; then
        echo "[WARNING] BACKEND_URL not set - skipping server registration"
        return 1
    fi

    local payload
    payload=$(cat << EOF
{
  "hostname": "$hostname",
  "operating_system": "$operating_system",
  "agent_version": "$agent_version"
}
EOF
)

    local response_file
    response_file=$(mktemp)
    local http_code
    
    echo "[*] Registering server with API at $BACKEND_URL" >&2
    
    # Use new API endpoint
    local endpoint="$BACKEND_URL/RegisterKey"
    
    # Build curl command
    local curl_cmd=(curl -s -o "$response_file" -w "%{http_code}" -X POST "$endpoint" -H "Content-Type: application/json" -H "X-API-Key: $API_KEY")
    
    # Add payload to curl command
    curl_cmd+=(-d "$payload")
    
    # Execute curl command
    http_code=$("${curl_cmd[@]}")
    
    # Check HTTP response
    if [ "$http_code" = "200" ] || [ "$http_code" = "201" ]; then
        # Parse response to get server_key
        if command -v jq >/dev/null 2>&1; then
            # Use jq if available
            server_key=$(jq -r '.server_key' "$response_file" 2>/dev/null)
            server_id=$(jq -r '.server_id' "$response_file" 2>/dev/null)
            status=$(jq -r '.status' "$response_file" 2>/dev/null)
        else
            # Fallback to grep/sed
            server_key=$(grep -o '"server_key":"[^"]*"' "$response_file" | sed 's/"server_key":"\([^"]*\)"/\1/' 2>/dev/null)
            server_id=$(grep -o '"server_id":"[^"]*"' "$response_file" | sed 's/"server_id":"\([^"]*\)"/\1/' 2>/dev/null)
            status=$(grep -o '"status":"[^"]*"' "$response_file" | sed 's/"status":"\([^"]*\)"/\1/' 2>/dev/null)
        fi
        
        # Clean up response file
        rm -f "$response_file"
        
        # Validate response
        if [ "$status" = "registered" ] && [ -n "$server_key" ]; then
            echo "[OK] Server registered successfully" >&2
            echo "[INFO] Server ID: $server_id" >&2
            echo "[INFO] Server Key: ${server_key:0:20}..." >&2
            
            # Return both server_id and server_key in format: "server_id|server_key"
            echo "${server_id}|${server_key}"
            return 0
        else
            echo "[ERROR] Invalid response from API" >&2
            if [ -f "$response_file" ]; then
                echo "[DEBUG] Response: $(cat "$response_file")" >&2
                rm -f "$response_file"
            fi
            return 1
        fi
    else
        echo "[ERROR] API request failed with HTTP code: $http_code" >&2
        if [ -f "$response_file" ]; then
            echo "[DEBUG] Response: $(cat "$response_file")" >&2
            rm -f "$response_file"
        fi
        return 1
    fi
}

register_key_with_api() {
    local secret_key="$1"
    local agent_version="$2"
    local os_info="$3"
    local hostname="$4"

    if [ -z "$BACKEND_URL" ]; then
        echo "[WARNING] BACKEND_URL not set - skipping backend API registration"
        return 1
    fi

    local payload
    payload=$(cat << EOF
{
  "secret_key": "$secret_key",
  "agent_version": "$agent_version",
  "os_info": "$os_info",
  "hostname": "$hostname"
}
EOF
)

    local response_file
    response_file=$(mktemp)
    local http_code
    
    # Prepare curl headers
    local curl_headers="-H 'Content-Type: application/json' -H 'X-API-Key: $API_KEY'"
    
    echo "[*] Registering key with backend API at $BACKEND_URL"
    
    # Use backend API endpoint
    local endpoint="$BACKEND_URL/api/v1/register-key"
    
    # Build curl command without eval
    local curl_cmd=(curl -s -o "$response_file" -w "%{http_code}" -X POST "$endpoint" -H "Content-Type: application/json" -H "X-API-Key: $API_KEY")
    
    # Add payload
    curl_cmd+=(-d "$payload")
    
    # Execute curl
    http_code=$("${curl_cmd[@]}")

    if [ "$http_code" = "200" ]; then
        echo "[OK] Key registered with ServerEye backend API!"
        rm -f "$response_file"
        return 0
    fi

    echo "[WARNING] Backend API registration failed (status $http_code)"
    if [ -s "$response_file" ]; then
        echo "          Response: $(cat "$response_file")"
    fi
    rm -f "$response_file"
    return 1
}

echo "[*] Installing ServerEye Agent..."

# Check dependencies
echo "[*] Checking dependencies..."
for cmd in wget curl openssl systemctl sha256sum; do
    if ! command -v $cmd &> /dev/null; then
        echo "[ERROR] Required command '$cmd' not found. Please install it first."
        exit 1
    fi
done

# Check for netcat (optional, for Kafka auto-detection)
if ! command -v nc &> /dev/null; then
    echo "[INFO] netcat not found - Kafka auto-detection will be limited"
    NETCAT_AVAILABLE=false
else
    NETCAT_AVAILABLE=true
fi

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

# Create proper home directory for servereye user to fix systemd namespace issues
if [ ! -d "/home/$AGENT_USER" ]; then
    echo "[*] Creating home directory for $AGENT_USER..."
    mkdir -p "/home/$AGENT_USER"
    chown "$AGENT_USER:$AGENT_USER" "/home/$AGENT_USER"
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

# Ensure API endpoint is available
ensure_api_env
export BACKEND_URL API_KEY DATABASE_URL

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

# Verify checksum
if [ "$ACTUAL_CHECKSUM" != "$EXPECTED_CHECKSUM" ]; then
    echo "[ERROR] Binary integrity check failed!"
    echo "Expected: $EXPECTED_CHECKSUM"
    echo "Actual:   $ACTUAL_CHECKSUM"
    echo ""
    echo "[SECURITY] Installation aborted for security reasons"
    rm -f "$AGENT_DIR/servereye-agent.new"
    exit 1
fi

echo "[OK] Binary integrity verified"
echo "   Checksum: ${ACTUAL_CHECKSUM:0:16}..."

# Move new binary to final location
mv "$AGENT_DIR/servereye-agent.new" "$AGENT_DIR/servereye-agent"
chmod +x "$AGENT_DIR/servereye-agent"
chown "$AGENT_USER:$AGENT_USER" "$AGENT_DIR/servereye-agent"

# Configuration handling
if [ "$UPDATE_MODE" = true ] && [ -f "$CONFIG_DIR/config.yaml" ]; then
    echo "[*] Keeping existing configuration..."
    SECRET_KEY=$(grep 'secret_key:' "$CONFIG_DIR/config.yaml" | awk '{print $2}' | tr -d '"')
    
    echo "[OK] Configuration preserved for update"
    echo "[INFO] Existing server_key will be used"
else
    # New installation - register server with API and get server_key
    echo "[*] Registering server with ServerEye API..."
    
    # Get system information
    AGENT_VERSION=$("$AGENT_DIR/servereye-agent" --version 2>/dev/null | awk '{print $3}' | sed 's/^v//' | cut -d'-' -f1 || echo "unknown")
    HOSTNAME=$(hostname)
    
    # Get operating system information
    if [ -f /etc/os-release ]; then
        # For Linux systems with /etc/os-release
        . /etc/os-release
        OPERATING_SYSTEM="$PRETTY_NAME"
    elif command -v uname >/dev/null 2>&1; then
        # Fallback to uname
        OPERATING_SYSTEM="$(uname -s) $(uname -r)"
    else
        OPERATING_SYSTEM="Unknown"
    fi
    
    echo "[INFO] Agent Version: $AGENT_VERSION"
    echo "[INFO] Hostname: $HOSTNAME"
    echo "[INFO] Operating System: $OPERATING_SYSTEM"
    
    # Register server with API and get server_id and server_key
    REGISTRATION_RESULT=$(register_server_with_api "$AGENT_VERSION" "$OPERATING_SYSTEM" "$HOSTNAME")
    
    if [ $? -eq 0 ] && [ -n "$REGISTRATION_RESULT" ]; then
        # Parse registration result: "server_id|server_key"
        SERVER_ID=$(echo "$REGISTRATION_RESULT" | cut -d'|' -f1)
        SERVER_KEY=$(echo "$REGISTRATION_RESULT" | cut -d'|' -f2)
        
        echo "[OK] Server registered successfully!"
        echo "[INFO] Server ID: $SERVER_ID"
        echo "[INFO] Server Key: ${SERVER_KEY:0:20}..."
        SECRET_KEY="$SERVER_KEY"
    else
        echo "[ERROR] Failed to register server with API"
        echo "[INFO] Please check your network connection and API credentials"
        echo "[INFO] BACKEND_URL: $BACKEND_URL"
        exit 1
    fi

    # Create configuration file
    echo "[*] Creating configuration..."
    
    # Create configuration with WebSocket support
    cat > "$CONFIG_DIR/config.yaml" << EOF
server:
  name: "$HOSTNAME"
  description: "ServerEye monitored server"
  secret_key: "$SECRET_KEY"
  server_id: "$SERVER_ID"

api:
  base_url: "$BACKEND_URL"
  api_key: "$API_KEY"
  timeout: "30s"

websocket:
  enabled: true
  url: "wss://api.servereye.dev/ws"
  reconnect_interval: "5s"
  max_reconnect_attempts: 10
  ping_interval: "30s"
  write_timeout: "10s"
  read_timeout: "10s"
  handshake_timeout: "10s"
  buffer_size: 1000
  enable_compression: true
  metric_buffer_size: 100
  metric_buffer_flush: "30s"
  command_queue_size: 100
  command_timeout: "30s"

metrics:
  cpu_usage: false
  memory_usage: false
  disk_usage: false
  cpu_temperature: true
  interval: "30s"

logging:
  level: "info"
  file: "/var/log/servereye/agent.log"
EOF
    echo "[OK] Configuration created with server-provided key"

    chown root:$AGENT_USER "$CONFIG_DIR/config.yaml"
    chmod 640 "$CONFIG_DIR/config.yaml"

    echo "[OK] ServerEye agent installation completed successfully!"
    echo "[INFO] Configuration file: $CONFIG_DIR/config.yaml"
    echo "[INFO] Log directory: $LOG_DIR"
    echo "[INFO] Agent binary: $AGENT_DIR/servereye-agent"
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
EnvironmentFile=/etc/servereye/agent.env
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
        echo "  - Restart: sudo systemctl restart servereye-agent"
        echo "  - Stop: sudo systemctl stop servereye-agent"
        echo "  - Start: sudo systemctl start servereye-agent"
        echo "  - Logs: sudo journalctl -u servereye-agent -f"
        echo "  - Enable: sudo systemctl enable servereye-agent"
        echo "  - Disable: sudo systemctl disable servereye-agent"
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
        echo "  - Restart: sudo systemctl restart servereye-agent"
        echo "  - Stop: sudo systemctl stop servereye-agent"
        echo "  - Start: sudo systemctl start servereye-agent"
        echo "  - Logs: sudo journalctl -u servereye-agent -f"
        echo "  - Enable: sudo systemctl enable servereye-agent"
        echo "  - Disable: sudo systemctl disable servereye-agent"
        echo ""
        echo "Complete uninstallation:"
        echo "  sudo systemctl stop servereye-agent"
        echo "  sudo systemctl disable servereye-agent"
        echo "  sudo rm -f /etc/systemd/system/servereye-agent.service"
        echo "  sudo systemctl daemon-reload"
        echo "  sudo rm -rf /opt/servereye /etc/servereye /var/log/servereye"
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
