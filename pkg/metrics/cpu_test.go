package metrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCPUMetrics_GetTemperature(t *testing.T) {
	cpu := NewCPUMetrics()

	// Test with mock temperature file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "temp")
	
	// Write mock temperature data (45.5°C in millidegrees)
	err := os.WriteFile(tempFile, []byte("45500\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create mock temperature file: %v", err)
	}

	// Test reading from mock file
	temp, err := cpu.readTemperatureFromFile(tempFile)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expectedTemp := 45.5
	if temp != expectedTemp {
		t.Errorf("Expected temperature %.1f, got %.1f", expectedTemp, temp)
	}
}

func TestCPUMetrics_readTemperatureFromFile_InvalidData(t *testing.T) {
	cpu := NewCPUMetrics()
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "temp")

	// Test with invalid data
	err := os.WriteFile(tempFile, []byte("invalid\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create mock temperature file: %v", err)
	}

	_, err = cpu.readTemperatureFromFile(tempFile)
	if err == nil {
		t.Error("Expected error for invalid temperature data, got nil")
	}
}

func TestCPUMetrics_readTemperatureFromFile_UnreasonableValue(t *testing.T) {
	cpu := NewCPUMetrics()
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "temp")

	// Test with unreasonable temperature (200°C)
	err := os.WriteFile(tempFile, []byte("200000\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create mock temperature file: %v", err)
	}

	_, err = cpu.readTemperatureFromFile(tempFile)
	if err == nil {
		t.Error("Expected error for unreasonable temperature, got nil")
	}
}

func TestCPUMetrics_readTemperatureFromFile_NonExistentFile(t *testing.T) {
	cpu := NewCPUMetrics()

	_, err := cpu.readTemperatureFromFile("/non/existent/file")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestCPUMetrics_GetSensorInfo(t *testing.T) {
	cpu := NewCPUMetrics()
	
	info := cpu.GetSensorInfo()
	if info == "" {
		t.Error("Expected sensor info, got empty string")
	}
}

func BenchmarkCPUMetrics_GetTemperature(b *testing.B) {
	cpu := NewCPUMetrics()
	
	// Create a mock temperature file for benchmarking
	tempDir := b.TempDir()
	tempFile := filepath.Join(tempDir, "temp")
	err := os.WriteFile(tempFile, []byte("45500\n"), 0644)
	if err != nil {
		b.Fatalf("Failed to create mock temperature file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cpu.readTemperatureFromFile(tempFile)
	}
}
