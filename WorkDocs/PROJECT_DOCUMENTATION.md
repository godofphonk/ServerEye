# ServerEye - Enterprise Server Monitoring System

## Overview

ServerEye is a comprehensive enterprise-level server monitoring system written in Go. It provides real-time monitoring capabilities through a lightweight agent that communicates with a central server via WebSocket connections.

## Architecture

### High-Level Architecture

```
┌─────────────────┐    WebSocket    ┌─────────────────┐
│   ServerEye     │◄──────────────►│   ServerEye     │
│   Telegram Bot  │                 │   Backend API   │
└─────────────────┘                 └─────────────────┘
                                            │
                                            ▼
                                    ┌─────────────────┐
                                    │   ServerEye     │
                                    │   Agent         │
                                    │   (This Repo)   │
                                    └─────────────────┘
```

### Agent Architecture

The ServerEye agent follows a modular architecture with clear separation of concerns:

```
cmd/agent/                 # Entry point and CLI
├── main.go               # Main application logic
└── installation/        # Install/uninstall functionality

internal/                  # Private business logic
├── agent/               # Core agent implementation
│   ├── agent.go         # Main agent struct and lifecycle
│   ├── heartbeat.go     # Heartbeat mechanism
│   ├── helpers.go       # Utility functions
│   ├── metrics_collector.go  # Metrics collection logic
│   └── metric_adapter.go     # Metric format adaptation
├── config/              # Configuration management
│   ├── config.go        # Configuration structs and loading
│   └── config_test.go   # Configuration tests
└── version/             # Version information
    ├── version.go       # Version variables
    └── version_test.go  # Version tests

pkg/                      # Public reusable packages
├── commands/            # Command processing
│   ├── http_consumer.go     # HTTP command consumer
│   └── websocket_consumer.go # WebSocket command consumer
├── docker/              # Docker integration
│   ├── client.go            # Docker client wrapper
│   ├── client_test.go       # Docker client tests
│   ├── health.go            # Container health checks
│   └── management.go        # Container management
├── metrics/             # Metrics collection and publishing
│   ├── cpu.go               # CPU temperature metrics
│   ├── cpu_test.go          # CPU metrics tests
│   ├── system.go            # System metrics
│   ├── system_test.go       # System metrics tests
│   └── websocket_publisher.go # WebSocket metrics publisher
├── protocol/            # Communication protocol
│   ├── message.go           # Message definitions
│   └── message_test.go      # Protocol tests
├── websocket/           # WebSocket client
│   └── client.go            # WebSocket implementation
├── types/               # Type definitions
│   ├── metric.go            # Metric types
│   └── websocket_adapter.go # WebSocket adapters
└── publisher/           # Metrics publishing
    └── multi_publisher.go   # Multiple publisher support
```

## Core Components

### 1. Agent Core (`internal/agent/`)

The agent core manages the entire lifecycle of the monitoring agent:

#### Main Agent (`agent.go`)
- **Agent struct**: Central coordinator with all subsystems
- **WebSocket management**: Publisher and consumer initialization
- **Lifecycle management**: Start/stop operations with graceful shutdown
- **Command handling**: Interface for processing incoming commands

#### Heartbeat System (`heartbeat.go`)
- **Regular heartbeat**: Every 60 seconds via WebSocket
- **Connection monitoring**: Detects connection issues
- **Status reporting**: Agent health and connectivity status

#### Metrics Collection (`metrics_collector.go`)
- **Periodic collection**: Every 30 seconds (configurable)
- **Multi-source metrics**: CPU, memory, disk, Docker containers
- **Buffered publishing**: Handles offline scenarios

### 2. Configuration System (`internal/config/`)

