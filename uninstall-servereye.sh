#!/bin/bash

# ServerEye Agent Complete Uninstall Script
# Run with sudo: sudo ./uninstall-servereye.sh

set -e  # Exit on any error

echo "🗑️  ServerEye Agent Uninstall Script"
echo "==================================="
echo "This script will completely remove ServerEye Agent from your system"
echo ""

# Check if running as root/sudo
if [ "$EUID" -ne 0 ]; then
    echo "❌ This script must be run as root or with sudo"
    echo "   Usage: sudo ./uninstall-servereye.sh"
    exit 1
fi

echo "✅ Running with root privileges"

# 1. Stop Agent Service
echo ""
echo "🛑 Step 1: Stopping ServerEye Agent service..."
if systemctl is-active --quiet servereye-agent 2>/dev/null; then
    systemctl stop servereye-agent
    echo "✅ ServerEye Agent service stopped"
else
    echo "ℹ️  ServerEye Agent service was not running"
fi

# 2. Disable and Remove Service
echo ""
echo "🗑️  Step 2: Removing systemd service..."
if systemctl list-unit-files | grep -q "servereye-agent.service"; then
    systemctl disable servereye-agent
    rm -f /etc/systemd/system/servereye-agent.service
    systemctl daemon-reload
    echo "✅ Systemd service removed"
else
    echo "ℹ️  Systemd service not found"
fi

# 3. Remove Binary and Configuration
echo ""
echo "🗑️  Step 3: Removing agent files and configuration..."
if [ -d "/opt/servereye" ]; then
    rm -rf /opt/servereye
    echo "✅ Binary directory /opt/servereye removed"
else
    echo "ℹ️  Binary directory /opt/servereye not found"
fi

if [ -d "/etc/servereye" ]; then
    rm -rf /etc/servereye
    echo "✅ Configuration directory /etc/servereye removed"
else
    echo "ℹ️  Configuration directory /etc/servereye not found"
fi

if [ -d "/var/log/servereye" ]; then
    rm -rf /var/log/servereye
    echo "✅ Log directory /var/log/servereye removed"
else
    echo "ℹ️  Log directory /var/log/servereye not found"
fi

# 4. Remove System User
echo ""
echo "🗑️  Step 4: Removing system user..."
if id "servereye" &>/dev/null; then
    userdel servereye
    echo "✅ System user 'servereye' removed"
else
    echo "ℹ️  System user 'servereye' not found"
fi

if getent group servereye &>/dev/null; then
    groupdel servereye
    echo "✅ System group 'servereye' removed"
else
    echo "ℹ️  System group 'servereye' not found"
fi

# 5. Clean Temporary Files
echo ""
echo "🗑️  Step 5: Cleaning temporary files..."
rm -rf /tmp/servereye-*
echo "✅ Temporary files cleaned"

# 6. Verification
echo ""
echo "🔍 Step 6: Verification..."
echo "Checking for remaining ServerEye processes..."

if pgrep -f "servereye-agent" > /dev/null; then
    echo "⚠️  WARNING: ServerEye processes still running:"
    ps aux | grep servereye-agent | grep -v grep
else
    echo "✅ No ServerEye processes found"
fi

echo ""
echo "Checking for remaining files..."
for dir in "/opt/servereye" "/etc/servereye" "/var/log/servereye"; do
    if [ -d "$dir" ]; then
        echo "⚠️  WARNING: Directory still exists: $dir"
    else
        echo "✅ Directory removed: $dir"
    fi
done

echo ""
echo "Checking systemd service..."
if systemctl list-unit-files | grep -q "servereye-agent.service"; then
    echo "⚠️  WARNING: Service file still exists"
else
    echo "✅ Systemd service removed"
fi

# 7. Database Cleanup Instructions
echo ""
echo "🗄️  Step 7: Database Cleanup (Optional)"
echo "======================================"
echo "Agent keys are still stored in PostgreSQL database."
echo "To remove them, connect to your database and run:"
echo ""
echo "  DELETE FROM generated_keys WHERE secret_key LIKE 'srv_%';"
echo ""
echo "Or keep them for audit purposes."

echo ""
echo "🎉 ServerEye Agent uninstall completed!"
echo "======================================"
echo ""
echo "Next steps:"
echo "1. Verify no agent processes are running: ps aux | grep servereye"
echo "2. Optional: Clean database keys if needed (see SQL above)"
echo "3. Remove this script if no longer needed"
echo ""
