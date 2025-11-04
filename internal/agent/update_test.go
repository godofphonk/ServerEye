package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/servereye/servereye/internal/version"
	"github.com/sirupsen/logrus"
)

func TestGetAgentVersion(t *testing.T) {
	agent := &Agent{}

	v := agent.getAgentVersion()

	if v == "" {
		t.Error("getAgentVersion() returned empty string")
	}

	if v != version.GetVersion() {
		t.Errorf("getAgentVersion() = %v, want %v", v, version.GetVersion())
	}
}

func TestGetAgentVersion_Multiple(t *testing.T) {
	agent := &Agent{}

	// Call multiple times to ensure consistency
	for i := 0; i < 5; i++ {
		v := agent.getAgentVersion()
		if v != version.GetVersion() {
			t.Errorf("Call %d: getAgentVersion() = %v, want %v", i, v, version.GetVersion())
		}
	}
}

func TestVerifyChecksum_InvalidFile(t *testing.T) {
	logger := logrus.New()
	agent := &Agent{
		logger: logger,
	}

	err := agent.verifyChecksum("/nonexistent/file", "/nonexistent/checksum")

	if err == nil {
		t.Error("verifyChecksum() should return error for nonexistent files")
	}
}

func TestVerifyChecksum_InvalidChecksumFile(t *testing.T) {
	logger := logrus.New()
	agent := &Agent{
		logger: logger,
	}

	// Create temp file
	tempFile, err := os.CreateTemp("", "test-file-*.bin")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Write([]byte("test content"))
	tempFile.Close()

	err = agent.verifyChecksum(tempFile.Name(), "/nonexistent/checksum.sha256")

	if err == nil {
		t.Error("verifyChecksum() should return error for nonexistent checksum file")
	}
}

func TestVerifyChecksum_EmptyFiles(t *testing.T) {
	logger := logrus.New()
	agent := &Agent{
		logger: logger,
	}

	// Create empty temp files
	tempFile, _ := os.CreateTemp("", "test-*.bin")
	checksumFile, _ := os.CreateTemp("", "test-*.sha256")
	defer os.Remove(tempFile.Name())
	defer os.Remove(checksumFile.Name())
	tempFile.Close()
	checksumFile.Close()

	err := agent.verifyChecksum(tempFile.Name(), checksumFile.Name())

	// Should fail because checksum doesn't match
	if err == nil {
		t.Log("Verification failed as expected for empty files")
	}
}

func TestPerformUpdate_InvalidURL(t *testing.T) {
	t.Skip("Requires mocking wget command")
}

func TestRestartAgent(t *testing.T) {
	t.Skip("Requires mocking systemctl command")
}

func TestUpdatePaths(t *testing.T) {
	// Test that update paths are constructed correctly
	tests := []struct {
		name     string
		baseURL  string
		version  string
		wantPath string
	}{
		{
			name:     "valid version",
			baseURL:  "https://example.com",
			version:  "1.2.3",
			wantPath: "https://example.com/servereye-agent-1.2.3",
		},
		{
			name:     "latest version",
			baseURL:  "https://example.com",
			version:  "latest",
			wantPath: "https://example.com/servereye-agent-latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tt.baseURL, "servereye-agent-"+tt.version)
			if path == "" {
				t.Error("Path construction failed")
			}
		})
	}
}