#### Configuration Structure
```yaml
server:
  name: "server-hostname"
  description: "Server description"
  secret_key: "generated-secret-key"
  server_id: "unique-server-id"

api:
  base_url: "https://api.servereye.dev"
  api_key: "api-key"
  timeout: "30s"

websocket:
  enabled: true
  url: "wss://api.servereye.dev/ws"
  reconnect_interval: "5s"
  max_reconnect_attempts: 10
  ping_interval: "30s"
  write_timeout: "10s"
  read_timeout: "10s"
  handshake_timeout: "10s"
  buffer_size: 1000
  enable_compression: true
  metric_buffer_size: 100
  metric_buffer_flush: "30s"
  command_queue_size: 100
  command_timeout: "30s"

metrics:
  cpu_usage: false
  memory_usage: true
  disk_usage: false
  cpu_temperature: true
  interval: "30s"

logging:
  level: "info"
  file: "/var/log/servereye/agent.log"
```

#### Configuration Features
- **YAML format**: Human-readable configuration
- **Validation**: Ensures required fields are present
- **Defaults**: Sensible defaults for optional settings
- **Environment support**: Flexible deployment options

### 3. WebSocket Communication

#### Publisher (`pkg/metrics/websocket_publisher.go`)
- **Metric streaming**: Real-time metric transmission
- **Buffer management**: Handles connection interruptions
- **Reconnection logic**: Automatic recovery with exponential backoff
- **Compression**: Optional message compression for bandwidth efficiency

#### Consumer (`pkg/commands/websocket_consumer.go`)
- **Command processing**: Handles incoming commands from backend
- **Queue management**: Buffered command processing
- **Timeout handling**: Prevents hanging operations
- **Error handling**: Graceful error recovery

#### Client (`pkg/websocket/client.go`)
- **Connection management**: WebSocket lifecycle
- **Authentication**: Secure connection establishment
- **Message framing**: Proper message formatting
- **Health monitoring**: Connection health checks

### 4. Metrics System

#### CPU Metrics (`pkg/metrics/cpu.go`)
- **Temperature monitoring**: CPU thermal sensors
- **Platform support**: Linux sensor reading
- **Error handling**: Sensor unavailable scenarios
- **Unit conversion**: Celsius/Kelvin support

#### System Metrics (`pkg/metrics/system.go`)
- **Memory usage**: RAM utilization statistics
- **Disk usage**: Storage capacity monitoring
- **Network stats**: Interface traffic monitoring
- **Process information**: Running processes tracking

#### Docker Integration (`pkg/docker/`)
- **Container listing**: Status and information
- **Container management**: Start/stop/restart operations
- **Health checks**: Container health monitoring
- **Resource usage**: Container resource consumption

### 5. Protocol (`pkg/protocol/`)

#### Message Types
```go
// Commands from bot to agent
const (
    TypeGetCPUTemp       MessageType = "get_cpu_temp"
    TypeGetSystemInfo    MessageType = "get_system_info"
    TypeGetContainers    MessageType = "get_containers"
    TypeStartContainer   MessageType = "start_container"
    TypeStopContainer    MessageType = "stop_container"
    TypeRestartContainer MessageType = "restart_container"
    TypeGetMemoryInfo    MessageType = "get_memory_info"
    TypeGetDiskInfo      MessageType = "get_disk_info"
    TypePing             MessageType = "ping"
)

// Responses from agent to bot
const (
    TypeCPUTempResponse         MessageType = "cpu_temp_response"
    TypeSystemInfoResponse      MessageType = "system_info_response"
    TypeContainersResponse      MessageType = "containers_response"
    TypeContainerActionResponse MessageType = "container_action_response"
    TypeMemoryInfoResponse      MessageType = "memory_info_response"
    TypeDiskInfoResponse        MessageType = "disk_info_response"
    TypePong                    MessageType = "pong"
    TypeErrorResponse           MessageType = "error_response"
)
```

#### Message Structure
```go
type Message struct {
    ID        string      `json:"id"`
    Type      MessageType `json:"type"`
    Timestamp time.Time   `json:"timestamp"`
    ServerID  string      `json:"server_id,omitempty"`
    ServerKey string      `json:"server_key,omitempty"`
    Version   string      `json:"version"`
    Payload   interface{} `json:"payload"`
}
```

## Installation and Deployment

### Automated Installation

