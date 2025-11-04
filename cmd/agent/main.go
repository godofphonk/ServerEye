package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/servereye/servereye/internal/agent"
	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/internal/version"
	"github.com/sirupsen/logrus"
)

const (
	defaultConfigPath = "/etc/servereye/config.yaml"
	defaultLogLevel   = "info"
	defaultBotURL     = "http://192.168.0.105:8090" // ServerEye bot URL
)

// KeyRegistrationRequest represents a request to register a generated key
type KeyRegistrationRequest struct {
	SecretKey    string `json:"secret_key"`
	AgentVersion string `json:"agent_version,omitempty"`
	OSInfo       string `json:"os_info,omitempty"`
	Hostname     string `json:"hostname,omitempty"`
}

func main() {
	var (
		configPath  = flag.String("config", defaultConfigPath, "Path to configuration file")
		logLevel    = flag.String("log-level", defaultLogLevel, "Log level (debug, info, warn, error)")
		install     = flag.Bool("install", false, "Install agent and generate secret key")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("ServerEye Agent v%s\n", version.GetFullVersion())
		return
	}

	// Handle installation
	if *install {
		if err := handleInstall(); err != nil {
			fmt.Printf("Installation failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Setup logger
	logger := setupLogger(*logLevel)

	// Load configuration
	cfg, err := config.LoadAgentConfig(*configPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Create and start agent
	agentInstance, err := agent.New(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create agent")
	}

	if err := agentInstance.Start(); err != nil {
		logger.WithError(err).Fatal("Failed to start agent")
	}

	logger.Info("ServerEye Agent started successfully")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down agent...")
	if err := agentInstance.Stop(); err != nil {
		logger.WithError(err).Error("Error during shutdown")
	}
}

// setupLogger configures and returns a logger instance
func setupLogger(level string) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Set formatter
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})

	return logger
}

// handleInstall handles the installation process
func handleInstall() error {
	fmt.Println("🚀 Installing ServerEye Agent...")

	// Generate secret key
	secretKey, err := generateSecretKey()
	if err != nil {
		return fmt.Errorf("failed to generate secret key: %v", err)
	}

	// Try to use system config directory, fallback to user home
	configDir := "/etc/servereye"
	logPath := "/var/log/servereye/agent.log"

	// Test if we can write to system directories
	testFile := configDir + "/test"
	if err := os.MkdirAll(configDir, 0755); err != nil || os.WriteFile(testFile, []byte("test"), 0600) != nil {
		// Fallback to user home directory
		homeDir, _ := os.UserHomeDir()
		configDir = homeDir + "/.servereye"
		logPath = configDir + "/logs/agent.log"
		fmt.Printf("📁 Using user config directory: %s\n", configDir)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %v", err)
		}
	} else {
		// Clean up test file
		os.Remove(testFile)
	}

	// Create default configuration
	configContent := fmt.Sprintf(`server:
  name: "Production Server"
  description: "ServerEye monitored server"
  secret_key: "%s"

redis:
  address: "localhost:6379"
  password: ""
  db: 0

metrics:
  cpu_temperature: true
  interval: "30s"

logging:
  level: "info"
  file: "%s"
`, secretKey, logPath)

	configPath := fmt.Sprintf("%s/config.yaml", configDir)
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("failed to write configuration: %v", err)
	}

	// Create log directory based on chosen config directory
	var logDir string
	if configDir == "/etc/servereye" {
		logDir = "/var/log/servereye"
	} else {
		logDir = configDir + "/logs"
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Try to register key with bot
	fmt.Println("🔄 Registering key with ServerEye bot...")
	if err := registerKeyWithBot(secretKey); err != nil {
		fmt.Printf("⚠️  Key registration failed: %v\n", err)
	}

	// Display success message
	fmt.Println("✅ ServerEye Agent installed successfully!")
	fmt.Println()
	fmt.Printf("🔑 Your secret key: %s\n", secretKey)
	fmt.Println()
	fmt.Println("📱 To connect to Telegram bot:")
	fmt.Println("1. Find @ServerEyeBot")
	fmt.Println("2. Send /start command")
	fmt.Printf("3. Enter your key: %s\n", secretKey)
	fmt.Println()
	fmt.Printf("📝 Configuration file: %s\n", configPath)
	fmt.Printf("📋 Log file: %s\n", fmt.Sprintf("%s/agent.log", logDir))

	return nil
}

// generateSecretKey generates a cryptographically secure secret key
func generateSecretKey() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "srv_" + hex.EncodeToString(bytes), nil
}

// registerKeyWithBot registers the generated key with the bot via HTTP API
func registerKeyWithBot(secretKey string) error {
	hostname, _ := os.Hostname()

	req := KeyRegistrationRequest{
		SecretKey:    secretKey,
		AgentVersion: version.GetVersion(),
		OSInfo:       runtime.GOOS + " " + runtime.GOARCH,
		Hostname:     hostname,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	// Try to register with bot (non-blocking)
	fmt.Printf("🔗 Connecting to bot at: %s\n", defaultBotURL+"/api/register-key")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(defaultBotURL+"/api/register-key", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// Don't fail installation if bot is not available
		fmt.Printf("⚠️  Could not register key with bot: %v\n", err)
		fmt.Printf("📝 JSON payload: %s\n", string(jsonData))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Key automatically registered with ServerEye bot!")
	} else {
		// Read response body for error details
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		fmt.Printf("⚠️  Key registration failed - Status: %d, Response: %s\n", resp.StatusCode, string(body[:n]))
	}

	return nil
}
