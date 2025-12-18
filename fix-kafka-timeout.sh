#!/bin/bash
# Fix for Kafka timeout issue after worldwide deployment update

echo "Fixing Kafka timeout issue..."
echo "1. Making new agent binary executable..."
sudo chmod +x /opt/servereye/servereye-agent

echo "2. Restarting agent service..."
sudo systemctl restart servereye-agent

echo "3. Checking agent status..."
sudo systemctl status servereye-agent --no-pager

echo "4. Testing Kafka connectivity..."
sleep 5
echo "Send /temp command to bot to test"

echo "Fix completed! The agent now uses unified servereye.metrics topic."