The project includes a comprehensive installation script (`scripts/install-agent.sh`):

#### Features
- **Binary verification**: SHA256 checksum validation
- **User management**: Creates dedicated `servereye` user
- **Systemd service**: Automatic service configuration
- **API registration**: Server registration with backend
- **Configuration generation**: Automatic config creation
- **Security setup**: Proper permissions and isolation

#### Installation Process
1. **Dependency check**: Verifies required tools
2. **User creation**: Creates non-root system user
3. **Directory setup**: Creates required directories
4. **Binary download**: Fetches latest release with verification
5. **Configuration**: Generates config with server registration
6. **Service setup**: Installs and starts systemd service
7. **Verification**: Confirms successful installation

### Docker Deployment

#### Dockerfile (`deployments/Dockerfile.agent`)
```dockerfile
# Multi-stage build for optimization
FROM golang:1.24-alpine AS builder
# Build stage with compilation

FROM alpine:latest
# Runtime stage with minimal footprint
```

#### Features
- **Multi-stage build**: Minimal final image size
- **Security**: Non-root user execution
- **Alpine Linux**: Small and secure base image
- **Proper permissions**: Secure file ownership

## CI/CD Pipeline

### GitHub Actions (`.github/workflows/ci.yml`)

#### Build and Test
- **Go 1.24**: Latest stable Go version
- **Multi-module testing**: Individual module testing
- **Coverage reporting**: Detailed coverage analysis
- **Codecov integration**: Coverage tracking

#### Code Quality
- **golangci-lint**: Comprehensive Go linting
- **gosec**: Security vulnerability scanning
- **govulncheck**: Dependency vulnerability checking
- **Format validation**: Code formatting checks

#### Pipeline Stages
1. **Build**: Compilation and basic testing
2. **Module tests**: Individual component testing
3. **Linting**: Code quality checks
4. **Security**: Vulnerability scanning
5. **Coverage**: Test coverage analysis

## Development Workflow

### Build System (`Makefile`)

#### Primary Targets
```makefile
build-agent          # Build agent binary
release              # Build optimized release binaries
test                 # Run all tests
test-coverage        # Run tests with coverage
lint                 # Run code linting
security             # Run security scan
docker-build         # Build Docker image
install-agent        # Install agent locally
```

#### Development Targets
```makefile
dev-agent           # Run agent in development mode
fmt                 # Format code
deps                # Download dependencies
mocks               # Generate test mocks
clean               # Clean build artifacts
```

### Version Management

#### Version Information (`internal/version/version.go`)
```go
var (
    Version   = "1.0.0"  // Set during build
    BuildDate = "dev"    // Set during build
    GitCommit = "dev"    // Set during build
)
```

#### Build Process
- **LDFlags**: Version injection during build
- **Git integration**: Automatic commit hash inclusion
- **Release tagging**: Semantic versioning support

## Security Considerations

### Agent Security
- **Non-root execution**: Runs as dedicated user
- **File permissions**: Restricted access to sensitive files
- **Systemd hardening**: Security-focused service configuration
- **Secret management**: Secure key handling

### Communication Security
- **WebSocket encryption**: WSS for secure communication
- **Authentication**: Server key validation
- **API keys**: Secure backend communication
- **Checksum verification**: Binary integrity validation

### System Security
- **Principle of least privilege**: Minimal required permissions
- **Sandboxing**: Isolated execution environment
- **Audit logging**: Comprehensive activity logging
- **Secure defaults**: Safe out-of-the-box configuration

## Monitoring Capabilities

### System Metrics
- **CPU Temperature**: Thermal monitoring with sensor detection
- **Memory Usage**: RAM utilization with detailed breakdown
- **Disk Usage**: Storage capacity and utilization
- **Network Statistics**: Interface traffic and error rates
- **System Uptime**: System availability tracking

### Docker Monitoring
- **Container Status**: Running/stopped/exited containers
- **Container Management**: Start/stop/restart operations
- **Resource Usage**: Container resource consumption
- **Health Checks**: Container health monitoring
- **Image Information**: Container image details

