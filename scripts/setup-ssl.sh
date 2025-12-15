#!/bin/bash

# Setup SSL certificates with Let's Encrypt
set -e

DOMAIN=${DOMAIN:-servereye.local}
EMAIL=${LETSENCRYPT_EMAIL:-admin@servereye.local}

echo "Setting up SSL certificates for domain: $DOMAIN"
echo "Email: $EMAIL"

# Create certbot directories if they don't exist
mkdir -p certbot/conf certbot/www

# Generate initial certificates (staging for testing)
echo "Generating initial certificates (staging)..."
docker-compose -f docker-compose.prod.yml run --rm --entrypoint "\
  certbot certonly --webroot --webroot-path=/var/www/certbot \
    --email $EMAIL \
    --agree-tos \
    --no-eff-email \
    --staging \
    -d $DOMAIN" certbot

echo "Staging certificates generated successfully!"
echo ""
echo "To generate production certificates, run:"
echo "./scripts/setup-ssl-prod.sh"
echo ""
echo "After certificates are generated, restart nginx:"
echo "docker-compose -f docker-compose.prod.yml restart nginx"
