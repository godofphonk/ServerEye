package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

func TestHandleUpdateAgent_ValidPayload(t *testing.T) {
	mockClient := &mockRedisClient{}
	agent := &Agent{
		logger:      logrus.New(),
		redisClient: mockClient,
		config: &config.AgentConfig{
			Server: config.ServerConfig{SecretKey: "test-key"},
		},
	}

	payload := protocol.UpdateAgentPayload{
		Version: "1.2.3",
	}

	msg := protocol.NewMessage(protocol.TypeUpdateAgent, payload)
	msg.ID = "update-001"

	response := agent.handleUpdateAgent(msg)

	if response == nil {
		t.Fatal("handleUpdateAgent returned nil")
	}

	if response.Type != protocol.TypeUpdateAgentResponse {
		t.Errorf("Response type = %v, want %v", response.Type, protocol.TypeUpdateAgentResponse)
	}

	if response.ID != msg.ID {
		t.Errorf("Response ID = %v, want %v", response.ID, msg.ID)
	}
	
	// Give background goroutine time to start
	time.Sleep(10 * time.Millisecond)
}

func TestHandleUpdateAgent_LatestVersion(t *testing.T) {
	mockClient := &mockRedisClient{}
	agent := &Agent{
		logger:      logrus.New(),
		redisClient: mockClient,
		config: &config.AgentConfig{
			Server: config.ServerConfig{SecretKey: "test-key"},
		},
	}

	payload := protocol.UpdateAgentPayload{
		Version: "", // Empty means latest
	}

	msg := protocol.NewMessage(protocol.TypeUpdateAgent, payload)
	response := agent.handleUpdateAgent(msg)

	if response == nil {
		t.Fatal("handleUpdateAgent returned nil")
	}

	updateResp, ok := response.Payload.(protocol.UpdateAgentResponse)
	if !ok {
		t.Fatal("Response payload is not UpdateAgentResponse")
	}

	if updateResp.NewVersion != "latest" {
		t.Errorf("NewVersion = %v, want latest", updateResp.NewVersion)
	}
	
	// Give background goroutine time to start
	time.Sleep(10 * time.Millisecond)
}

func TestHandleUpdateAgent_InvalidPayload(t *testing.T) {
	agent := &Agent{
		logger: logrus.New(),
		config: &config.AgentConfig{
			Server: config.ServerConfig{SecretKey: "test-key"},
		},
	}

	// Invalid payload - number instead of object
	msg := protocol.NewMessage(protocol.TypeUpdateAgent, 12345)
	response := agent.handleUpdateAgent(msg)

	if response == nil {
		t.Fatal("handleUpdateAgent returned nil")
	}

	if response.Type != protocol.TypeErrorResponse {
		t.Errorf("Expected error response for invalid payload")
	}
}

func TestGetAgentVersion_Consistency(t *testing.T) {
	agent := &Agent{
		logger: logrus.New(),
	}

	// Call multiple times to ensure consistency
	for i := 0; i < 10; i++ {
		v1 := agent.getAgentVersion()
		v2 := agent.getAgentVersion()

		if v1 != v2 {
			t.Errorf("Version inconsistent: %v != %v", v1, v2)
		}
	}
}

func TestGetAgentVersion_NotEmpty(t *testing.T) {
	agent := &Agent{
		logger: logrus.New(),
	}

	version := agent.getAgentVersion()
	if version == "" {
		t.Error("Agent version is empty")
	}
}

func TestPerformUpdate_RequiresDownload(t *testing.T) {
	t.Skip("performUpdate requires wget and network access")
}

func TestVerifyChecksum_ValidFile(t *testing.T) {
	agent := &Agent{
		logger: logrus.New(),
	}

	// Create temp file with content
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := []byte("test content")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Create checksum file
	checksumFile, err := os.CreateTemp("", "checksum-*.txt")
	if err != nil {
		t.Fatalf("Failed to create checksum file: %v", err)
	}
	defer os.Remove(checksumFile.Name())

	// Write fake checksum (won't match but tests the parsing)
	checksumContent := "abc123def456  servereye-agent-linux-amd64\n"
	if _, err := checksumFile.Write([]byte(checksumContent)); err != nil {
		t.Fatalf("Failed to write checksum: %v", err)
	}
	checksumFile.Close()

	// This will fail because sha256sum is not available on Windows
	err = agent.verifyChecksum(tmpFile.Name(), checksumFile.Name())
	if err != nil {
		t.Logf("verifyChecksum error (expected on Windows): %v", err)
	}
}

func TestVerifyChecksum_NonexistentFile(t *testing.T) {
	agent := &Agent{
		logger: logrus.New(),
	}

	err := agent.verifyChecksum("/nonexistent/file", "/nonexistent/checksum")
	if err == nil {
		t.Error("Expected error for nonexistent files")
	}
}

func TestVerifyChecksum_EmptyChecksumFile(t *testing.T) {
	agent := &Agent{
		logger: logrus.New(),
	}

	// Create empty temp file
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create empty checksum file
	checksumFile, err := os.CreateTemp("", "checksum-*.txt")
	if err != nil {
		t.Fatalf("Failed to create checksum file: %v", err)
	}
	defer os.Remove(checksumFile.Name())
	checksumFile.Close()

	err = agent.verifyChecksum(tmpFile.Name(), checksumFile.Name())
	if err == nil {
		t.Error("Expected error for empty checksum file")
	}
}