### Process Monitoring
- **Process List**: Running processes with resource usage
- **CPU Usage**: Per-process CPU consumption
- **Memory Usage**: Per-process memory utilization
- **Process Status**: Process state monitoring

## Command Interface

### Supported Commands
- **ping**: Connectivity test
- **get_cpu_temp**: CPU temperature query
- **get_system_info**: System information
- **get_memory_info**: Memory usage statistics
- **get_disk_info**: Disk usage information
- **get_containers**: Docker container listing
- **start_container**: Start specific container
- **stop_container**: Stop specific container
- **restart_container**: Restart specific container

### Command Processing
- **Asynchronous handling**: Non-blocking command execution
- **Timeout management**: Prevents hanging operations
- **Error handling**: Graceful error responses
- **Response formatting**: Structured JSON responses

## Configuration Management

### Environment Variables
- **SERVEREYE_API_URL**: Backend API endpoint
- **SERVEREYE_BACKEND_URL**: Alternative backend URL
- **SERVEREYE_API_KEY**: API authentication key

### Configuration Files
- **Primary config**: `/etc/servereye/config.yaml`
- **Environment file**: `/etc/servereye/agent.env`
- **Log configuration**: Configurable log levels and outputs

### Runtime Configuration
- **Hot reload**: Configuration changes without restart
- **Validation**: Configuration integrity checks
- **Defaults**: Sensible fallback values
- **Override support**: Environment variable overrides

## Troubleshooting and Maintenance

### Logging System
- **Structured logging**: JSON format with logrus
- **Multiple levels**: Debug, info, warn, error
- **File output**: Rotating log files
- **Systemd integration**: Journal logging

### Health Monitoring
- **Heartbeat mechanism**: Regular health checks
- **Connection monitoring**: WebSocket health tracking
- **Error tracking**: Comprehensive error logging
- **Performance metrics**: Internal performance monitoring

### Maintenance Tasks
- **Log rotation**: Automatic log file management
- **Cache cleanup**: Temporary file cleanup
- **Configuration backup**: Config versioning
- **Update management**: Automated update process

## Extensibility and Customization

### Plugin Architecture
- **Modular design**: Easy component extension
- **Interface-based**: Standardized component interfaces
- **Configuration-driven**: Behavior through configuration
- **Metric expansion**: Custom metric support

### Custom Metrics
- **Plugin system**: Extensible metric collection
- **Custom handlers**: User-defined metric processors
- **Format adaptation**: Flexible metric formats
- **Publishing options**: Multiple output destinations

### Integration Points
- **API endpoints**: RESTful API integration
- **Webhook support**: Event-driven notifications
- **Third-party systems**: External system integration
- **Custom protocols**: Protocol extension support

## Performance and Scalability

### Resource Efficiency
- **Low memory footprint**: Optimized memory usage
- **CPU efficiency**: Minimal CPU overhead
- **Network optimization**: Efficient data transmission
- **Storage efficiency**: Minimal disk usage

### Scalability Features
- **Buffer management**: Efficient memory buffering
- **Connection pooling**: WebSocket connection reuse
- **Batch processing**: Efficient metric batching
- **Asynchronous operations**: Non-blocking I/O

### Performance Monitoring
- **Internal metrics**: Agent performance tracking
- **Resource usage**: Memory and CPU monitoring
- **Network efficiency**: Bandwidth utilization
- **Response times**: Command processing latency

## Compliance and Standards

### Enterprise Standards
- **Systemd compliance**: Linux service standards
- **Filesystem hierarchy**: FHS compliance
- **Security standards**: Industry security practices
- **Logging standards**: Structured logging formats

### Documentation Standards
- **Code documentation**: Comprehensive Go doc comments
- **API documentation**: Protocol specification
- **Configuration docs**: Complete configuration reference
- **Troubleshooting guides**: Common issue resolution

This comprehensive documentation covers all aspects of the ServerEye project, from architecture and implementation to deployment and maintenance. The system is designed for enterprise use with a focus on reliability, security, and extensibility.
