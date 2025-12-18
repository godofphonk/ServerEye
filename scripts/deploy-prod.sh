#!/bin/bash

# ServerEye Production Deployment Script
# This script deploys ServerEye in production mode

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
DOMAIN="${DOMAIN:-servereye.dev}"
API_KEY="${API_KEY:-prod-api-key-2024-change-me}"
EMAIL="${EMAIL:-admin@servereye.dev}"

echo -e "${GREEN}Starting ServerEye Production Deployment${NC}"
echo "Domain: $DOMAIN"
echo "Email: $EMAIL"

# Check if .env.prod exists
if [ ! -f .env.prod ]; then
    echo -e "${RED}Error: .env.prod file not found${NC}"
    exit 1
fi

# Load environment variables
source .env.prod

# Check required environment variables
if [ -z "$API_KEY" ]; then
    echo -e "${RED}Error: API_KEY is not set${NC}"
    exit 1
fi

# Create necessary directories
echo -e "${YELLOW}Creating directories...${NC}"
mkdir -p certbot/conf certbot/www logs

# Stop existing containers
echo -e "${YELLOW}Stopping existing containers...${NC}"
docker-compose -f docker-compose.prod.yml down -v

# Pull latest images
echo -e "${YELLOW}Pulling latest images...${NC}"
docker-compose -f docker-compose.prod.yml pull

# Build and start services
echo -e "${YELLOW}Building and starting services...${NC}"
docker-compose -f docker-compose.prod.yml up -d --build

# Wait for services to be healthy
echo -e "${YELLOW}Waiting for services to be healthy...${NC}"
sleep 30

# Check service health
echo -e "${YELLOW}Checking service health...${NC}"
docker-compose -f docker-compose.prod.yml ps

# Generate SSL certificates (if needed)
if [ ! -f "certbot/conf/live/$DOMAIN/fullchain.pem" ]; then
    echo -e "${YELLOW}Generating SSL certificates...${NC}"
    docker-compose -f docker-compose.prod.yml run --rm --entrypoint "\
        certbot certonly --webroot --webroot-path=/var/www/certbot \
        --email $EMAIL --agree-tos --no-eff-email \
        -d $DOMAIN" certbot
fi

# Reload nginx
echo -e "${YELLOW}Reloading nginx...${NC}"
docker-compose -f docker-compose.prod.yml exec nginx nginx -s reload

# Test deployment
echo -e "${YELLOW}Testing deployment...${NC}"
curl -f http://localhost/health || {
    echo -e "${RED}Health check failed${NC}"
    exit 1
}

echo -e "${GREEN}Deployment completed successfully!${NC}"
echo "API is available at: https://$DOMAIN/v1/"
echo "Health check: https://$DOMAIN/health"
