#!/bin/bash

# Deployment script for ServerEye enterprise stack to home server
set -e

SERVER_IP="192.168.0.104"
SERVER_USER="server"
REMOTE_DIR="/opt/servereye"
DOMAIN="${DOMAIN:-servereye.local}"

echo "🚀 Deploying ServerEye enterprise stack to $SERVER_IP..."

# Check if SSH key authentication works
if ! ssh -o BatchMode=yes -o ConnectTimeout=5 $SERVER_USER@$SERVER_IP exit &>/dev/null; then
    echo "❌ SSH key authentication failed. Please set up SSH keys first:"
    echo "   ssh-copy-id $SERVER_USER@$SERVER_IP"
    exit 1
fi

# Create production docker-compose for external access
echo "📝 Creating production configuration for external access..."
cat > docker-compose.server.yml << EOF
version: '3.8'

services:
  # ZooKeeper for Kafka
  zookeeper:
    image: confluentinc/cp-zookeeper:7.5.0
    hostname: zookeeper
    container_name: servereye-zookeeper
    ports:
      - "2181:2181"
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    volumes:
      - zookeeper_data:/var/lib/zookeeper/data
      - zookeeper_logs:/var/lib/zookeeper/log
    networks:
      - servereye-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "2181"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Kafka broker with external access
  kafka:
    image: confluentinc/cp-kafka:7.5.0
    hostname: kafka
    container_name: servereye-kafka
    depends_on:
      zookeeper:
        condition: service_healthy
    ports:
      # Internal PLAINTEXT for Docker network
      - "9092:9092"
      # External PLAINTEXT for agents (WORLD ACCESS)
      - "9093:9093"
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      
      # Dual listeners configuration
      KAFKA_LISTENERS: INTERNAL://0.0.0.0:9092,EXTERNAL://0.0.0.0:9093
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: INTERNAL:PLAINTEXT,EXTERNAL:PLAINTEXT
      KAFKA_INTER_BROKER_LISTENER_NAME: INTERNAL
      KAFKA_ADVERTISED_LISTENERS: INTERNAL://kafka:9092,EXTERNAL://$SERVER_IP:9093
      
      # Performance settings
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: 0
      
    volumes:
      - kafka_data:/var/lib/kafka/data
    networks:
      - servereye-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "kafka-topics", "--bootstrap-server", "kafka:9092", "--list"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Backend API server
  backend:
    build:
      context: .
      dockerfile: backend/Dockerfile
    container_name: servereye-backend
    depends_on:
      kafka:
        condition: service_healthy
    ports:
      - "8080:8080"
    environment:
      HOST: "0.0.0.0"
      PORT: "8080"
      KAFKA_BROKERS: "kafka:9093"
      KAFKA_TOPIC_PREFIX: "metrics"
      API_KEY: "\${API_KEY:-prod-api-key-2024-change-me}"
    networks:
      - servereye-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Nginx reverse proxy
  nginx:
    image: nginx:1.25-alpine
    container_name: servereye-nginx
    depends_on:
      backend:
        condition: service_healthy
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - ./certbot/conf:/etc/letsencrypt:ro
      - ./certbot/www:/var/www/certbot:ro
    networks:
      - servereye-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Certbot for Let's Encrypt certificates
  certbot:
    image: certbot/certbot:latest
    container_name: servereye-certbot
    volumes:
      - ./certbot/conf:/etc/letsencrypt
      - ./certbot/www:/var/www/certbot
    entrypoint: ["/bin/sh", "-c", "trap exit TERM; while :; do certbot renew; sleep 12h & wait \$\{!}; done;"]
    restart: unless-stopped

volumes:
  zookeeper_data:
  zookeeper_logs:
  kafka_data:

networks:
  servereye-network:
    driver: bridge
EOF

# Create server environment file
cat > .env.server << EOF
# API Configuration
API_KEY=prod-api-key-2024-change-me

# Domain Configuration
DOMAIN=$DOMAIN

# Email for Let's Encrypt
LETSENCRYPT_EMAIL=admin@$DOMAIN
EOF

