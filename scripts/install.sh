#!/bin/bash

# ServerEye Installation Script
# This script installs ServerEye agent on Linux systems

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="godofphonk/ServerEye"
INSTALL_DIR="/opt/servereye"
CONFIG_DIR="/etc/servereye"
SERVICE_NAME="servereye-agent"
USER="servereye"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Detect system architecture
detect_arch() {
    local arch=$(uname -m)
    case $arch in
        x86_64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        armv7l)
            echo "arm"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Get latest release version
get_latest_version() {
    log_info "Using master branch (no releases yet)"
    echo "master"
}

# Download and install binary
install_binary() {
    local version=$1
    local arch=$2
    
    log_info "Installing ServerEye agent..."
    
    # Create directories
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    
    # Download binary (for now, we'll build from source since we don't have releases yet)
    log_info "Downloading ServerEye source code..."
    
    # Install dependencies
    if command -v apt-get &> /dev/null; then
        apt-get update
        apt-get install -y git golang-go
    elif command -v yum &> /dev/null; then
        yum install -y git golang
    elif command -v dnf &> /dev/null; then
        dnf install -y git golang
    else
        log_error "Package manager not supported. Please install git and golang manually."
        exit 1
    fi
    
    # Clone and build
    cd /tmp
    rm -rf ServerEye
    git clone "https://github.com/$GITHUB_REPO.git"
    cd ServerEye
    
    log_info "Building ServerEye agent..."
    export CGO_ENABLED=0
    go build -ldflags="-s -w" -o servereye-agent ./cmd/agent
    
    # Install binary
    cp servereye-agent "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/servereye-agent"
    
    # Test binary exists
    if [[ ! -f "$INSTALL_DIR/servereye-agent" ]]; then
        log_error "Failed to build ServerEye agent"
        exit 1
    fi
    
    log_success "ServerEye agent installed to $INSTALL_DIR"
}

# Create user
create_user() {
    if ! id "$USER" &>/dev/null; then
        log_info "Creating user $USER..."
        useradd --system --shell /bin/false --home-dir "$INSTALL_DIR" --no-create-home "$USER"
        log_success "User $USER created"
    else
        log_info "User $USER already exists"
    fi
}

# Create configuration
create_config() {
    log_info "Creating configuration..."
    
    # Generate random server key
    local server_key="srv_$(openssl rand -hex 16)"
    
    cat > "$CONFIG_DIR/config.yaml" << EOF
server:
  name: "$(hostname)"
  secret_key: "$server_key"

redis:
  address: "localhost:6379"
  password: ""
  db: 0

logging:
  level: "info"
  format: "json"
EOF

    # Set permissions
    chown -R "$USER:$USER" "$CONFIG_DIR"
    chmod 600 "$CONFIG_DIR/config.yaml"
    
    log_success "Configuration created at $CONFIG_DIR/config.yaml"
    log_warning "Your server key is: $server_key"
    log_warning "Save this key! You'll need it to connect to your Telegram bot."
}

# Create systemd service
create_service() {
    log_info "Creating systemd service..."
    
    cat > "/etc/systemd/system/$SERVICE_NAME.service" << EOF
[Unit]
Description=ServerEye Agent
After=network.target
Wants=network.target

[Service]
Type=simple
User=$USER
Group=$USER
ExecStart=$INSTALL_DIR/servereye-agent -config $CONFIG_DIR/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    # Reload systemd and enable service
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    
    log_success "Systemd service created and enabled"
}

# Install Redis (optional)
install_redis() {
    if command -v redis-server &> /dev/null; then
        log_info "Redis already installed"
        return
    fi
    
    read -p "Redis is required for ServerEye. Install Redis? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "Installing Redis..."
        if command -v apt-get &> /dev/null; then
            apt-get install -y redis-server
            systemctl enable redis-server
            systemctl start redis-server
        elif command -v yum &> /dev/null; then
            yum install -y redis
            systemctl enable redis
            systemctl start redis
        elif command -v dnf &> /dev/null; then
            dnf install -y redis
            systemctl enable redis
            systemctl start redis
        fi
        log_success "Redis installed and started"
    else
        log_warning "Redis not installed. You'll need to install and configure Redis manually."
    fi
}

# Start service
start_service() {
    log_info "Starting ServerEye agent..."
    systemctl start "$SERVICE_NAME"
    
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log_success "ServerEye agent started successfully"
    else
        log_error "Failed to start ServerEye agent"
        log_info "Check logs with: journalctl -u $SERVICE_NAME -f"
        exit 1
    fi
}

# Show final instructions
show_instructions() {
    local server_key=$(grep "secret_key:" "$CONFIG_DIR/config.yaml" | awk '{print $2}' | tr -d '"')
    
    echo
    log_success "ServerEye installation completed!"
    echo
    echo -e "${BLUE}Next steps:${NC}"
    echo "1. Start your Telegram bot: https://t.me/YourServerEyeBot"
    echo "2. Send this server key to the bot: ${GREEN}$server_key${NC}"
    echo "3. Try commands like: /temp, /memory, /disk, /containers"
    echo
    echo -e "${BLUE}Useful commands:${NC}"
    echo "• Check status: systemctl status $SERVICE_NAME"
    echo "• View logs: journalctl -u $SERVICE_NAME -f"
    echo "• Restart: systemctl restart $SERVICE_NAME"
    echo "• Config file: $CONFIG_DIR/config.yaml"
    echo
}

# Main installation function
main() {
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════╗"
    echo "║           ServerEye Installer        ║"
    echo "║     Server Monitoring via Telegram   ║"
    echo "╚══════════════════════════════════════╝"
    echo -e "${NC}"
    
    check_root
    
    local arch=$(detect_arch)
    local version=$(get_latest_version)
    
    log_info "Installing ServerEye $version for $arch"
    
    create_user
    install_binary "$version" "$arch"
    create_config
    create_service
    install_redis
    start_service
    show_instructions
}

# Run main function
main "$@"
