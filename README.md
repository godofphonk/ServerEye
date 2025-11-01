# 🔍 ServerEye - Server Monitoring via Telegram Bot

[![CI Status](https://github.com/godofphonk/ServerEye/workflows/CI/badge.svg)](https://github.com/godofphonk/ServerEye/actions)
[![Release](https://img.shields.io/github/v/release/godofphonk/ServerEye?style=flat-square)](https://github.com/godofphonk/ServerEye/releases/latest)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker)](https://www.docker.com/)
[![Telegram](https://img.shields.io/badge/Telegram-Bot-26A5E4?style=flat-square&logo=telegram)](https://telegram.org/)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

**ServerEye** is a production-ready server monitoring system that lets you monitor your servers and manage Docker containers through a Telegram bot. Built with Go and modern microservices architecture.

## ✨ Key Features

- 📊 **Real-time Monitoring** - CPU temperature, memory, disk, uptime, processes
- 🐳 **Docker Management** - Start, stop, restart containers via Telegram
- 🔒 **Secure Architecture** - HTTPS, Cloudflare tunnels, key-based authentication
- 🌐 **Multi-server Support** - Monitor unlimited servers from one bot

## 🚀 Quick Start

### Installation 

**1. Install Agent on Your Server (Automatic):**
```bash
wget -qO- https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install-agent.sh | sudo bash
```

> 🔒 **Security:** Script automatically downloads latest release from GitHub and verifies checksums. [Review script](scripts/install-agent.sh)

**OR Download Binary Manually:**

Get the latest binaries from [GitHub Releases](https://github.com/godofphonk/ServerEye/releases/latest):
- 🐧 Linux (amd64): `servereye-agent-linux-amd64`
- 🐧 Linux (arm64): `servereye-agent-linux-arm64` 
- 🪟 Windows: `servereye-agent-windows-amd64.exe`

```bash
# Example manual install:
wget https://github.com/godofphonk/ServerEye/releases/latest/download/servereye-agent-linux-amd64
chmod +x servereye-agent-linux-amd64
sudo mv servereye-agent-linux-amd64 /usr/local/bin/servereye-agent
```

**2. Connect to Telegram:**

Find **@ServerEyeBot** in Telegram:
```
/start
/add srv_your_key_here MyServer
```

**3. Start Monitoring:**
```
/temp       - CPU temperature  
/memory     - Memory usage
/disk       - Disk space
/containers - Docker containers
```

That's it! Your server is now monitored. 🎉

## 📱 Usage Examples

### Basic Monitoring
```
/temp           Get CPU temperature
/memory         Check memory usage  
/disk           View disk space
/uptime         System uptime
/processes      Top processes
```

### Docker Management
```
/containers              List all containers
/start_container nginx   Start container
/stop_container nginx    Stop container  
/restart_container nginx Restart container
```

### Multi-Server
```
/servers                         List all your servers
/add srv_abc123 Production       Add new server
/temp 2                          Temperature from server #2
/rename_server srv_abc123 Prod   Rename server
```

## 📚 Documentation

- **[Architecture Guide](docs/ARCHITECTURE.md)** - System design, components, scalability
- **[Development Guide](docs/DEVELOPMENT.md)** - Build, test, contribute
- **[Security Guide](docs/SECURITY.md)** - Best practices, known limitations
- **[Monitoring & Observability](docs/MONITORING.md)** - Metrics, logs, alerts

## 🛠️ Technology Stack

- **Backend:** Go 1.23+ with modern idioms
- **Database:** PostgreSQL with connection pooling
- **Message Broker:** Redis for real-time communication
- **Containerization:** Docker & Docker Compose
- **API:** Telegram Bot API with HTTPS
- **Security:** Cloudflare tunnels, TLS encryption

## 📄 License

This project is open source and available under the [MIT License](LICENSE).

