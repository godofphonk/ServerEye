#!/bin/bash
# ServerEye Quick Installer
# Usage: wget -qO- https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install.sh | sudo bash

set -e

INSTALL_SCRIPT_URL="https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install-agent.sh"

echo "🚀 ServerEye Quick Installer"
echo "Downloading and running installation script..."

# Download and execute the main installer
wget -qO- "$INSTALL_SCRIPT_URL" | bash

echo "✅ ServerEye installation completed!"
echo "Check the output above for your secret key and next steps."
