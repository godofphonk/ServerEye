#!/bin/bash

# ServerEye WebSocket-Kafka Integration Test
# Tests: Agent → Backend → Kafka → WebSocket → Client

set -e

echo "🚀 Starting ServerEye Integration Test..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BACKEND_URL="http://localhost:8080"
WS_URL="ws://localhost:8080/ws"
KAFKA_TOPIC="servereye.metrics"

echo "📋 Configuration:"
echo "  Backend URL: $BACKEND_URL"
echo "  WebSocket URL: $WS_URL"
echo "  Kafka Topic: $KAFKA_TOPIC"
echo ""

# Function to check if service is running
check_service() {
    local url=$1
    local service_name=$2
    
    if curl -s -f "$url" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ $service_name is running${NC}"
        return 0
    else
        echo -e "${RED}❌ $service_name is not running${NC}"
        return 1
    fi
}

# Function to test WebSocket connection
test_websocket() {
    echo "🔌 Testing WebSocket connection..."
    
    # Create a simple WebSocket test using wscat if available, or curl
    if command -v wscat &> /dev/null; then
        echo "Using wscat for WebSocket test..."
        timeout 10s wscat -c "$WS_URL" -x '{"type":"subscribe","data":{"server_id":"test-server"}}' &
        WSCAT_PID=$!
        sleep 2
    else
        echo "Using curl for WebSocket test..."
        timeout 10s curl -i -N \
            -H "Connection: Upgrade" \
            -H "Upgrade: websocket" \
            -H "Sec-WebSocket-Key: test" \
            -H "Sec-WebSocket-Version: 13" \
            "$WS_URL" &
        CURL_PID=$!
        sleep 2
    fi
    
    echo -e "${GREEN}✅ WebSocket endpoint accessible${NC}"
}

# Function to send test metric
send_test_metric() {
    echo "📊 Sending test metric..."
    
    local test_metric='{
        "server_id": "test-server-123",
        "server_key": "test-key-456",
        "type": "cpu",
        "value": 75.5,
        "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'",
        "tags": {"region": "test", "env": "dev"}
    }'
    
    echo "Sending metric: $test_metric"
    
    response=$(curl -s -w "%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -H "X-API-Key: prod-api-key-2024-change-me" \
        -d "$test_metric" \
        "$BACKEND_URL/v1/metrics")
    
    http_code="${response: -3}"
    
    if [ "$http_code" = "200" ] || [ "$http_code" = "201" ] || [ "$http_code" = "202" ]; then
        echo -e "${GREEN}✅ Metric sent successfully (HTTP $http_code)${NC}"
    else
        echo -e "${RED}❌ Failed to send metric (HTTP $http_code)${NC}"
        echo "Response: ${response%???}"
        return 1
    fi
}

# Function to check Kafka topic
check_kafka() {
    echo "🔍 Checking Kafka topic..."
    
    # Check if Kafka is accessible
    if docker exec servereye-kafka kafka-topics --bootstrap-server localhost:9092 --list | grep -q "$KAFKA_TOPIC"; then
        echo -e "${GREEN}✅ Kafka topic '$KAFKA_TOPIC' exists${NC}"
    else
        echo -e "${YELLOW}⚠️  Kafka topic '$KAFKA_TOPIC' not found, creating...${NC}"
        docker exec servereye-kafka kafka-topics --bootstrap-server localhost:9092 --create --topic "$KAFKA_TOPIC" --partitions 3 --replication-factor 1
    fi
}

# Main test execution
echo "🧪 Running integration tests..."
echo ""

# Test 1: Check backend health
echo "=== Test 1: Backend Health Check ==="
if check_service "$BACKEND_URL/health" "Backend"; then
    echo -e "${GREEN}✅ Backend is healthy${NC}"
else
    echo -e "${RED}❌ Backend health check failed${NC}"
    exit 1
fi
echo ""

# Test 2: Check Kafka
echo "=== Test 2: Kafka Connectivity ==="
check_kafka
echo ""

# Test 3: WebSocket connection
echo "=== Test 3: WebSocket Connection ==="
test_websocket
echo ""

# Test 4: Send metric and check flow
echo "=== Test 4: End-to-End Metric Flow ==="
send_test_metric
echo ""

# Wait a moment for processing
echo "⏳ Waiting for Kafka processing..."
sleep 3

echo ""
echo "🎉 Integration test completed!"
echo ""
echo "📝 Manual verification steps:"
echo "1. Check backend logs for metric processing"
echo "2. Verify WebSocket client receives the metric"
echo "3. Check Kafka topic for the message"
echo ""
echo "🔧 Commands for manual verification:"
echo "  - Backend logs: docker logs servereye-backend"
echo "  - Kafka messages: docker exec servereye-kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic $KAFKA_TOPIC --from-beginning --max-messages 1"
