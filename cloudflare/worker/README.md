# Cloudflare Worker Deployment Guide

This guide will help you deploy the ServerEye registration Worker to handle global key registration through Cloudflare's edge network.

## Prerequisites

- Cloudflare account with your domain configured (`servereye.dev`)
- Node.js 18+ installed locally
- Wrangler CLI installed: `npm install -g wrangler`

## Step 1: Initial Setup

```bash
# Navigate to the worker directory
cd cloudflare/worker

# Install dependencies
npm install

# Login to Cloudflare
wrangler auth login
```

## Step 2: Create D1 Database

```bash
# Create the database
npm run d1:create

# Note the database ID from output
# Example: ✅ Created database 'servereye-registration' with id: 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'
```

## Step 3: Update Configuration

Edit `wrangler.toml`:

1. Replace `REPLACE_WITH_D1_DATABASE_ID` with the actual database ID from Step 2
2. Set a secure `INSTALLER_KEY` (optional but recommended)

```toml
[vars]
INSTALLER_KEY = "your-secure-installer-key-here"
```

## Step 4: Run Database Migration

```bash
# Apply the initial migration
npm run d1:migrate
```

## Step 5: Deploy the Worker

```bash
# Deploy to Cloudflare
npm run deploy
```

After deployment, you'll get a URL like:

```text
✅ Published servereye-registration-worker (1.0.0)
https://servereye-registration-worker.your-subdomain.workers.dev
```

## Step 6: Configure Custom Domain

### Option A: Via Cloudflare Dashboard

1. Go to Cloudflare Dashboard → Workers & Pages
2. Select your worker
3. Click "Triggers" → "Custom Domains"
4. Add: `api.servereye.dev`

### Option B: Via Wrangler CLI

```bash
# Add custom domain
wrangler custom-domains add api.servereye.dev
```

## Step 7: Test the Worker

```bash
# Test without installer key (if not configured)
curl -X POST https://api.servereye.dev/api/register-key \
  -H "Content-Type: application/json" \
  -d '{"secret_key":"srv_test123","hostname":"test-server"}'

# Test with installer key (if configured)
curl -X POST https://api.servereye.dev/api/register-key \
  -H "Content-Type: application/json" \
  -H "x-installer-key: your-secure-installer-key-here" \
  -d '{"secret_key":"srv_test456","hostname":"test-server-2"}'
```

Expected response:
```json
{
  "success": true,
  "data": {
    "secret_key": "srv_test123",
    "status": "stored"
  }
}
```

## Step 8: Set Up Sync to Local PostgreSQL

On your home server (192.168.0.104):

```bash
# Create environment file
sudo mkdir -p /opt/servereye
sudo tee /opt/servereye/.sync-env > /dev/null <<EOF
CLOUDFLARE_ACCOUNT_ID=your-account-id
CLOUDFLARE_API_TOKEN=your-api-token
D1_DATABASE_ID=your-d1-database-id
LOCAL_DB_URL=postgres://servereye_keys:PASSWORD@localhost:5433/PgRegisteredKeys?sslmode=disable
EOF

# Make sync script executable
chmod +x scripts/sync-d1-to-postgres.sh

# Copy to server
scp scripts/sync-d1-to-postgres.sh server@192.168.0.104:/opt/servereye/

# Test sync once
ssh server@192.168.0.104 "cd /opt/servereye && ./sync-d1-to-postgres.sh"

# Set up as daemon (optional)

```bash
ssh server@192.168.0.104 "cd /opt/servereye && nohup ./sync-d1-to-postgres.sh --daemon > sync.log 2>&1 &"
```

## Creating Cloudflare API Token

1. Go to Cloudflare Dashboard → My Profile → API Tokens
2. Create token with permissions:
   - Account: Cloudflare D1:Edit
   - Zone: Zone:Read (if needed)
   - Account Resource: Include your account
   - Zone Resource: Include `servereye.dev`

## Step 9: Update Install Script Usage

The install script now defaults to using the Worker API. To install agents:

```bash
# Default: Uses Cloudflare Worker
curl -fsSL https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install-agent.sh | sudo bash

# With custom installer key
SERVEREYE_INSTALLER_KEY="your-secure-key" curl -fsSL https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install-agent.sh | sudo bash

# Force use of local backend (if needed)

```bash
USE_WORKER_API=false SERVEREYE_API_URL="http://your-backend:8080" curl -fsSL https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install-agent.sh | sudo bash
```

## Monitoring

### View Worker Logs

```bash
# Real-time logs
npm run tail

# Or via Wrangler
wrangler tail
```

### Check D1 Data

```bash
# Query D1 directly
wrangler d1 execute servereye-registration --command "SELECT * FROM generated_keys ORDER BY created_at DESC LIMIT 10"
```

### Check Local PostgreSQL

```bash
docker exec -it PGRegisteredKeysServerEye psql -U servereye_keys -d PgRegisteredKeys -c "SELECT * FROM generated_keys ORDER BY created_at DESC LIMIT 10;"
```
```

## Troubleshooting

### Worker Returns 401 Unauthorized

- Check if `INSTALLER_KEY` is set in `wrangler.toml`
- Ensure the request includes `x-installer-key` header

### Worker Returns 422 Unprocessable Entity

- Ensure `secret_key` is present in the request body

### Sync Script Fails

- Verify Cloudflare API token has correct permissions
- Check that D1 database ID is correct
- Ensure local PostgreSQL is accessible

### Custom Domain Not Working

- Ensure DNS is configured to point to Cloudflare
- Check SSL certificate status in Cloudflare Dashboard
- Verify Worker is deployed and healthy

## Security Notes

1. Always use HTTPS for the Worker endpoint
2. Set a strong `INSTALLER_KEY` to prevent unauthorized registrations
3. Rotate API tokens regularly
4. Monitor Worker logs for suspicious activity
5. Consider implementing rate limiting in the Worker if needed

## Architecture Diagram

```text
[Agent] → HTTPS → [Cloudflare Worker] → D1 Database
                                      ↓
[Home Server] ← Sync ← [Cloudflare API] ← D1 Database
```

The Worker provides a global, highly available endpoint that's not affected by ISP blocking, while your home server remains secure and private.