# Create remote directory structure (using user's home to avoid sudo)
echo "📁 Creating remote directory structure..."
ssh $SERVER_USER@$SERVER_IP "mkdir -p ~/servereye/{nginx/conf.d,certbot/{conf,www},scripts,logs}"
REMOTE_DIR="~/servereye"

# Sync files to server
echo "📦 Syncing files to server..."
rsync -avz --progress \
    --exclude='.git' \
    --exclude='*.log' \
    --exclude='kafka-security' \
    --exclude='backend/main-simple.go' \
    ./ $SERVER_USER@$SERVER_IP:$REMOTE_DIR/

# Copy server-specific configurations
scp docker-compose.server.yml $SERVER_USER@$SERVER_IP:$REMOTE_DIR/docker-compose.yml
scp .env.server $SERVER_USER@$SERVER_IP:$REMOTE_DIR/.env

# Create setup script on server
echo "🔧 Creating setup script on server..."
ssh $SERVER_USER@$SERVER_IP "cat > $REMOTE_DIR/scripts/setup.sh << 'EOFSETUP'
#!/bin/bash
set -e

echo '🐳 Installing Docker...'
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    sudo usermod -aG docker \$USER
    echo 'Docker installed. Please log out and log back in to use Docker without sudo.'
fi

echo '📦 Installing Docker Compose...'
if ! command -v docker-compose &> /dev/null; then
    sudo curl -L \"https://github.com/docker/compose/releases/latest/download/docker-compose-\$(uname -s)-\$(uname -m)\" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
fi

echo '🔥 Configuring firewall...'
sudo ufw --force enable 2>/dev/null || echo 'ufw not available, skipping firewall setup'
sudo ufw allow ssh 2>/dev/null || true
sudo ufw allow 80/tcp 2>/dev/null || true
sudo ufw allow 443/tcp 2>/dev/null || true
sudo ufw allow 9093/tcp 2>/dev/null || true

echo '✅ Setup complete!'
EOFSETUP"

# Make setup script executable
ssh $SERVER_USER@$SERVER_IP "chmod +x $REMOTE_DIR/scripts/setup.sh"

# Run setup script
echo "🚀 Running setup on server..."
ssh -t $SERVER_USER@$SERVER_IP "cd $REMOTE_DIR && ./scripts/setup.sh"

# Create deployment script on server
echo "📝 Creating deployment script on server..."
ssh $SERVER_USER@$SERVER_IP "cat > $REMOTE_DIR/scripts/deploy.sh << 'EOFDEPLOY'
#!/bin/bash
set -e

echo '🛑 Stopping existing services...'
docker-compose down || true

echo '🔨 Building and starting services...'
docker-compose up -d --build

echo '⏳ Waiting for services to be healthy...'
sleep 30

echo '📊 Checking service status...'
docker-compose ps

echo '🧪 Testing API health...'
curl -f http://localhost:8080/health || echo 'Backend not ready yet'

echo '🎉 Deployment complete!'
echo ''
echo 'Services:'
echo '  - Kafka: $SERVER_IP:9093 (external)'
echo '  - API: http://$SERVER_IP:8080'
echo '  - Nginx: http://$SERVER_IP'
echo ''
echo 'To test with agent, use:'
echo "  api:"
echo "    base_url: \"http://$SERVER_IP:8080\""
echo "    api_key: \"prod-api-key-2024-change-me\""
EOFDEPLOY"

# Make deployment script executable
ssh $SERVER_USER@$SERVER_IP "chmod +x $REMOTE_DIR/scripts/deploy.sh"

echo ""
echo "✅ Preparation complete! To deploy to server, run:"
echo "   ssh $SERVER_USER@$SERVER_IP 'cd $REMOTE_DIR && ./scripts/deploy.sh'"
echo ""
echo "📋 Next steps:"
echo "1. SSH into server: ssh $SERVER_USER@$SERVER_IP"
echo "2. Run deployment: cd $REMOTE_DIR && ./scripts/deploy.sh"
echo "3. Test with agent using external IP: $SERVER_IP"
echo ""
echo "🔐 For SSL certificates (optional):"
echo "   ssh $SERVER_USER@$SERVER_IP 'cd $REMOTE_DIR && ./scripts/setup-ssl.sh'"
