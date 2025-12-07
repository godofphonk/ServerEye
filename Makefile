# ServerEye Agent Makefile

.PHONY: build build-agent test clean docker-build docker-up docker-down install-agent

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary name
AGENT_BINARY=servereye-agent

# Build directories
BUILD_DIR=build

# Default target
all: build

# Build agent only
build: build-agent

# Версия и build info (определяется выше для release, но дублируем для clarity)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS = -X github.com/servereye/servereye/internal/version.Version=$(VERSION) \
	-X github.com/servereye/servereye/internal/version.BuildDate=$(BUILD_DATE) \
	-X github.com/servereye/servereye/internal/version.GitCommit=$(GIT_COMMIT)

# Build agent
build-agent:
	@echo "Building agent $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(AGENT_BINARY) ./cmd/agent
	@echo "✅ Agent built: $(BUILD_DIR)/$(AGENT_BINARY)"

# Run all tests
test:
	@echo "Running all tests..."
	$(GOTEST) -v -short ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -short -coverprofile=coverage.txt -covermode=atomic ./...
	@echo ""
	@echo "📊 Coverage Report:"
	go tool cover -func=coverage.txt | grep -E "internal/agent|^total:"

# Test specific module
test-agent:
	@echo "Testing internal/agent..."
	$(GOTEST) -v -short -cover ./internal/agent/...

test-pkg:
	@echo "Testing pkg modules..."
	$(GOTEST) -v -short -cover ./pkg/...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Copy Linux binaries to downloads folder
downloads: release
	@echo "Copying Linux binaries to downloads folder..."
	@mkdir -p downloads
	@cp $(BUILD_DIR)/$(AGENT_BINARY)-linux-amd64 downloads/
	@cp $(BUILD_DIR)/$(AGENT_BINARY)-linux-arm64 downloads/
	@chmod +x downloads/$(AGENT_BINARY)-linux-amd64
	@chmod +x downloads/$(AGENT_BINARY)-linux-arm64
	@echo "✅ Linux binaries copied to downloads/"
	@ls -la downloads/

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run

# Build Docker images
docker-build:
	@echo "Building Docker images..."
	docker build -f deployments/Dockerfile.bot -t servereye/bot:latest .
	docker build -f deployments/Dockerfile.agent -t servereye/agent:latest .

# Start services with Docker Compose
docker-up:
	@echo "Starting services..."
	cd deployments && docker-compose up -d

# Stop services
docker-down:
	@echo "Stopping services..."
	cd deployments && docker-compose down

# View logs
docker-logs:
	cd deployments && docker-compose logs -f

# Install agent (Linux only)
install-agent: build-agent
	@echo "Installing agent..."
	sudo cp $(BUILD_DIR)/$(AGENT_BINARY) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(AGENT_BINARY)
	@echo "Agent installed to /usr/local/bin/$(AGENT_BINARY)"
	@echo "Run 'sudo $(AGENT_BINARY) --install' to complete setup"

# Development targets
dev-agent:
	@echo "Running agent in development mode..."
	$(GOCMD) run ./cmd/agent --log-level=debug

# Generate mocks (requires mockgen)
mocks:
	@echo "Generating mocks..."
	go generate ./...

# Security scan (requires gosec)
security:
	@echo "Running security scan..."
	gosec ./...

# Check for vulnerabilities (requires govulncheck)
vuln-check:
	@echo "Checking for vulnerabilities..."
	govulncheck ./...

# Release build with optimizations and version
RELEASE_LDFLAGS = -w -s \
	-X github.com/servereye/servereye/internal/version.Version=$(VERSION) \
	-X github.com/servereye/servereye/internal/version.BuildDate=$(BUILD_DATE) \
	-X github.com/servereye/servereye/internal/version.GitCommit=$(GIT_COMMIT)

release: clean
	@echo "Building release binaries for version $(VERSION)..."
	@echo "Build date: $(BUILD_DATE)"
	@echo "Git commit: $(GIT_COMMIT)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags="$(RELEASE_LDFLAGS)" -o $(BUILD_DIR)/$(AGENT_BINARY)-linux-amd64 ./cmd/agent
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags="$(RELEASE_LDFLAGS)" -o $(BUILD_DIR)/$(AGENT_BINARY)-linux-arm64 ./cmd/agent
	@echo "✅ Release build complete!"
	@$(BUILD_DIR)/$(AGENT_BINARY)-linux-amd64 --version

# Показать текущую версию
version:
	@echo "Version: $(VERSION)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Git Commit: $(GIT_COMMIT)"

# Help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  build         - Build agent"
	@echo "  build-agent   - Build agent only"
	@echo "  release       - Build optimized release binaries"
	@echo "  downloads     - Copy Linux binaries to downloads folder"
	@echo ""
	@echo "Tests:"
	@echo "  test          - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-agent    - Test agent module only"
	@echo "  test-pkg      - Test pkg modules only"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  security      - Run security scan"
	@echo "  vuln-check    - Check for vulnerabilities"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-up     - Start services with Docker Compose"
	@echo "  docker-down   - Stop services"
	@echo "  docker-logs   - View service logs"
	@echo ""
	@echo "Development:"
	@echo "  dev-agent     - Run agent in development mode"
	@echo ""
	@echo "Utilities:"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  version       - Show current version"
	@echo "  help          - Show this help"
