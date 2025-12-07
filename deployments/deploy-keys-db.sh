#!/bin/bash
# ServerEye Key Database Deployment Script
# Run this on your server: server@192.168.0.104

set -e

echo "🚀 Deploying ServerEye Key Registration Database..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
   echo -e "${RED}Please run as root (use sudo)${NC}"
   exit 1
fi

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}Installing Docker...${NC}"
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh
    rm get-docker.sh
    systemctl enable docker
    systemctl start docker
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo -e "${YELLOW}Installing Docker Compose...${NC}"
    curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
fi

# Create ServerEye directory
mkdir -p /opt/servereye
cd /opt/servereye

# Download required files (you'll need to copy these manually or use git)
echo -e "${YELLOW}Please copy the following files to /opt/servereye:${NC}"
echo "  - docker-compose.yml"
echo "  - init-keys-db.sql"
echo "  - .env"
echo ""
echo "After copying files, press Enter to continue..."
read

# Update .env with secure password
echo -e "${YELLOW}Generating secure password for database...${NC}"
SECURE_PASSWORD=$(openssl rand -base64 32)
sed -i "s/KEYS_DB_PASSWORD=change_me/KEYS_DB_PASSWORD=$SECURE_PASSWORD/" .env
sed -i "s/servereye_keys:change_me/servereye_keys:$SECURE_PASSWORD/" .env

echo -e "${GREEN}Generated password: $SECURE_PASSWORD${NC}"
echo "Please save this password for Cloudflare Tunnel configuration!"

# Start only the keys database first
echo -e "${YELLOW}Starting PGRegisteredKeysServerEye container...${NC}"
docker-compose -f docker-compose.yml --env-file .env up -d pg-registered-keys

# Wait for database to be ready
echo -e "${YELLOW}Waiting for database to initialize...${NC}"
sleep 10

# Verify database is running
if docker exec -it PGRegisteredKeysServerEye pg_isready -U servereye_keys > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Database is ready!${NC}"
else
    echo -e "${RED}❌ Database failed to start${NC}"
    exit 1
fi

# Show database info
echo ""
echo -e "${GREEN}=== Database Information ===${NC}"
echo "Container: PGRegisteredKeysServerEye"
echo "Port: 5433"
echo "Database: PgRegisteredKeys"
echo "User: servereye_keys"
echo "Password: $SECURE_PASSWORD"
echo ""
echo "Next steps:"
echo "1. Set up Cloudflare Tunnel for port 5433"
echo "2. Update SERVEREYE_DB_URL with tunnel URL"
echo "3. Start other services: docker-compose up -d"
echo ""
echo -e "${GREEN}Deployment completed successfully!${NC}"
