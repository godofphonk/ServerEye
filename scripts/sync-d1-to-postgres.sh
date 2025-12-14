#!/bin/bash
# Sync ServerEye registration keys from Cloudflare D1 to local PostgreSQL
# Run this on your home server: server@192.168.0.104

set -e

# Load environment variables from file if it exists
ENV_FILE="/home/gospodin/servereye/.sync-env"
if [ -f "$ENV_FILE" ]; then
    source "$ENV_FILE"
fi

# Configuration
CLOUDFLARE_ACCOUNT_ID="${CLOUDFLARE_ACCOUNT_ID:-}"
CLOUDFLARE_API_TOKEN="${CLOUDFLARE_API_TOKEN:-}"
D1_DATABASE_ID="${D1_DATABASE_ID:-}"
LOCAL_DB_URL="${LOCAL_DB_URL:-${POSTGRES_URL:-postgres://servereye_keys:PASSWORD@localhost:5433/PgRegisteredKeys?sslmode=disable}}"
SYNC_INTERVAL="${SYNC_INTERVAL:-300}"  # 5 minutes

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check required environment variables
check_env() {
    if [ -z "$CLOUDFLARE_ACCOUNT_ID" ] || [ -z "$CLOUDFLARE_API_TOKEN" ] || [ -z "$D1_DATABASE_ID" ]; then
        echo -e "${RED}Missing required environment variables:${NC}"
        echo "  - CLOUDFLARE_ACCOUNT_ID"
        echo "  - CLOUDFLARE_API_TOKEN"
        echo "  - D1_DATABASE_ID"
        echo ""
        echo "Set them in ~/servereye/.sync-env or export them before running"
        exit 1
    fi
}

check_env

# Create last sync tracking file
LAST_SYNC_FILE="$HOME/servereye/.last-sync"
mkdir -p "$HOME/servereye"

# Get last sync timestamp (default to 24 hours ago)
get_last_sync() {
    if [ -f "$LAST_SYNC_FILE" ]; then
        cat "$LAST_SYNC_FILE"
    else
        # Default to 24 hours ago (format matching D1 storage: YYYY-MM-DD HH:MM:SS)
        date -u -d '24 hours ago' '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -u -v-24H '+%Y-%m-%d %H:%M:%S' 2>/dev/null
    fi
}

# Update last sync timestamp
update_last_sync() {
    date -u '+%Y-%m-%d %H:%M:%S' > "$LAST_SYNC_FILE"
}

# Fetch records from D1
fetch_from_d1() {
    local since="$1"
    echo -e "${YELLOW}Fetching records from D1 since $since${NC}" >&2
    
    # Use Cloudflare API to query D1
    local response
    response=$(curl -s -X POST \
        "https://api.cloudflare.com/client/v4/accounts/$CLOUDFLARE_ACCOUNT_ID/d1/database/$D1_DATABASE_ID/query" \
        -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"sql\": \"SELECT secret_key, agent_version, os_info, hostname, status, created_at, updated_at FROM generated_keys WHERE updated_at > ? ORDER BY updated_at ASC\",
            \"params\": [\"$since\"]
        }")
    
    # Check for API errors
    if echo "$response" | grep -q '"success":false'; then
        echo -e "${RED}Failed to fetch from D1:${NC}"
        echo "$response" | jq -r '.errors[0].message' 2>/dev/null || echo "$response"
        return 1
    fi
    
    # Extract records
    echo "$response" | jq -r '.result[0].results[] | @base64' 2>/dev/null || echo ""
}

# Insert records into local PostgreSQL
insert_into_pg() {
    local records="$1"
    
    if [ -z "$records" ]; then
        echo -e "${GREEN}No new records to sync${NC}"
        return 0
    fi
    
    echo -e "${YELLOW}Syncing $(echo "$records" | wc -l) records to PostgreSQL${NC}"
    
    # Process each record
    echo "$records" | while IFS= read -r record; do
        if [ -n "$record" ]; then
            # Decode base64 and extract fields
            local decoded
            decoded=$(echo "$record" | base64 -d 2>/dev/null || echo "$record")
            
            local secret_key
            local agent_version
            local os_info
            local hostname
            local status
            local created_at
            local updated_at
            
            secret_key=$(echo "$decoded" | jq -r '.secret_key // empty' 2>/dev/null || echo "")
            agent_version=$(echo "$decoded" | jq -r '.agent_version // "unknown"' 2>/dev/null || echo "unknown")
            os_info=$(echo "$decoded" | jq -r '.os_info // "unknown"' 2>/dev/null || echo "unknown")
            hostname=$(echo "$decoded" | jq -r '.hostname // "unknown"' 2>/dev/null || echo "unknown")
            status=$(echo "$decoded" | jq -r '.status // "generated"' 2>/dev/null || echo "generated")
            created_at=$(echo "$decoded" | jq -r '.created_at // empty' 2>/dev/null || echo "")
            updated_at=$(echo "$decoded" | jq -r '.updated_at // empty' 2>/dev/null || echo "")
            
            if [ -n "$secret_key" ]; then
                # Insert into PostgreSQL using parameterized query via variables
                psql "$LOCAL_DB_URL" \
                    -v secret_key="$secret_key" \
                    -v agent_version="$agent_version" \
                    -v os_info="$os_info" \
                    -v hostname="$hostname" \
                    -v status="$status" \
                    -v created_at="${created_at:-}" \
                    -v updated_at="${updated_at:-}" <<'SQL'
BEGIN;
    INSERT INTO generated_keys (secret_key, agent_version, os_info, hostname, status, created_at, updated_at)
    VALUES (
        :'secret_key',
        :'agent_version',
        :'os_info',
        :'hostname',
        :'status',
        CASE WHEN :'created_at' = '' THEN NOW() ELSE :'created_at'::timestamptz END,
        CASE WHEN :'updated_at' = '' THEN NOW() ELSE :'updated_at'::timestamptz END
    )
    ON CONFLICT (secret_key) DO UPDATE SET
        agent_version = EXCLUDED.agent_version,
        os_info = EXCLUDED.os_info,
        hostname = EXCLUDED.hostname,
        status = EXCLUDED.status,
        updated_at = EXCLUDED.updated_at;
COMMIT;
SQL
            fi
        fi
    done
}

# Main sync function
sync() {
    echo -e "${GREEN}=== Starting D1 to PostgreSQL sync ===${NC}"
    
    local last_sync
    last_sync=$(get_last_sync)
    
    local records
    records=$(fetch_from_d1 "$last_sync")
    
    if [ $? -eq 0 ]; then
        insert_into_pg "$records"
        update_last_sync
        echo -e "${GREEN}=== Sync completed ===${NC}"
    else
        echo -e "${RED}=== Sync failed ===${NC}"
        return 1
    fi
}

# Run continuously if requested
if [ "$1" = "--daemon" ]; then
    echo -e "${GREEN}Starting sync daemon (interval: ${SYNC_INTERVAL}s)${NC}"
    while true; do
        sync
        sleep "$SYNC_INTERVAL"
    done
else
    # Run once
    sync
fi
