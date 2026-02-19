package metrics

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// GetCPUModel retrieves the CPU model name from /proc/cpuinfo
func (c *CPUMetrics) GetCPUModel() (string, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "", fmt.Errorf("failed to open /proc/cpuinfo: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			// Format: "model name	: Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz"
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading /proc/cpuinfo: %w", err)
	}

	return "", fmt.Errorf("CPU model not found in /proc/cpuinfo")
}

// GetCPUFrequency retrieves CPU frequency in MHz (public method)
func (c *CPUMetrics) GetCPUFrequency() (float64, error) {
	return c.getCPUFrequency()
}
