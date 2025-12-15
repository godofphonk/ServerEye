#!/bin/bash

# Create SCRAM-SHA-512 users in Kafka
set -e

echo "Creating SCRAM-SHA-512 users in Kafka..."

# Wait for Kafka to be ready
echo "Waiting for Kafka to start..."
docker exec servereye-kafka-ssl bash -c 'cub kafka-ready -b localhost:9092 1 30'

# Create admin user
echo "Creating admin user..."
docker exec servereye-kafka-ssl kafka-configs.sh --bootstrap-server localhost:9092 --alter --add-config 'SCRAM-SHA-512=[password=admin-secret-2024]' --entity-type users --entity-name admin

# Create agent user
echo "Creating agent user..."
docker exec servereye-kafka-ssl kafka-configs.sh --bootstrap-server localhost:9092 --alter --add-config 'SCRAM-SHA-512=[password=agent-secret-2024]' --entity-type users --entity-name servereye-agent

# Create bot user
echo "Creating bot user..."
docker exec servereye-kafka-ssl kafka-configs.sh --bootstrap-server localhost:9092 --alter --add-config 'SCRAM-SHA-512=[password=bot-secret-2024]' --entity-type users --entity-name servereye-bot

# List users to verify
echo "Verifying created users..."
docker exec servereye-kafka-ssl kafka-configs.sh --bootstrap-server localhost:9092 --describe --entity-type users

echo "SCRAM-SHA-512 users created successfully!"
echo "Users created:"
echo "  - admin: admin-secret-2024"
echo "  - servereye-agent: agent-secret-2024"
echo "  - servereye-bot: bot-secret-2024"
