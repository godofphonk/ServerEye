# 🔒 Security Considerations

This document outlines security considerations, best practices, and known limitations of ServerEye.

## Table of Contents

- [Binary Integrity](#binary-integrity)
- [Docker Access](#docker-access)
- [Secret Management](#secret-management)
- [Network Security](#network-security)
- [Authentication & Authorization](#authentication--authorization)
- [Data Protection](#data-protection)
- [Security Best Practices](#security-best-practices)
- [Known Limitations](#known-limitations)
- [Reporting Vulnerabilities](#reporting-vulnerabilities)

## Binary Integrity

### SHA256 Checksum Verification

All release binaries are protected with SHA256 checksums to ensure integrity.

**What we do:**
- ✅ Generate checksums for all binaries during build
- ✅ Store checksums in version-controlled `SHA256SUMS` file
- ✅ Install script automatically verifies checksums before execution
- ✅ Public access to checksums for manual verification

**Manual Verification:**

```bash
# Download binary and checksum file
wget https://raw.githubusercontent.com/godofphonk/ServerEye/master/downloads/servereye-agent-linux
wget https://raw.githubusercontent.com/godofphonk/ServerEye/master/downloads/SHA256SUMS

# Verify checksum
sha256sum -c SHA256SUMS --ignore-missing

# Should output:
# servereye-agent-linux: OK
```

**Protection Against:**
- ✅ Man-in-the-middle (MITM) attacks
- ✅ Corrupted downloads
- ✅ Tampered binaries
- ✅ Version mismatches

## Docker Access

### The Docker Security Challenge

The agent requires membership in the `docker` group to manage containers.

> ⚠️ **Critical Security Note:** Users in the docker group have **root-equivalent privileges** on the host system. This is a fundamental Docker limitation, not specific to ServerEye.

### What This Means

**Capabilities granted:**
- ✅ Start/stop/restart existing containers
- ✅ Inspect container configurations
- ✅ View container logs

**Capabilities NOT granted:**
- ❌ Create new containers
- ❌ Modify container images
- ❌ Access Docker build system
- ❌ Change Docker daemon settings

**However, docker group membership allows:**
- ⚠️ Mount host filesystem into container
- ⚠️ Run privileged containers
- ⚠️ Escape container isolation

### Example Attack Vector

A malicious actor with docker group access could:

```bash
# Mount root filesystem and gain root access
docker run -v /:/host -it alpine chroot /host /bin/bash

# Or run a privileged container
docker run --privileged -it alpine
```

### Mitigation Strategies

#### 1. Rootless Docker (Recommended for Production)

Run Docker daemon as non-root user:

```bash
# Install rootless Docker
curl -fsSL https://get.docker.com/rootless | sh

# Configure agent to use rootless Docker
export DOCKER_HOST=unix:///run/user/1000/docker.sock
```

**Benefits:**
- ✅ No root-equivalent privileges
- ✅ Container isolation improved
- ✅ Reduced attack surface

**Trade-offs:**
- ❌ Limited port binding (<1024)
- ❌ Some volume mount restrictions
- ❌ Performance overhead

#### 2. Docker Socket Proxy

Use a proxy to restrict Docker API access:

```bash
# Run socket proxy
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -p 2375:2375 \
  tecnativa/docker-socket-proxy

# Configure agent
api:
  docker_host: "tcp://localhost:2375"
```

**Benefits:**
- ✅ Fine-grained access control
- ✅ Audit logging
- ✅ API filtering

#### 3. AppArmor/SELinux Profiles

Restrict agent capabilities:

```bash
# AppArmor profile for agent
sudo aa-enforce /etc/apparmor.d/servereye-agent

# SELinux context
sudo chcon -t docker_t /opt/servereye/servereye-agent
```

#### 4. Principle of Least Privilege

Only give docker access to trusted servers:

```bash
# Don't add agent to docker group by default
# Use sudo with specific commands instead
sudo visudo
# Add: servereye ALL=(ALL) NOPASSWD: /usr/bin/docker ps, /usr/bin/docker start *, /usr/bin/docker stop *
```

### Production Recommendations

**For production deployments:**

1. ✅ Use rootless Docker when possible
2. ✅ Implement Docker socket proxy with access controls
3. ✅ Run agent in restricted environment (container, VM)
4. ✅ Regular security audits
5. ✅ Monitor agent activities
6. ✅ Rotate server keys periodically

**Risk Assessment:**

| Deployment Type | Risk Level | Recommendation |
|----------------|-----------|----------------|
| Personal server | Low | Standard docker group |
| Development env | Low-Medium | Standard docker group |
| Production (internal) | Medium | Docker socket proxy |
| Production (public) | High | Rootless Docker + proxy |
| Multi-tenant | Critical | Isolated environments |

## Secret Management

### Server Keys

**Generation:**
- ✅ Generated locally on each server using `openssl rand -hex 16`
- ✅ 32 characters hexadecimal (128-bit entropy)
- ✅ Prefixed with `srv_` for identification
- ✅ Never transmitted during installation
- ✅ Cryptographically secure random number generator

**Security Properties:**
- **Key Space:** 2^128 = 3.4×10^38 possible keys
- **Brute Force:** Computationally infeasible
- **Collision Probability:** Negligible (< 10^-36)
- **Algorithm:** OpenSSL CSPRNG (Cryptographically Secure Pseudo-Random Number Generator)

**Key Format:**
```
srv_[32 hex characters]
Example: srv_a1b2c3d4e5f6789012345678901234567890abcd
```

**Storage:**
```bash
# Config file with restricted permissions
/etc/servereye/config.yaml  # chmod 640
                            # owner: root:servereye
```

**Lifetime:**
- ⚠️ **No expiration** - keys are valid indefinitely
- ✅ Manually rotatable (generate new key, update config)
- 📝 Recommendation: Rotate keys every 6-12 months for production

**Rotation:**
```bash
# Generate new key
openssl rand -hex 16 | sed 's/^/srv_/'

# Update config
sudo vi /etc/servereye/config.yaml

# Update in bot
/add srv_new_key MyServer

# Restart agent
sudo systemctl restart servereye-agent
```

### Telegram Bot Token

**How to obtain:**
1. Open Telegram and search for `@BotFather`
2. Send `/newbot` command
3. Follow the instructions to create your bot
4. Save the token securely - **you'll only see it once**

**Storage (Development):**
```bash
# .env file 
TELEGRAM_TOKEN=1234567890:ABCdefGHIjklMNOpqrsTUVwxyz

# Load from .env
export $(cat .env | xargs)

# Or export directly
export TELEGRAM_TOKEN="your_token_here"
```

**Storage (Production):**
```bash
# Option 1: Docker Compose with .env
# deployments/.env (gitignored)
TELEGRAM_TOKEN=your_token_here

# Option 2: Docker Secrets (recommended)
echo "your_token_here" | docker secret create telegram_token -
docker service update --secret-add telegram_token mybot

# Option 3: Kubernetes Secrets
kubectl create secret generic telegram-token \
  --from-literal=token='your_token_here'
```

**Code Implementation:**
```go
// ✅ CORRECT - Load from environment
token := os.Getenv("TELEGRAM_TOKEN")
if token == "" {
    return errors.New("TELEGRAM_TOKEN not set")
}

// ❌ WRONG - Hardcoded token
token := "123456:ABC-DEF..."  // NEVER do this!
```

**Security Checklist:**
- [ ] Token loaded from environment variables
- [ ] No token in source code
- [ ] No token in config files
- [ ] `.env` file in `.gitignore`
- [ ] Token not logged in application logs
- [ ] Token regenerated if leaked

**If Token is Leaked:**
1. Open `@BotFather` in Telegram
2. Send `/mybots` → select your bot
3. Go to **Bot Settings** → **Regenerate Token**
4. Update your environment variables
5. Restart your application
6. Investigate how the leak occurred

### Database Credentials

**PostgreSQL:**
```bash
# Use environment variables
export DATABASE_URL="postgresql://user:pass@host:port/db?sslmode=require"

# Or Docker secrets
docker secret create postgres_password /run/secrets/postgres_password
```

**Redis:**
```bash
# Enable authentication
requirepass your_strong_password_here

# Use environment variables
export REDIS_URL="redis://:password@localhost:6379"
```

## Network Security

### HTTPS Everywhere

**All external communication uses HTTPS:**
- ✅ Telegram Bot API (mandatory)
- ✅ HTTP API endpoints via Cloudflare
- ✅ Webhook callbacks
- ✅ Redis HTTP proxy

### Cloudflare Tunnel

**Benefits:**
- ✅ Server IP address hidden
- ✅ DDoS protection
- ✅ TLS termination
- ✅ Rate limiting
- ✅ Web Application Firewall (WAF)

**Setup:**
```bash
# Install cloudflared
wget https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64
sudo mv cloudflared-linux-amd64 /usr/local/bin/cloudflared
sudo chmod +x /usr/local/bin/cloudflared

# Create tunnel
cloudflared tunnel create servereye

# Route tunnel
cloudflared tunnel route dns servereye api.servereye.dev

# Run tunnel
cloudflared tunnel run servereye
```

### Internal Network Isolation

**Redis and PostgreSQL:**
- ✅ Bind to localhost or internal network only
- ✅ No public exposure
- ✅ Firewall rules to restrict access

```bash
# PostgreSQL - listen on internal network only
listen_addresses = '127.0.0.1'

# Redis - bind to localhost
bind 127.0.0.1

# Firewall rules
sudo ufw allow from 192.168.0.0/24 to any port 5432
sudo ufw allow from 192.168.0.0/24 to any port 6379
```

## Authentication & Authorization

### Multi-Level Security

**Layer 1: Telegram Authentication**
- User must have valid Telegram account
- Bot verifies user_id from Telegram API

**Layer 2: Server Key Authentication**
- Each server has unique secret key
- Commands validated against registered servers

**Layer 3: User-Server Ownership**
- PostgreSQL tracks which users own which servers
- Users can only control their own servers

### Authorization Flow

```
User → /temp → Bot
           ↓
    Check: Is user registered?
           ↓
    Check: Does user own server?
           ↓
    Check: Is server key valid?
           ↓
    Execute: Get temperature
```

## Data Protection

### Data at Rest

**Database:**
- ✅ Use encrypted filesystem
- ✅ Regular backups
- ✅ Backup encryption

**Logs:**
- ✅ Redact sensitive information
- ✅ Log rotation
- ✅ Access controls

### Data in Transit

- ✅ TLS 1.3 for all external communication
- ✅ Certificate pinning for critical endpoints
- ✅ No sensitive data in URLs

### Personally Identifiable Information (PII)

**What we store:**
- Telegram user_id (required for bot operation)
- Telegram username (optional, for display)
- Server hostnames (user-provided)

**What we log:**
- Command execution (without arguments)
- Error messages (sanitized)
- System metrics (anonymous)

## Security Best Practices

### For Server Owners

1. ✅ Review installation script before running
2. ✅ Use strong, unique server keys
3. ✅ Keep agent updated
4. ✅ Monitor agent logs
5. ✅ Restrict physical server access
6. ✅ Use firewall rules
7. ✅ Enable audit logging

### For Bot Operators

1. ✅ Keep bot token secret
2. ✅ Use environment variables for secrets
3. ✅ Enable database encryption
4. ✅ Regular security audits
5. ✅ Monitor suspicious activities
6. ✅ Rate limit bot commands
7. ✅ Implement user blocking mechanism

### For Developers

1. ✅ Follow secure coding practices
2. ✅ Regular dependency updates
3. ✅ Static analysis (golangci-lint)
4. ✅ Input validation and sanitization
5. ✅ Error handling without information leaks
6. ✅ Security-focused code reviews
7. ✅ Penetration testing

## Known Limitations

### Docker Group Privileges

- ⚠️ Docker group membership grants root-equivalent access
- **Mitigation:** Use rootless Docker or socket proxy

### Redis Pub/Sub

- ⚠️ No message encryption (uses internal network)
- **Mitigation:** Ensure Redis on internal network only

### No End-to-End Encryption

- ⚠️ Messages decrypted at bot/agent layer
- **Mitigation:** TLS for transport, trusted infrastructure

### Single-Factor Authentication

- ⚠️ Only Telegram authentication
- **Mitigation:** Strong Telegram account security (2FA)

### Command Injection Risk

- ⚠️ Docker commands constructed from user input
- **Mitigation:** Whitelist validation, no shell execution

## Agent Updates & Rollback

### Safe Update Process

**Option 1: Manual Update (Recommended)**

```bash
# 1. Download new binary
cd /tmp
wget https://raw.githubusercontent.com/godofphonk/ServerEye/master/downloads/servereye-agent-linux

# 2. Verify checksum
wget https://raw.githubusercontent.com/godofphonk/ServerEye/master/downloads/SHA256SUMS
sha256sum -c SHA256SUMS --ignore-missing

# 3. Backup current version
sudo cp /opt/servereye/servereye-agent /opt/servereye/servereye-agent.backup

# 4. Stop agent
sudo systemctl stop servereye-agent

# 5. Replace binary
sudo cp servereye-agent-linux /opt/servereye/servereye-agent
sudo chmod +x /opt/servereye/servereye-agent

# 6. Test new version
/opt/servereye/servereye-agent -version

# 7. Start agent
sudo systemctl start servereye-agent

# 8. Verify it's working
sudo systemctl status servereye-agent
sudo journalctl -u servereye-agent -n 20
```

**Option 2: Re-run Install Script**

```bash
# This will overwrite the agent but keep your config
wget -qO- https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install-agent.sh | sudo bash
```

### Rollback Process

**If update fails:**

```bash
# 1. Stop broken version
sudo systemctl stop servereye-agent

# 2. Restore backup
sudo cp /opt/servereye/servereye-agent.backup /opt/servereye/servereye-agent

# 3. Start old version
sudo systemctl start servereye-agent

# 4. Verify rollback
sudo systemctl status servereye-agent
```

### Multi-Server Update Strategy

**For managing multiple servers:**

```bash
# Create update script
cat > update-agent.sh << 'EOF'
#!/bin/bash
SERVERS="server1 server2 server3"
for server in $SERVERS; do
    echo "Updating $server..."
    ssh $server 'bash -s' < /path/to/local-update-script.sh
    if [ $? -eq 0 ]; then
        echo "✅ $server updated successfully"
    else
        echo "❌ $server update failed"
    fi
done
EOF
chmod +x update-agent.sh
```

## Reporting Vulnerabilities

If you discover a security vulnerability:

1. **DO NOT** create a public GitHub issue
2. Email: [cacaca86@mail.ru]
3. Include:
   - Description of vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (optional)

**See also:**
- [Architecture](ARCHITECTURE.md)
- [Development Guide](DEVELOPMENT.md)
- [Monitoring & Observability](MONITORING.md)
