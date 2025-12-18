#!/bin/bash
# Fix for local agent to use server Kafka instead of non-existent public Kafka

echo "Fixing local agent configuration..."

# Backup current config
sudo cp /etc/servereye/config.yaml /etc/servereye/config.yaml.backup

# Create new config with server Kafka
sudo tee /etc/servereye/config.yaml > /dev/null << 'EOF'
server:
  name: "$(hostname)"
  description: "ServerEye monitored server"
  secret_key: "$(grep 'secret_key' /etc/servereye/config.yaml.backup | awk '{print $2}')"

api:
  base_url: "https://api.servereye.dev"
  timeout: "30s"

# Kafka configured for server connection
kafka:
  enabled: true
  brokers:
    - "192.168.0.104:9093"
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
  file: "/var/log/servereye/agent.log"
EOF

echo "Configuration updated. Restarting agent..."
sudo systemctl restart servereye-agent

echo "Checking status..."
sleep 3
sudo systemctl status servereye-agent --no-pager -l

echo "Fixed! Now test with /temp command in bot."
