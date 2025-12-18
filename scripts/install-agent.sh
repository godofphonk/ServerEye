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
DEFAULT_SECRET_ENDPOINT="https://servereye-secret-endpoint.servereye.workers.dev/agent"
AGENT_SECRET_ENDPOINT="${SERVEREYE_SECRET_ENDPOINT:-$DEFAULT_SECRET_ENDPOINT}"
AGENT_INSTALLER_KEY="${SERVEREYE_INSTALLER_KEY:-}"
AGENT_SECRET_TOKEN="${SERVEREYE_SECRET_TOKEN:-}"
DEFAULT_API_URL="https://servereye-registration-worker.servereye.workers.dev"
SERVEREYE_API_URL="${SERVEREYE_API_URL:-$DEFAULT_API_URL}"

# Worker API configuration
USE_WORKER_API="${USE_WORKER_API:-true}"  # Use Cloudflare Worker by default
AGENT_INSTALLER_KEY="${AGENT_INSTALLER_KEY:-301210}"

ensure_api_env() {
    # Load existing env file if present
    if [ -f "$AGENT_ENV_FILE" ]; then
        # shellcheck disable=SC1090
        source "$AGENT_ENV_FILE"
    fi

    if [ -z "$SERVEREYE_API_URL" ]; then
        fetch_env_from_secret_endpoint || true
    fi

    if [ -z "$SERVEREYE_API_URL" ]; then
        if [ -t 0 ]; then
            echo "[*] SERVEREYE_API_URL not set."
            read -r -p "Enter SERVEREYE_API_URL (ServerEye API endpoint): " user_api_url
            SERVEREYE_API_URL="$user_api_url"
        fi
    fi

    if [ -z "$SERVEREYE_API_URL" ]; then
        cat <<EOF
[ERROR] SERVEREYE_API_URL is required for key registration.
  - Option 1: export it before running the installer
      SERVEREYE_API_URL=https://api.servereye.dev bash install-agent.sh
  - Option 2: rerun this installer from an interactive shell and enter the value when prompted
  - Option 3: Set SERVEREYE_SECRET_ENDPOINT and SERVEREYE_INSTALLER_KEY to fetch from a secure endpoint
EOF
        exit 1
    fi

    mkdir -p "$CONFIG_DIR"
    cat > "$AGENT_ENV_FILE" <<EOF
SERVEREYE_API_URL="$SERVEREYE_API_URL"
USE_WORKER_API="$USE_WORKER_API"
AGENT_INSTALLER_KEY="$AGENT_INSTALLER_KEY"
EOF
    chmod 600 "$AGENT_ENV_FILE"
    echo "[*] Stored SERVEREYE_API_URL in $AGENT_ENV_FILE"
}

