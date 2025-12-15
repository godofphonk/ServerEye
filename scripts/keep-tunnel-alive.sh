#!/bin/bash

# Cloudflare Tunnel Auto-Restart Script
# Handles WiFi instability by monitoring and restarting tunnel

TUNNEL_NAME="servereye-api"
LOG_FILE="/tmp/cloudflared.log"
PID_FILE="/tmp/cloudflared.pid"
CHECK_INTERVAL=30  # seconds

echo "$(date): Starting Cloudflare tunnel keeper for $TUNNEL_NAME"

while true; do
    # Check if tunnel process is running
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p "$PID" > /dev/null 2>&1; then
            echo "$(date): Tunnel process $PID is running"
        else
            echo "$(date): Tunnel process $PID not found, removing stale PID file"
            rm -f "$PID_FILE"
        fi
    fi
    
    # Start tunnel if not running
    if [ ! -f "$PID_FILE" ]; then
        echo "$(date): Starting Cloudflare tunnel $TUNNEL_NAME"
        nohup ~/bin/cloudflared tunnel run --protocol http2 "$TUNNEL_NAME" > "$LOG_FILE" 2>&1 &
        echo $! > "$PID_FILE"
        sleep 5
        
        # Verify tunnel started successfully
        if ps -p "$(cat $PID_FILE)" > /dev/null 2>&1; then
            echo "$(date): Tunnel started successfully with PID $(cat $PID_FILE)"
        else
            echo "$(date): Failed to start tunnel, will retry in $CHECK_INTERVAL seconds"
            rm -f "$PID_FILE"
        fi
    fi
    
    # Test tunnel connectivity
    if [ -f "$PID_FILE" ]; then
        echo "$(date): Testing tunnel connectivity..."
        if curl -s -m 5 "https://metrics.servereye.dev/health" > /dev/null 2>&1; then
            echo "$(date): Tunnel connectivity OK"
        else
            echo "$(date): Tunnel connectivity failed, restarting..."
            kill "$(cat $PID_FILE)" 2>/dev/null || true
            rm -f "$PID_FILE"
        fi
    fi
    
    sleep "$CHECK_INTERVAL"
done