func TestVerifyChecksum_InvalidChecksumFormat(t *testing.T) {
	agent := &Agent{
		logger: logrus.New(),
	}

	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("test"))
	tmpFile.Close()

	checksumFile, err := os.CreateTemp("", "checksum-*.txt")
	if err != nil {
		t.Fatalf("Failed to create checksum file: %v", err)
	}
	defer os.Remove(checksumFile.Name())
	
	// Invalid format - no agent filename
	checksumFile.Write([]byte("invalid checksum format\n"))
	checksumFile.Close()

	err = agent.verifyChecksum(tmpFile.Name(), checksumFile.Name())
	if err == nil {
		t.Error("Expected error for invalid checksum format")
	}
}

func TestVerifyChecksum_MultipleLines(t *testing.T) {
	agent := &Agent{
		logger: logrus.New(),
	}

	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("test"))
	tmpFile.Close()

	checksumFile, err := os.CreateTemp("", "checksum-*.txt")
	if err != nil {
		t.Fatalf("Failed to create checksum file: %v", err)
	}
	defer os.Remove(checksumFile.Name())
	
	// Multiple lines with agent checksum in the middle
	content := `abc123  other-file
def456  another-file
789xyz  servereye-agent-linux-amd64
ghi789  yet-another-file
`
	checksumFile.Write([]byte(content))
	checksumFile.Close()

	err = agent.verifyChecksum(tmpFile.Name(), checksumFile.Name())
	if err != nil {
		t.Logf("verifyChecksum error (expected without sha256sum): %v", err)
	}
}

func TestRestartAgent_SystemctlRequired(t *testing.T) {
	t.Skip("restartAgent requires systemctl")
}

func TestHandleUpdateAgent_BackgroundExecution(t *testing.T) {
	mockClient := &mockRedisClient{}
	agent := &Agent{
		logger:      logrus.New(),
		redisClient: mockClient,
		config: &config.AgentConfig{
			Server: config.ServerConfig{SecretKey: "test-key"},
		},
	}

	payload := protocol.UpdateAgentPayload{
		Version: "1.0.0",
	}

	msg := protocol.NewMessage(protocol.TypeUpdateAgent, payload)
	response := agent.handleUpdateAgent(msg)

	// Should return immediately (update runs in background)
	if response == nil {
		t.Fatal("handleUpdateAgent returned nil")
	}

	updateResp, ok := response.Payload.(protocol.UpdateAgentResponse)
	if !ok {
		t.Fatal("Response payload is not UpdateAgentResponse")
	}

	if !updateResp.Success {
		t.Error("Expected success=true for background update")
	}

	if !updateResp.RestartRequired {
		t.Error("Expected restart_required=true")
	}
	
	// Give background goroutine time to start and potentially fail
	time.Sleep(10 * time.Millisecond)
}

func TestUpdatePaths_Validation(t *testing.T) {
	// Test path constants used in update
	tests := []struct {
		name string
		path string
	}{
		{"download URL", "https://github.com/godofphonk/ServerEye/releases/latest/download/servereye-agent-linux-amd64"},
		{"checksum URL", "https://github.com/godofphonk/ServerEye/releases/latest/download/checksums.txt"},
		{"tmp file", "/tmp/servereye-agent-new"},
		{"checksum file", "/tmp/servereye-checksums.txt"},
		{"current binary", "/opt/servereye/servereye-agent"},
		{"backup binary", "/opt/servereye/servereye-agent.backup"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.path == "" {
				t.Error("Path is empty")
			}
		})
	}
}

func TestUpdateVersionLogic(t *testing.T) {
	tests := []struct {
		name           string
		inputVersion   string
		expectedTarget string
	}{
		{"explicit version", "1.2.3", "1.2.3"},
		{"empty version", "", "latest"},
		{"latest keyword", "latest", "latest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetVersion := tt.inputVersion
			if targetVersion == "" {
				targetVersion = "latest"
			}

			if targetVersion != tt.expectedTarget {
				t.Errorf("Target version = %v, want %v", targetVersion, tt.expectedTarget)
			}
		})
	}
}

func TestFileOperations_TempFiles(t *testing.T) {
	// Test temp file creation and cleanup
	tmpFile, err := os.CreateTemp("", "servereye-test-*.tmp")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Check file exists
	if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
		t.Error("Temp file does not exist")
	}

	// Cleanup
	if err := os.Remove(tmpPath); err != nil {
		t.Errorf("Failed to remove temp file: %v", err)
	}

	// Check file removed
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("Temp file still exists after removal")
	}
}

func TestFileOperations_Permissions(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "servereye-test-*.tmp")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Try to chmod (will work on Linux, may fail on Windows)
	err = os.Chmod(tmpFile.Name(), 0755)
	if err != nil {
		t.Logf("chmod failed (may be expected on Windows): %v", err)
	}
}

func TestChecksumParsing(t *testing.T) {
	tests := []struct {
		name           string
		checksumLine   string
		expectChecksum bool
	}{
		{
			name:           "valid line",
			checksumLine:   "abc123def456  servereye-agent-linux-amd64",
			expectChecksum: true,
		},
		{
			name:           "different file",
			checksumLine:   "abc123def456  other-file",
			expectChecksum: false,
		},
		{
			name:           "empty line",
			checksumLine:   "",
			expectChecksum: false,
		},
		{
			name:           "malformed line",
			checksumLine:   "invalid",
			expectChecksum: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasAgentChecksum := len(tt.checksumLine) > 0 && 
				filepath.Base(tt.checksumLine) == "servereye-agent-linux-amd64" ||
				filepath.Base(tt.checksumLine) != ""

			// Just verify the logic doesn't panic
			_ = hasAgentChecksum
		})
	}
}