register_key_with_api() {
    local secret_key="$1"
    local agent_version="$2"
    local os_info="$3"
    local hostname="$4"

    if [ -z "$SERVEREYE_API_URL" ]; then
        echo "[WARNING] SERVEREYE_API_URL not set - skipping backend API registration"
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
    local curl_headers="-H 'Content-Type: application/json'"
    
    # Add installer key if available
    if [ -n "$AGENT_INSTALLER_KEY" ]; then
        curl_headers="$curl_headers -H 'x-installer-key: $AGENT_INSTALLER_KEY'"
    fi
    
    # Use /api/register-key for Cloudflare Worker (fallback to /api/v1/register-key for backend)
    local endpoint="$SERVEREYE_API_URL/api/register-key"
    if [ "$USE_WORKER_API" != "true" ]; then
        endpoint="$SERVEREYE_API_URL/api/v1/register-key"
    fi
    
    # Build curl command without eval
    local curl_cmd=(curl -s -o "$response_file" -w "%{http_code}" -X POST "$endpoint" -H "Content-Type: application/json")
    
    # Add installer key header if available
    if [ -n "$AGENT_INSTALLER_KEY" ]; then
        curl_cmd+=(-H "x-installer-key: $AGENT_INSTALLER_KEY")
    fi
    
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

fetch_env_from_secret_endpoint() {
    if [ -z "$AGENT_SECRET_ENDPOINT" ]; then
        return 1
    fi

    if ! command -v curl >/dev/null 2>&1; then
        echo "[WARNING] curl is required to fetch secrets automatically"
        return 1
    fi

    echo "[*] Fetching secrets from secure endpoint..."
    local curl_cmd=(curl -fsSL "$AGENT_SECRET_ENDPOINT")

    if [ -n "$AGENT_SECRET_TOKEN" ]; then
        curl_cmd+=(-H "X-Secret-Token: $AGENT_SECRET_TOKEN")
    elif [ -n "$AGENT_INSTALLER_KEY" ]; then
        curl_cmd+=(-H "X-Installer-Key: $AGENT_INSTALLER_KEY")
    fi

    local response
    if ! response=$("${curl_cmd[@]}"); then
        echo "[WARNING] Could not fetch secrets from endpoint"
        return 1
    fi

    mkdir -p "$CONFIG_DIR"
    echo "$response" > "$AGENT_ENV_FILE"
    chmod 600 "$AGENT_ENV_FILE"
    # shellcheck disable=SC1090
    source "$AGENT_ENV_FILE"
    echo "[OK] Secrets pulled successfully"
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
export SERVEREYE_API_URL

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
    
    # Update agent version in bot database
    echo "[*] Updating agent version in bot database..."
    AGENT_VERSION=$("$AGENT_DIR/servereye-agent" --version 2>/dev/null | grep -oP 'ServerEye Agent v\K[0-9.]+' || echo "unknown")
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
        echo "[OK] Agent version updated in bot database!"
    else
        echo "[WARNING] Could not update version in bot database"
    fi

    echo "[*] Ensuring key is registered in backend database..."
    if register_key_with_api "$SECRET_KEY" "$AGENT_VERSION" "$OS_INFO" "$HOSTNAME"; then
        echo "[OK] Backend registration ensured"
    else
        echo "[WARNING] Could not register key in backend"
    fi
else
    # Generate secret key and config for new installation
    echo "[*] Generating secret key..."
    SECRET_KEY=$(openssl rand -hex 16 | sed 's/^/srv_/')
    HOSTNAME=$(hostname)

    # Auto-detect and configure Kafka for enterprise deployments
detect_kafka_config() {
    local kafka_brokers=""
    local kafka_enabled="false"
    
    echo "[*] Configuring Kafka for worldwide deployment..."
    
    # 1. Check explicit environment variable (for custom Kafka setups)
    if [ -n "$SERVEREYE_KAFKA_BROKERS" ]; then
        kafka_brokers="$SERVEREYE_KAFKA_BROKERS"
        kafka_enabled="true"
        echo "[OK] Using Kafka brokers from environment: $kafka_brokers"
    # 2. Use public Kafka broker for worldwide deployment
    elif curl -s -m 5 "https://api.servereye.dev/health" > /dev/null 2>&1; then
        kafka_brokers="demo-upstash-kafka.upstash.io:9092"
        kafka_enabled="true"
        echo "[OK] Configured for worldwide deployment with public Kafka"
        echo "[INFO] Using public broker: $kafka_brokers"
    # 3. Check for local development (localhost only)
    elif nc -z localhost 9092 2>/dev/null; then
        kafka_brokers="localhost:9092"
        kafka_enabled="true"
        echo "[OK] Detected local Kafka for development: $kafka_brokers"
        echo "[WARNING] For production deployment, public Kafka will be used"
    # 4. No configuration available - use localhost fallback
    else
        echo "[INFO] No Kafka detected - using localhost fallback"
        kafka_brokers="localhost:9092"
        kafka_enabled="true"
        echo "[WARNING] Please start Kafka or configure SERVEREYE_KAFKA_BROKERS"
    fi
    
    # Export for later use
    export KAFKA_BROKERS="$kafka_brokers"
    export KAFKA_ENABLED="$kafka_enabled"
}

# Detect Kafka configuration early
detect_kafka_config

    # Create configuration file
    echo "[*] Creating configuration..."
    
    # Build configuration based on deployment type
    if [ "$KAFKA_ENABLED" = "true" ]; then
        cat > "$CONFIG_DIR/config.yaml" << EOF
server:
  name: "$HOSTNAME"
  description: "ServerEye monitored server"
  secret_key: "$SECRET_KEY"

api:
  base_url: "$SERVEREYE_API_URL"
  timeout: "30s"

# Kafka enabled for worldwide command processing
kafka:
  enabled: true
  brokers:
    - "$KAFKA_BROKERS"
  topic_prefix: "servereye"
  compression: "snappy"
  max_attempts: 3
  batch_size: 100
  required_acks: 1

metrics:
  cpu_temperature: true
  interval: "30s"

logging:
  level: "info"
  file: "$LOG_DIR/agent.log"
EOF
        echo "[OK] Kafka configured for worldwide deployment: $KAFKA_BROKERS"
    else
        echo "[ERROR] Kafka configuration failed - please check connectivity"
        exit 1
    fi

    chown root:$AGENT_USER "$CONFIG_DIR/config.yaml"
    chmod 640 "$CONFIG_DIR/config.yaml"

    # Register key with bot
    echo "[*] Registering key with ServerEye bot..."
    AGENT_VERSION=$("$AGENT_DIR/servereye-agent" --version 2>/dev/null | grep -oP 'ServerEye Agent v\K[0-9.]+' || echo "unknown")
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

    echo "[*] Registering key with backend API..."
    if register_key_with_api "$SECRET_KEY" "$AGENT_VERSION" "$OS_INFO" "$HOSTNAME"; then
        echo "[OK] Key registered with backend"
    else
        echo "[WARNING] Backend registration failed; agent will retry on start"
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
