# ServerEye

A comprehensive server monitoring system with Telegram bot integration, built with Go.

## Features

- **Real-time CPU temperature monitoring** - Get instant temperature readings from your servers
- **Docker container monitoring** - View status, ports, and details of all Docker containers
- **Telegram bot interface** - Control and monitor your servers through Telegram
- **Multi-server support** - Monitor multiple servers from a single bot
- **Redis Pub/Sub communication** - Reliable message passing between components
- **PostgreSQL storage** - Persistent storage for users and server configurations
- **Systemd integration** - Automatic agent startup and management
- **Docker deployment** - Easy deployment with Docker Compose

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Telegram Bot  │    │      Redis      │    │   ServerEye     │
│   (Docker)      │◄──►│   (Pub/Sub)     │◄──►│    Agent        │
│                 │    │                 │    │   (Systemd)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   PostgreSQL    │    │   Message       │    │   Docker API    │
│   (Database)    │    │   Protocol      │    │   CPU Sensors   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+
- Linux server with systemd

### 1. Clone the repository

```bash
git clone https://github.com/yourusername/ServerEye.git
cd ServerEye
```

### 2. Configure environment

```bash
cp deployments/.env.example deployments/.env
# Edit deployments/.env with your Telegram bot token and database password
```

### 3. Deploy with Docker Compose

```bash
cd deployments
docker-compose up -d
```

### 4. Install agent on monitored servers

```bash
# Build agent
make build-agent

# Configure agent
mkdir -p ~/.servereye
cat > ~/.servereye/config.yaml << EOF
server:
  name: "My Server"
  description: "Production server"
  secret_key: "srv_your_secret_key_here"

redis:
  address: "your-redis-host:6379"
  password: ""
  db: 0

metrics:
  cpu_temperature: true
  interval: 30s

logging:
  level: info
  file: ./agent.log
EOF

# Install as systemd service
sudo make install-agent
```

### 5. Connect to Telegram bot

1. Find your bot: `@YourServerEyeBot`
2. Send `/start`
3. Send your server secret key: `srv_your_secret_key_here`
4. Use commands:
   - `/temp` - Get CPU temperature
   - `/containers` - List Docker containers
   - `/status` - Server status
   - `/help` - Show all commands

## Commands

| Command | Description |
|---------|-------------|
| `/start` | Initialize bot and register user |
| `/temp` | Get current CPU temperature |
| `/containers` | List all Docker containers with status |
| `/status` | Get server status and uptime |
| `/servers` | List your connected servers |
| `/help` | Show available commands |

## Development

### Project Structure

```
ServerEye/
├── cmd/                    # Application entry points
│   ├── agent/             # Agent main
│   └── bot/               # Bot main
├── internal/              # Private application code
│   ├── agent/             # Agent implementation
│   ├── bot/               # Bot implementation
│   └── config/            # Configuration management
├── pkg/                   # Public library code
│   ├── docker/            # Docker client
│   ├── metrics/           # System metrics collection
│   ├── protocol/          # Message protocol
│   └── redis/             # Redis client
├── deployments/           # Docker and deployment configs
├── scripts/               # Deployment and utility scripts
└── .github/workflows/     # CI/CD pipelines
```

### Build and Test

```bash
# Install dependencies
make deps

# Run tests
make test

# Build binaries
make build

# Run linting
make lint

# Build Docker images
make docker-build
```

### Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'feat: add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

## Configuration

### Bot Configuration (.env)

```env
TELEGRAM_TOKEN=your_bot_token_here
DB_PASSWORD=your_secure_password
REDIS_URL=redis://redis:6379
DATABASE_URL=postgresql://servereye:password@postgres:5432/servereye
ENVIRONMENT=production
```

### Agent Configuration (config.yaml)

```yaml
server:
  name: "Production Server"
  description: "Main application server"
  secret_key: "srv_unique_secret_key"

redis:
  address: "redis-host:6379"
  password: ""
  db: 0

metrics:
  cpu_temperature: true
  interval: 30s

logging:
  level: info
  file: ./agent.log
```

## Deployment

### Production Deployment

1. **Server Setup**: Deploy bot, Redis, and PostgreSQL using Docker Compose
2. **Agent Installation**: Install agents on monitored servers using systemd
3. **Security**: Use secure passwords, proper firewall rules, and SSL/TLS
4. **Monitoring**: Set up log aggregation and health checks

### Docker Compose Services

- **servereye-bot**: Telegram bot service
- **redis**: Message broker for Pub/Sub communication
- **postgres**: Database for user and server data

## Monitoring and Logging

- **Agent logs**: `journalctl --user -u servereye-agent.service -f`
- **Bot logs**: `docker-compose logs -f servereye-bot`
- **Redis logs**: `docker-compose logs -f redis`
- **Database logs**: `docker-compose logs -f postgres`

## Security

- All communication uses Redis Pub/Sub with unique server keys
- Database connections use SSL/TLS in production
- Sensitive configuration stored in environment variables
- Agent runs as unprivileged user with systemd

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- Create an [Issue](https://github.com/yourusername/ServerEye/issues) for bug reports
- Start a [Discussion](https://github.com/yourusername/ServerEye/discussions) for questions
- Check [Documentation](docs/) for detailed guides

## Roadmap

- [ ] Web dashboard interface
- [ ] More system metrics (RAM, disk, network)
- [ ] Alert thresholds and notifications
- [ ] Multi-language support
- [ ] Plugin system for custom metrics
- [ ] Grafana integration
