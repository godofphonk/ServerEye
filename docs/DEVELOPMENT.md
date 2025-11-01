# 🏗️ Development Guide

## Project Structure

```
ServerEye/
├── cmd/
│   ├── bot/                    # Bot entry point
│   │   └── main.go
│   └── agent/                  # Agent entry point
│       └── main.go
├── internal/
│   ├── bot/                    # Bot implementation
│   │   ├── bot.go             # Core bot logic (117 lines)
│   │   ├── handlers.go        # Command routing (123 lines)
│   │   ├── monitoring.go      # Monitoring commands (310 lines)
│   │   ├── containers.go      # Docker management (287 lines)
│   │   ├── callbacks.go       # Inline buttons (229 lines)
│   │   ├── servers.go         # Server management (130 lines)
│   │   ├── api.go             # Agent API (366 lines)
│   │   ├── utils.go           # Utilities (67 lines)
│   │   ├── interfaces.go      # Interfaces for DI
│   │   ├── errors.go          # Error definitions
│   │   ├── metrics.go         # Metrics collection
│   │   ├── validator.go       # Input validation
│   │   ├── logger.go          # Structured logging
│   │   ├── adapters.go        # Interface adapters
│   │   └── http_server.go     # HTTP API server
│   ├── agent/                  # Agent implementation
│   │   ├── agent.go           # Core agent logic
│   │   └── handlers.go        # Command handlers
│   └── config/                 # Configuration
│       └── config.go          # Config loading
├── pkg/
│   ├── protocol/              # Message protocol
│   │   └── protocol.go
│   ├── redis/                 # Redis clients
│   │   ├── client.go          # Direct Redis client
│   │   └── http_client.go     # HTTP proxy client
│   ├── docker/                # Docker integration
│   │   ├── client.go
│   │   ├── management.go
│   │   ├── health.go
│   │   └── client_test.go
│   └── metrics/               # System metrics
│       ├── cpu.go
│       ├── memory.go
│       ├── disk.go
│       └── system.go
├── deployments/               # Deployment configs
│   ├── Dockerfile.bot
│   ├── Dockerfile.bot-simple
│   ├── Dockerfile.agent
│   └── docker-compose.yml
├── scripts/                   # Scripts
│   ├── install-agent.sh      # Agent installer
│   └── servereye-agent.service # Systemd service
├── downloads/                 # Release binaries
│   ├── servereye-bot-linux
│   ├── servereye-agent-linux
│   └── SHA256SUMS
└── docs/                      # Documentation
    ├── ARCHITECTURE.md
    ├── DEVELOPMENT.md
    ├── SECURITY.md
    └── MONITORING.md
```

## Prerequisites

### Required Software
- **Go 1.23+** - Programming language
- **Docker & Docker Compose** - Containerization
- **PostgreSQL 14+** - Database
- **Redis 7+** - Message broker
- **Git** - Version control

### Development Tools
- **golangci-lint** - Linting
- **air** - Hot reload (optional)
- **make** - Build automation

## Getting Started

### 1. Clone Repository

```bash
git clone https://github.com/godofphonk/ServerEye.git
cd ServerEye
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Set Up Infrastructure

**Using Docker Compose:**

```bash
cd deployments
docker-compose up -d postgres redis
```

**Manual Setup:**

```bash
# PostgreSQL
docker run -d --name servereye-postgres \
  -e POSTGRES_DB=servereye \
  -e POSTGRES_USER=servereye \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 postgres:14

# Redis
docker run -d --name servereye-redis \
  -p 6379:6379 redis:7-alpine
```

### 4. Configure Environment

**Bot configuration:**

```bash
export TELEGRAM_TOKEN="your_bot_token_here"
export REDIS_URL="redis://localhost:6379"
export DATABASE_URL="postgresql://servereye:password@localhost:5432/servereye?sslmode=disable"
```

**Agent configuration:**

Create `/etc/servereye/config.yaml`:

```yaml
server:
  name: "dev-server"
  description: "Development server"
  secret_key: "srv_your_secret_key_here"

api:
  base_url: "http://localhost:8080"
  timeout: "30s"

metrics:
  cpu_temperature: true
  interval: "30s"

logging:
  level: "debug"
  file: "/var/log/servereye/agent.log"
