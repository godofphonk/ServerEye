package metrics

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestCPUMetrics_GetDetailedUsage(t *testing.T) {
	cpu := NewCPUMetrics()

	// Test GetDetailedUsage method
	cpuUsage, err := cpu.GetDetailedUsage()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if cpuUsage == nil {
		t.Fatal("Expected CPU usage info, got nil")
	}

	// Check that cores are set correctly
	expectedCores := runtime.NumCPU()
	if cpuUsage.Cores != expectedCores {
		t.Errorf("Expected %d cores, got %d", expectedCores, cpuUsage.Cores)
	}

	// Check that usage values are reasonable (0-100%)
	if cpuUsage.UsageTotal < 0 || cpuUsage.UsageTotal > 100 {
		t.Errorf("UsageTotal should be between 0 and 100, got %.2f", cpuUsage.UsageTotal)
	}

	if cpuUsage.UsageUser < 0 || cpuUsage.UsageUser > 100 {
		t.Errorf("UsageUser should be between 0 and 100, got %.2f", cpuUsage.UsageUser)
	}

	if cpuUsage.UsageSystem < 0 || cpuUsage.UsageSystem > 100 {
		t.Errorf("UsageSystem should be between 0 and 100, got %.2f", cpuUsage.UsageSystem)
	}

	if cpuUsage.UsageIdle < 0 || cpuUsage.UsageIdle > 100 {
		t.Errorf("UsageIdle should be between 0 and 100, got %.2f", cpuUsage.UsageIdle)
	}

	// Check load averages if available
	if cpuUsage.LoadAverage != nil {
		if cpuUsage.LoadAverage.Load1Min < 0 {
			t.Errorf("Load1Min should be non-negative, got %.2f", cpuUsage.LoadAverage.Load1Min)
		}
		if cpuUsage.LoadAverage.Load5Min < 0 {
			t.Errorf("Load5Min should be non-negative, got %.2f", cpuUsage.LoadAverage.Load5Min)
		}
		if cpuUsage.LoadAverage.Load15Min < 0 {
			t.Errorf("Load15Min should be non-negative, got %.2f", cpuUsage.LoadAverage.Load15Min)
		}
	}

	// Check frequency if available
	if cpuUsage.Frequency > 0 {
		// Should be reasonable CPU frequency (100MHz to 10GHz)
		if cpuUsage.Frequency < 100 || cpuUsage.Frequency > 10000 {
			t.Errorf("Frequency should be between 100 and 10000 MHz, got %.2f", cpuUsage.Frequency)
		}
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

func TestCPUMetrics_readFrequencyFromFile(t *testing.T) {
	cpu := NewCPUMetrics()
	tempDir := t.TempDir()
	freqFile := filepath.Join(tempDir, "frequency")

	// Test with valid frequency data (2400 MHz in KHz)
	err := os.WriteFile(freqFile, []byte("2400000\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create mock frequency file: %v", err)
	}

	freq, err := cpu.readFrequencyFromFile(freqFile)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expectedFreq := 2400.0
	if freq != expectedFreq {
		t.Errorf("Expected frequency %.1f MHz, got %.1f MHz", expectedFreq, freq)
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

func BenchmarkCPUMetrics_GetDetailedUsage(b *testing.B) {
	cpu := NewCPUMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cpu.GetDetailedUsage()
	}
}
