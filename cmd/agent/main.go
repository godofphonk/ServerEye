package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/servereye/servereye/internal/agent"
	"github.com/servereye/servereye/internal/config"
	"github.com/sirupsen/logrus"
)

const (
	defaultConfigPath = "/etc/servereye/config.yaml"
	defaultLogLevel   = "info"
)

func main() {
	var (
		configPath = flag.String("config", defaultConfigPath, "Path to configuration file")
		logLevel   = flag.String("log-level", defaultLogLevel, "Log level (debug, info, warn, error)")
		install    = flag.Bool("install", false, "Install agent and generate secret key")
		version    = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	// Show version
	if *version {
		fmt.Println("ServerEye Agent v1.0.0")
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

	// Create config directory
	configDir := "/etc/servereye"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
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
  file: "/var/log/servereye/agent.log"
`, secretKey)

	configPath := fmt.Sprintf("%s/config.yaml", configDir)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write configuration: %v", err)
	}

	// Create log directory
	logDir := "/var/log/servereye"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
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
