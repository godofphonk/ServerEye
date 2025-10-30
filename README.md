# 🔍 ServerEye - Server Monitoring via Telegram Bot

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker)](https://www.docker.com/)
[![Telegram](https://img.shields.io/badge/Telegram-Bot-26A5E4?style=flat-square&logo=telegram)](https://telegram.org/)

**ServerEye** is a server monitoring system that lets you monitor your servers and manage Docker containers through a Telegram bot. Built with Go and modern architecture patterns.

## 🚀 Key Features

### 📊 **Real-time Monitoring**
- **CPU Temperature** monitoring with alerts
- **Memory Usage** tracking with detailed breakdown
- **Disk Space** monitoring across all mounted drives
- **System Uptime** and boot time information
- **Process Management** with top CPU/memory consumers
- **Docker Container** lifecycle management

### 🏗️ **Architecture**
- **Microservices** design with Redis message broker
- **Multi-server** support with secure key-based authentication
- **Modular codebase** with clean separation of concerns
- **Docker containerization** for easy deployment

## 🏛️ Architecture Overview

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

**Plus supporting modules**: `interfaces.go`, `errors.go`, `metrics.go`, `validator.go`, `logger.go`

## 🛠️ Technology Stack

- **Backend**: Go 1.23+ with modern idioms
- **Database**: PostgreSQL with connection pooling
- **Message Broker**: Redis for real-time communication
- **Containerization**: Docker & Docker Compose
- **Monitoring**: Custom metrics collection
- **API**: Telegram Bot API with webhook support
- **Logging**: Structured logging with Logrus

## 📖 How to Use

### 1. One-Line Installation
```bash
# Automatic installation with systemd service
curl -sSL https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install.sh | sudo bash
```

### 2. Manual Installation
```bash
# Download and run installer
wget https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install-agent.sh
sudo chmod +x install-agent.sh
sudo ./install-agent.sh
```

### 3. What Happens Automatically
- ✅ Downloads and installs the agent
- ✅ Generates a unique server key
- ✅ Registers the key with ServerEye bot
- ✅ Starts the monitoring service
- ✅ Enables auto-start on boot

### 4. Start Monitoring
Your server is now monitored! Find **@ServerEyeBot** in Telegram and use:
```
/temp           - CPU temperature
/memory         - Memory usage
/disk           - Disk usage
/containers     - Docker containers
/status         - Server status
```

## 🎯 Usage Examples

### Basic Monitoring
```
/start          - Initialize bot and register user
/temp           - Get CPU temperature
/memory         - Check memory usage
/disk           - View disk space
/containers     - List Docker containers
/status         - Overall system status
```

### Multi-Server Management
```
/add srv_abc123 MyServer    - Connect new server
/servers                    - List all servers
/temp 2                     - Get temperature from server #2
/rename_server srv_abc123 ProductionServer
```

### Container Management
```
/start_container nginx      - Start container
/stop_container nginx       - Stop container
/restart_container nginx    - Restart container
```

## 🏗️ Development

### Project Structure
```
ServerEye/
├── cmd/
│   ├── bot/           # Telegram bot entry point
│   └── agent/         # Server agent entry point
├── internal/
│   ├── bot/           # Bot implementation (modular)
│   ├── agent/         # Agent implementation
│   └── config/        # Configuration management
├── pkg/
│   ├── protocol/      # Inter-service communication
│   └── redis/         # Redis client wrapper
├── deployments/       # Docker & K8s configs
├── docs/             # Documentation
└── scripts/          # Build & deployment scripts
```

## 📊 Monitoring & Observability

### Built-in Metrics
- Command execution latency
- Error rates by type
- Active user count
- Agent response times
- Database connection pool stats

### Health Checks
```bash
# Bot health
curl http://localhost:8080/health

# Metrics endpoint
curl http://localhost:8080/metrics
```


## 📄 License

This project is open source. Feel free to use and modify.

#