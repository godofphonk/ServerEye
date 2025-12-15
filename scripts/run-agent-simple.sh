#!/bin/bash

# ServerEye Agent Runner (Simple version)
# Runs agent with basic error handling

AGENT_DIR="/home/gospodin/Рабочий стол/homeProjects/ServerEye"
CONFIG_FILE="$AGENT_DIR/examples/config-cloudflare.yaml"
AGENT_BIN="$AGENT_DIR/dist/servereye-agent"
LOG_DIR="$AGENT_DIR/logs"

# Create log directory
mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/agent.log"

echo "$(date): Starting ServerEye Agent"
echo "$(date): Config: $CONFIG_FILE"
echo "$(date): Log: $LOG_FILE"

# Run agent with logging
$AGENT_BIN -config "$CONFIG_FILE" 2>&1 | tee -a "$LOG_FILE"
