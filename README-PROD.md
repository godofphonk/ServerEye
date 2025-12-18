# ServerEye Production Deployment Guide

## Overview
ServerEye is a monitoring system that supports both local (Kafka) and worldwide (HTTP API) deployment modes. This guide covers production deployment.

## Prerequisites
- Docker 20.10+
- Docker Compose 2.0+
- Domain name with DNS A record pointing to your server
- SSL certificate (Let's Encrypt recommended)

## Quick Start

### 1. Clone and Configure
```bash
git clone https://github.com/godofphonk/ServerEye.git
cd ServerEye
cp .env.prod .env.local
# Edit .env.local with your configuration
```

### 2. Deploy
```bash
chmod +x scripts/deploy-prod.sh
./scripts/deploy-prod.sh
```

## Configuration

### Environment Variables (.env.prod)
- `API_KEY`: Authentication key for API access
- `DOMAIN`: Your domain name (default: servereye.dev)
- `LETSENCRYPT_EMAIL`: Email for SSL certificate
- `DATABASE_URL`: PostgreSQL connection string (optional)
- `KAFKA_BROKERS`: Kafka broker addresses
- `LOG_LEVEL`: Logging level (info/warn/debug)

### SSL Certificates
The deployment script automatically generates Let's Encrypt certificates. Ensure:
- Port 80 is open for HTTP validation
- Domain DNS points to your server

## Architecture

### Production Components
1. **Backend API** (Go)
   - HTTP/2 support
   - Rate limiting
   - Health checks
   - Metrics endpoint

2. **Nginx** (Reverse Proxy)
   - SSL termination
   - Rate limiting
   - Security headers
   - HTTP/2 support

3. **Kafka** (Optional for local mode)
   - Message broker
   - Topic management
   - Consumer groups

4. **Zookeeper** (Kafka dependency)
   - Kafka coordination

## API Endpoints

### Authentication
All API endpoints require `X-API-Key` header.

### Endpoints
- `GET /health` - Health check (no auth required)
- `POST /v1/commands` - Send command to agent
- `GET /v1/commands/{server_key}` - Get commands for agent
- `POST /v1/commands/{server_key}/response` - Send command response

## Deployment Modes

### Worldwide Mode (Recommended)
- Uses HTTP API for agent communication
- Works through Cloudflare tunnel
- No public Kafka required
- Agents poll API for commands

### Local Mode
- Uses Kafka for message delivery
- Requires Kafka broker access
- Higher performance for local networks

## Monitoring

### Health Checks
- Backend: `GET /health`
- Nginx: `GET /health`
- Kafka: Built-in health checks

### Logs
```bash
# View logs
docker-compose -f docker-compose.prod.yml logs -f

# Specific service
docker-compose -f docker-compose.prod.yml logs -f backend
```

## Security

### SSL/TLS
- TLS 1.2 and 1.3 only
- Strong cipher suites
- HSTS headers
- Certificate auto-renewal

### API Security
- API key authentication
- Rate limiting (10 req/s for API, 30 req/s for health)
- CORS protection
- Security headers

## Troubleshooting

### Common Issues

1. **SSL Certificate Error**
   ```bash
   # Regenerate certificate
   docker-compose -f docker-compose.prod.yml run --rm certbot certonly --webroot --webroot-path=/var/www/certbot --email admin@servereye.dev --agree-tos --no-eff-email -d servereye.dev
   ```

2. **Backend Not Starting**
   ```bash
   # Check logs
   docker-compose -f docker-compose.prod.yml logs backend
   
   # Verify environment variables
   docker-compose -f docker-compose.prod.yml exec backend env | grep API_KEY
   ```

3. **Kafka Connection Issues**
   ```bash
   # Check Kafka status
   docker-compose -f docker-compose.prod.yml exec kafka kafka-topics --bootstrap-server localhost:9092 --list
   ```

### Performance Tuning

#### Nginx
- Adjust `worker_processes` (default: auto)
- Modify `client_max_body_size` for large metrics
- Tune rate limits in nginx/conf.d/servereye.conf

#### Backend
- Set `GOMAXPROCS` environment variable
- Adjust timeouts in backend configuration
- Enable metrics collection

## Maintenance

### Updates
```bash
# Pull latest images
docker-compose -f docker-compose.prod.yml pull

# Restart services
docker-compose -f docker-compose.prod.yml up -d
```

### Backups
```bash
# Backup Kafka data
docker run --rm -v servereye_kafka_data:/data -v $(pwd):/backup alpine tar czf /backup/kafka-backup.tar.gz -C /data .

# Backup database (if used)
docker exec servereye-postgres-prod pg_dump -U postgres servereye > backup.sql
```

## Support

For issues and questions:
1. Check logs: `docker-compose logs`
2. Verify configuration in .env.prod
3. Check health endpoints
4. Review this guide

## License
MIT License - see LICENSE file for details.
