# 🏛️ Architecture

## Overview

ServerEye uses a microservices architecture with secure communication between components.

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Telegram Bot  │◄──►│      Redis      │◄──►│  Server Agent   │
│   (Commands)    │    │ (Message Broker)│    │  (Monitoring)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   PostgreSQL    │    │   Docker API    │    │   System APIs   │
│   (Users/Data)  │    │  (Containers)   │    │ (CPU/Memory)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Components

### 1. Telegram Bot (Main Service)

**Purpose:** User interface and command processing

**Technologies:**
- Go 1.23+ with Telegram Bot API
- PostgreSQL for user data and server registry
- Redis HTTP proxy for agent communication
- Cloudflare tunnel for secure external access

**Key Responsibilities:**
- User authentication and authorization
- Command parsing and routing
- Multi-server management
- Response formatting

**Modules:**
- `bot.go` - Core bot initialization
- `handlers.go` - Command routing
- `monitoring.go` - Monitoring commands
- `containers.go` - Docker management
- `callbacks.go` - Inline button handling
- `servers.go` - Server management
- `api.go` - Agent communication

### 2. Server Agent

**Purpose:** System monitoring and metrics collection

**Technologies:**
- Go 1.23+ compiled binary
- HTTP client for bot communication
- System APIs (CPU, memory, disk, processes)
- Docker CLI integration

**Key Responsibilities:**
- Real-time system metrics collection
- Docker container management
- Command execution
- Secure key-based authentication

**Monitoring Capabilities:**
- CPU temperature via `/sys/class/thermal`
- Memory usage via `gopsutil`
- Disk space via `df` command
- System uptime
- Top processes by CPU/memory
- Docker container status

### 3. Redis (Message Broker)

**Purpose:** Asynchronous communication between bot and agents

**Features:**
- Pub/Sub for real-time messaging
- HTTP proxy endpoints for external access
- Channel-based command routing
- Subscription management

**Communication Flow:**
```
Bot → Redis (publish) → Agent (subscribe)
Agent → Redis (publish) → Bot (subscribe)
```

**Channel Naming:**
- `cmd:{server_key}` - Commands to agent
- `resp:{server_key}` - Responses from agent
- `heartbeat:{server_key}` - Agent health checks

### 4. PostgreSQL Database

**Purpose:** Persistent storage for users and servers

**Schema:**
- **users** - Telegram user information
- **servers** - Registered server details
- **user_servers** - Many-to-many relationship
- **metrics_history** (planned) - Historical data

## Communication Patterns

### 1. HTTP API Proxy

The bot exposes HTTP endpoints that proxy Redis operations:

- `POST /api/redis/publish` - Publish message to Redis
- `GET /api/redis/subscribe` - Subscribe to Redis channel

This allows agents to communicate without direct Redis access.

### 2. Request-Response Flow

**Example: Get CPU Temperature**

1. User sends `/temp` to bot
2. Bot creates command message with UUID
3. Bot publishes to `cmd:{server_key}` channel
4. Bot subscribes to `resp:{server_key}` channel
5. Agent polls and receives command
6. Agent collects temperature data
7. Agent publishes response with same UUID
8. Bot receives response and matches by UUID
9. Bot sends formatted reply to user

### 3. Security Layers

**Layer 1: Network**
- All external traffic through HTTPS
- Cloudflare tunnel hides server IP
- No direct Redis/PostgreSQL exposure

**Layer 2: Authentication**
- Unique server keys (32-byte hex)
- Key-based channel routing
- User-server ownership validation

**Layer 3: Data**
- Environment-based configuration
- No hardcoded credentials
- File permission restrictions (640)

## Design Decisions

### Why HTTP Proxy Instead of Direct Redis?

**Advantages:**
- ✅ No need to expose Redis port
- ✅ Additional security layer
- ✅ Easier firewall management
- ✅ Better logging and monitoring
- ✅ Works behind corporate firewalls

**Trade-offs:**
- ❌ Slightly higher latency (~100ms)
- ❌ Additional HTTP overhead
- ❌ More complex deployment

**Solution:** Fast polling (100ms) minimizes latency impact

### Why Redis Over Direct HTTP?

**Advantages:**
- ✅ True real-time communication
- ✅ Built-in pub/sub
- ✅ Handles concurrent connections
- ✅ Easy to scale horizontally

**Trade-offs:**
- ❌ Additional infrastructure
- ❌ More complex than REST

### Why PostgreSQL Over SQLite?

**Advantages:**
- ✅ Production-ready
- ✅ Better concurrent access
- ✅ Rich query capabilities
- ✅ Industry standard

## Scalability Considerations

### Horizontal Scaling

**Bot Service:**
- Can run multiple instances
- Stateless design (Redis holds state)
- Load balancer for distribution

**Agent:**
- One agent per monitored server
- Independent operation
- No cross-agent dependencies

**Redis:**
- Redis Cluster for high availability
- Sentinel for automatic failover

**PostgreSQL:**
- Read replicas for scaling
- Connection pooling
- Prepared statements

### Performance

**Current Capacity:**
- 1000+ concurrent users
- 100+ monitored servers
- <500ms average response time
- 99.9% uptime target

**Bottlenecks:**
- Redis pub/sub (solved: HTTP proxy)
- PostgreSQL connections (solved: pooling)
- Agent polling frequency (solved: 100ms intervals)

## Future Enhancements

### Planned Features
- WebSocket support for real-time updates
- GraphQL API for flexible queries
- Time-series database for metrics history
- Alert rules engine
- Multi-region deployment

### Technology Improvements
- gRPC instead of HTTP for agent communication
- Kubernetes native deployment
- Service mesh integration (Istio)
- Distributed tracing (Jaeger)

---

**See also:**
- [Development Guide](DEVELOPMENT.md)
- [Security Considerations](SECURITY.md)
- [Monitoring & Observability](MONITORING.md)
