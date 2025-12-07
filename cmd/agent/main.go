package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/servereye/servereye/internal/agent"
	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/internal/version"
	"github.com/sirupsen/logrus"
)

const (
	defaultConfigPath = "/etc/servereye/config.yaml"
	defaultLogLevel   = "info"
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
		uninstall   = flag.Bool("uninstall", false, "Uninstall agent completely")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("ServerEye Agent %s\n", version.GetFullVersion())
		return
	}

	// Handle uninstall
	if *uninstall {
		if err := handleUninstall(); err != nil {
			fmt.Printf("Uninstall failed: %v\n", err)
			os.Exit(1)
		}
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
	fmt.Printf("🚀 Installing ServerEye Agent %s\n", version.GetFullVersion())

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

kafka:
  enabled: true
  brokers:
    - "localhost:9092"
  topic_prefix: "metrics"  # Change to "prod" or "dev" for environment isolation
  compression: "snappy"
  max_attempts: 3
  batch_size: 100
  required_acks: 1

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

	// Try to register key via API
	fmt.Println("🔄 Registering key with ServerEye API...")
	hostname, _ := os.Hostname()
	if err := registerKeyWithAPI(secretKey, version.GetVersion(), runtime.GOOS+" "+runtime.GOARCH, hostname); err != nil {
		fmt.Printf("⚠️  API registration failed: %v\n", err)
	} else {
		fmt.Println("✅ Key successfully registered with ServerEye API!")
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

// registerKeyWithAPI registers the key via HTTP API
func registerKeyWithAPI(secretKey, agentVersion, osInfo, hostname string) error {
	apiURL := os.Getenv("SERVEREYE_API_URL")
	if apiURL == "" {
		return fmt.Errorf("SERVEREYE_API_URL environment variable not set")
	}

	req := KeyRegistrationRequest{
		SecretKey:    secretKey,
		AgentVersion: agentVersion,
		OSInfo:       osInfo,
		Hostname:     hostname,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(apiURL+"/api/v1/register-key", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to register key: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
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

	// Get bot URL from environment variable
	botURL := os.Getenv("SERVEREYE_BOT_URL")
	if botURL == "" {
		return fmt.Errorf("SERVEREYE_BOT_URL environment variable not set")
	}

	// Try to register with bot (non-blocking)
	fmt.Printf("🔗 Connecting to bot at: %s\n", botURL+"/api/register-key")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(botURL+"/api/register-key", "application/json", bytes.NewBuffer(jsonData))
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

// registerKeyInDatabase inserts the generated key directly into the database
func registerKeyInDatabase(secretKey, agentVersion, osInfo, hostname string) error {
	dbURL := os.Getenv("SERVEREYE_DB_URL")
	if dbURL == "" {
		return fmt.Errorf("SERVEREYE_DB_URL environment variable not set")
	}

	// Open database connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Insert key with all metadata
	query := `
		INSERT INTO public.generated_keys (secret_key, agent_version, os_info, hostname, status)
		VALUES ($1, $2, $3, $4, 'generated')
		ON CONFLICT (secret_key) DO NOTHING
	`

	_, err = db.Exec(query, secretKey, agentVersion, osInfo, hostname)
	if err != nil {
		return fmt.Errorf("failed to insert key into database: %v", err)
	}

	fmt.Printf("✅ Key successfully registered in database: %s\n", secretKey)
	return nil
}

// handleUninstall handles the complete uninstallation of the agent
func handleUninstall() error {
	// Check if running as root
	if os.Geteuid() != 0 {
		return fmt.Errorf("uninstall requires root privileges. Run with sudo: sudo servereye-agent --uninstall")
	}

	fmt.Println("🗑️  Uninstalling ServerEye Agent...")

	// Try to find and execute the uninstall script
	scriptPaths := []string{
		"./uninstall-servereye.sh",
		"/opt/servereye/uninstall-servereye.sh",
	}

	var scriptPath string
	for _, path := range scriptPaths {
		if _, err := os.Stat(path); err == nil {
			scriptPath = path
			break
		}
	}

	if scriptPath == "" {
		return fmt.Errorf("uninstall script not found. Please locate uninstall-servereye.sh and run it manually with sudo")
	}

	fmt.Printf("📋 Running uninstall script: %s\n", scriptPath)

	// Execute the uninstall script
	cmd := exec.Command("bash", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("uninstall script execution failed: %v", err)
	}

	fmt.Println("✅ ServerEye Agent uninstalled successfully!")
	return nil
}
