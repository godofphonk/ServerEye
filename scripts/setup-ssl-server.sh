#!/bin/bash

# SSL setup script for ServerEye on home server
set -e

SERVER_IP="192.168.0.104"
SERVER_USER="server"
DOMAIN="${DOMAIN:-servereye.local}"

echo "🔐 Setting up SSL certificates for $DOMAIN on $SERVER_IP..."

# Create SSL setup script on server
ssh $SERVER_USER@$SERVER_IP "cat > /opt/servereye/scripts/setup-ssl.sh << 'EOFSSL'
#!/bin/bash
set -e

DOMAIN=\"\${DOMAIN:-servereye.local}\"
EMAIL=\"\${LETSENCRYPT_EMAIL:-admin@servereye.local}\"

echo \"Setting up SSL certificates for domain: \$DOMAIN\"
echo \"Email: \$EMAIL\"

# Create certbot directories if they don't exist
mkdir -p certbot/conf certbot/www

# Check if domain points to this server
echo \"⚠️  Make sure \$DOMAIN points to \$(curl -s ifconfig.me)\"
read -p \"Continue? (y/N) \" -n 1 -r
echo
if [[ ! \$REPLY =~ ^[Yy]\$ ]]; then
    echo \"Cancelled.\"
    exit 1
fi

# Generate certificates (staging first for testing)
echo \"🔧 Generating staging certificates...\"
docker-compose run --rm --entrypoint \"
  certbot certonly --webroot --webroot-path=/var/www/certbot \\
    --email \$EMAIL \\
    --agree-tos \\
    --no-eff-email \\
    --staging \\
    -d \$DOMAIN\" certbot

echo \"✅ Staging certificates generated!\"
echo \"\"
echo \"To generate production certificates, run:\"
echo \"  ./scripts/setup-ssl-prod.sh\"
echo \"\"
echo \"After certificates are ready, restart nginx:\"
echo \"  docker-compose restart nginx\"
EOFSSL"

# Make SSL script executable
ssh $SERVER_USER@$SERVER_IP "chmod +x /opt/servereye/scripts/setup-ssl.sh"

# Create production SSL script
ssh $SERVER_USER@$SERVER_IP "cat > /opt/servereye/scripts/setup-ssl-prod.sh << 'EOFPROD'
#!/bin/bash
set -e

DOMAIN=\"\${DOMAIN:-servereye.local}\"
EMAIL=\"\${LETSENCRYPT_EMAIL:-admin@servereye.local}\"

echo \"🚨 Generating PRODUCTION SSL certificates for \$DOMAIN\"
echo \"Email: \$EMAIL\"
echo \"\"
echo \"WARNING: This will generate REAL certificates!\"
read -p \"Continue? (y/N) \" -n 1 -r
echo
if [[ ! \$REPLY =~ ^[Yy]\$ ]]; then
    echo \"Cancelled.\"
    exit 1
fi

# Generate production certificates
echo \"🔧 Generating production certificates...\"
docker-compose run --rm --entrypoint \"
  certbot certonly --webroot --webroot-path=/var/www/certbot \\
    --email \$EMAIL \\
    --agree-tos \\
    --no-eff-email \\
    -d \$DOMAIN\" certbot

echo \"✅ Production certificates generated!\"
echo \"\"
echo \"🔄 Restarting nginx...\"
docker-compose restart nginx

echo \"\"
echo \"🎉 SSL setup complete!\"
echo \"Your site is now available at: https://\$DOMAIN\"
echo \"\"
echo \"📋 Certificate renewal is automatic via certbot container.\"
EOFPROD"

# Make production SSL script executable
ssh $SERVER_USER@$SERVER_IP "chmod +x /opt/servereye/scripts/setup-ssl-prod.sh"

echo ""
echo "✅ SSL scripts created on server!"
echo ""
echo "📋 To set up SSL certificates:"
echo "1. SSH into server: ssh $SERVER_USER@$SERVER_IP"
echo "2. Go to project: cd /opt/servereye"
echo "3. Test with staging: ./scripts/setup-ssl.sh"
echo "4. Generate production: ./scripts/setup-ssl-prod.sh"
echo ""
echo "📝 Note: Make sure your domain $DOMAIN points to $SERVER_IP"
