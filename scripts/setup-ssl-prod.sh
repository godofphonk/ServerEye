#!/bin/bash

# Setup production SSL certificates with Let's Encrypt
set -e

DOMAIN=${DOMAIN:-servereye.local}
EMAIL=${LETSENCRYPT_EMAIL:-admin@servereye.local}

echo "Setting up PRODUCTION SSL certificates for domain: $DOMAIN"
echo "Email: $EMAIL"
echo ""
echo "WARNING: This will generate REAL certificates that count against rate limits!"
echo "Make sure your domain is properly configured to point to this server."
read -p "Continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled."
    exit 1
fi

# Create certbot directories if they don't exist
mkdir -p certbot/conf certbot/www

# Generate production certificates
echo "Generating production certificates..."
docker-compose -f docker-compose.prod.yml run --rm --entrypoint "\
  certbot certonly --webroot --webroot-path=/var/www/certbot \
    --email $EMAIL \
    --agree-tos \
    --no-eff-email \
    -d $DOMAIN" certbot

echo "Production certificates generated successfully!"
echo ""
echo "Restarting nginx to apply new certificates..."
docker-compose -f docker-compose.prod.yml restart nginx

echo ""
echo "Setup complete! Your site is now available at: https://$DOMAIN"