```

### 5. Run Services

**Terminal 1 - Bot:**
```bash
go run cmd/bot/main.go
```

**Terminal 2 - Agent:**
```bash
go run cmd/agent/main.go -config /etc/servereye/config.yaml
```

## Building

### Build All Binaries

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o servereye-bot-linux cmd/bot/main.go
GOOS=linux GOARCH=amd64 go build -o servereye-agent-linux cmd/agent/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o servereye-bot.exe cmd/bot/main.go
GOOS=windows GOARCH=amd64 go build -o servereye-agent.exe cmd/agent/main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o servereye-bot-darwin cmd/bot/main.go
GOOS=darwin GOARCH=amd64 go build -o servereye-agent-darwin cmd/agent/main.go
```

### Build Docker Images

```bash
# Bot
docker build -f deployments/Dockerfile.bot -t servereye-bot:latest .

# Agent
docker build -f deployments/Dockerfile.agent -t servereye-agent:latest .
```

## Testing

### Unit Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...

# Specific package
go test ./pkg/docker/...
```

### Integration Tests

```bash
# Requires running infrastructure
go test -tags=integration ./...
```

### Test Coverage Report

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Code Quality

### Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run

# Auto-fix issues
golangci-lint run --fix
```

### Formatting

```bash
# Format all files
go fmt ./...

# Check formatting
gofmt -l .
```

### Vet

```bash
# Static analysis
go vet ./...
```

## Development Workflow

### 1. Feature Branch

```bash
git checkout -b feature/my-feature
```

### 2. Make Changes

Follow these conventions:
- Use `gofmt` for formatting
- Write tests for new features
- Update documentation
- Follow existing code structure

### 3. Test Changes

```bash
go test ./...
golangci-lint run
```

### 4. Commit

Use conventional commits:
```bash
git commit -m "feat: add new monitoring command"
git commit -m "fix: resolve Redis subscription race condition"
git commit -m "docs: update architecture documentation"
```

**Commit types:**
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Adding tests
- `chore:` - Maintenance tasks
- `security:` - Security improvements

### 5. Push

```bash
git push origin feature/my-feature
```

## Debugging

### Enable Debug Logging

**Bot:**
```go
logger.SetLevel(logrus.DebugLevel)
```

**Agent:**
```yaml
logging:
  level: "debug"
```

### Useful Debug Commands

```bash
# Check Redis pub/sub
redis-cli
> SUBSCRIBE cmd:srv_*
> PUBLISH cmd:srv_abc123 "test"

# Check PostgreSQL
psql -U servereye -d servereye
> SELECT * FROM servers;

# Check Docker
docker ps
docker logs servereye-bot
```

### Common Issues

**Issue 1: "Connection refused" to Redis**
```bash
# Check Redis is running
docker ps | grep redis
redis-cli ping
```

**Issue 2: "Database connection failed"**
```bash
# Check PostgreSQL
docker ps | grep postgres
psql -U servereye -d servereye -c "SELECT 1"
```

**Issue 3: Agent timeout**
```bash
# Check agent is polling
journalctl -u servereye-agent -f

# Verify HTTP API is accessible
curl http://localhost:8080/health
```

## Performance Profiling

### CPU Profiling

```bash
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

### Memory Profiling

```bash
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

### Runtime Profiling

Add to bot/agent:
```go
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Access at: http://localhost:6060/debug/pprof/

## Deployment

### Production Build

```bash
# Optimized binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s" \
  -o servereye-bot-linux \
  cmd/bot/main.go
```

### Generate Checksums

```bash
sha256sum servereye-bot-linux >> SHA256SUMS
sha256sum servereye-agent-linux >> SHA256SUMS
```

### Docker Production

```bash
docker build -f deployments/Dockerfile.bot \
  -t servereye-bot:v1.0.0 .
  
docker push servereye-bot:v1.0.0
```

## Contributing

We welcome contributions! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

**Before submitting:**
- ✅ All tests pass
- ✅ Linter is clean
- ✅ Documentation updated
- ✅ Commit messages follow conventions

---

**See also:**
- [Architecture](ARCHITECTURE.md)
- [Security Considerations](SECURITY.md)
- [Monitoring & Observability](MONITORING.md)
