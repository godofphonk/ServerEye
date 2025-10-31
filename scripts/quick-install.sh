#!/bin/bash
# ServerEye One-Line Installer
# Usage: bash <(wget -qO- https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/quick-install.sh)

echo "🚀 ServerEye One-Line Installer"
echo "Downloading installation script..."

# Download and run the installer directly
wget -qO- https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install-agent.sh | sudo bash
