#!/bin/bash
# ServerEye Quick Installer
# Usage: curl -sSL https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install.sh | sudo bash

set -e

INSTALL_SCRIPT_URL="https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install-agent.sh"

echo "🚀 ServerEye Quick Installer"
echo "Downloading and running installation script..."

# Download and execute the main installer
curl -sSL "$INSTALL_SCRIPT_URL" | sudo bash

echo "✅ ServerEye installation completed!"
echo "Check the output above for your secret key and next steps."
