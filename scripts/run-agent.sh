#!/bin/bash

# ServerEye Agent Runner with Auto-restart
# Handles Cloudflare tunnel instability with graceful restarts

AGENT_DIR="/home/gospodin/Рабочий стол/homeProjects/ServerEye"
CONFIG_FILE="$AGENT_DIR/examples/config-cloudflare.yaml"
AGENT_BIN="$AGENT_DIR/dist/servereye-agent"
LOG_FILE="/var/log/servereye/agent.log"
MAX_RESTARTS=10
RESTART_DELAY=30

# Create log directory
sudo mkdir -p /var/log/servereye
sudo chown $USER:$USER /var/log/servereye

echo "$(date): Starting ServerEye Agent with auto-restart"
echo "$(date): Agent binary: $AGENT_BIN"
echo "$(date): Config file: $CONFIG_FILE"

restart_count=0

while [ $restart_count -lt $MAX_RESTARTS ]; do
    echo "$(date): Starting agent (attempt $((restart_count + 1))/$MAX_RESTARTS)"
    
    # Run agent with timeout to prevent hanging
    timeout 300s $AGENT_BIN -config $CONFIG_FILE 2>&1 | tee -a $LOG_FILE
    
    exit_code=$?
    
    if [ $exit_code -eq 124 ]; then
        echo "$(date): Agent timeout, restarting..."
    elif [ $exit_code -eq 0 ]; then
        echo "$(date): Agent exited normally"
        break
    else
        echo "$(date): Agent exited with code $exit_code, restarting..."
    fi
    
    restart_count=$((restart_count + 1))
    
    if [ $restart_count -lt $MAX_RESTARTS ]; then
        echo "$(date): Waiting $RESTART_DELAY seconds before restart..."
        sleep $RESTART_DELAY
    fi
done

if [ $restart_count -eq $MAX_RESTARTS ]; then
    echo "$(date): Maximum restarts reached, stopping"
    exit 1
fi

echo "$(date): Agent runner completed successfully"
