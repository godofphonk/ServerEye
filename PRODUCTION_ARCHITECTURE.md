# ServerEye Production Architecture

## 🏗️ Overview
ServerEye production deployment uses a hybrid architecture combining Docker containers with host networking for optimal performance and reliability.

## 🌐 Network Configuration

### Bot Service
- **Container**: `servereye-bot` 
- **Network Mode**: `host` (direct host access)
- **Database URL**: `postgresql://servereye:xxx@localhost:5433/servereye`
- **HTTP Port**: `8090` (host network)
- **Advantage**: Bypasses docker networking overhead, direct host access

### Database Services
- **Primary Database**: Host service on port `5433`
- **Backup Database**: Docker container on port `5432` (mapped `5432:5432`)
- **Connection**: Bot connects to host postgres on `5433`
- **Redundancy**: Docker postgres available for backup/development

### External Access
- **Global API**: `https://api.servereye.dev` via Cloudflare Tunnel
- **Local API**: `http://localhost` via Nginx reverse proxy
- **Tunnel Service**: `cloudflared` systemd service with auto-restart
- **Nginx Proxy**: Local fallback on port `80`

## 🔌 Port Mapping

| Port | Service | Purpose | Network |
|------|---------|---------|---------|
| 5433 | Host Postgres | Primary database | Host |
| 5432 | Docker Postgres | Backup/development | Docker (host:container) |
| 8090 | Bot HTTP API | Agent registration | Host |
| 80 | Nginx Proxy | Local API access | Host |
| 443 | Cloudflare Tunnel | Global API access | Internet |

## 🚀 Data Flow Architecture

### Global Registration Flow
```
Agent → https://api.servereye.dev
     ↓
Cloudflare Tunnel (secure outbound)
     ↓
Bot Container (host:8090)
     ↓
Host Postgres (localhost:5433)
     ↓
Database Record with Metadata
```

### Local Registration Flow
```
Local Client → http://localhost
            ↓
Nginx Reverse Proxy (:80)
            ↓
Bot Container (host:8090)
            ↓
Host Postgres (localhost:5433)
            ↓
Database Record with Metadata
```

## 📊 Service Status

### ✅ Operational Services
- **Bot API**: Fully functional with agent registration
- **PostgreSQL**: Primary database operational on port 5433
- **Cloudflare Tunnel**: Active with systemd auto-restart
- **Nginx Proxy**: Configured and functional
- **Database Records**: Complete metadata capture working

### 🔄 Redundancy Features
- **Dual API Access**: Global (Tunnel) + Local (Nginx)
- **Database Backup**: Docker postgres available
- **Auto-restart**: Systemd-managed tunnel service
- **Health Monitoring**: Docker health checks enabled

## 🛠️ Configuration Details

### Docker Compose Production
```yaml
services:
  servereye-bot:
    network_mode: host  # Direct host access
    environment:
      DATABASE_URL: postgresql://servereye:xxx@localhost:5433/servereye
      HTTP_PORT: 8090
  
  postgres:
    ports:
      - "5432:5432"  # Backup database access
```

### Cloudflare Tunnel
```yaml
tunnel: a9dce285-f99c-4774-bffa-1d084f724db1
credentials-file: /etc/cloudflared/credentials.json
ingress:
  - hostname: api.servereye.dev
    service: http://localhost:8090
```

### Systemd Service
- **Service**: `cloudflared`
- **Status**: `active (running)`
- **Auto-restart**: Enabled
- **Config**: `/etc/cloudflared/config.yml`

## 🔍 Operational Notes

### Proxy Configuration
- **Issue**: Server has proxy on `127.0.0.1:7890`
- **Workaround**: Use `curl --noproxy '*'` or `unset http_proxy https_proxy`
- **Impact**: Affects curl/wget operations, not agent registration

### Database Connection
- **Primary**: Host postgres on port 5433 (recommended)
- **Backup**: Docker postgres on port 5432 (available)
- **Choice**: Bot uses host postgres for performance

### Monitoring
- **Tunnel Status**: `systemctl status cloudflared`
- **Bot Logs**: `docker logs servereye-bot`
- **Database**: Direct connection on port 5433

## 📈 Performance Characteristics

### Advantages
- **Host Networking**: Eliminates docker network overhead
- **Direct Database Access**: Optimal performance
- **Global Redundancy**: Cloudflare Tunnel + Nginx
- **Auto-recovery**: Systemd service management

### Considerations
- **Port Management**: Host networking requires port coordination
- **Security**: Direct host access needs proper firewall rules
- **Backup Strategy**: Docker postgres provides fallback option

---

*Documented: 2025-11-26*
*Architecture: Production Hybrid (Docker + Host Networking)*
*Status: Fully Operational*
