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

// GetCPUCores retrieves the number of physical CPU cores from /proc/cpuinfo
func (c *CPUMetrics) GetCPUCores() (int, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return 0, fmt.Errorf("failed to open /proc/cpuinfo: %w", err)
	}
	defer file.Close()

	processorIDs := make(map[int]bool)
	physicalIDs := make(map[int]bool)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "processor") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				var procID int
				_, err := fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &procID)
				if err == nil {
					processorIDs[procID] = true
				}
			}
		}

		if strings.HasPrefix(line, "physical id") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				var physID int
				_, err := fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &physID)
				if err == nil {
					physicalIDs[physID] = true
				}
			}
		}

		if strings.HasPrefix(line, "cpu cores") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				var cores int
				_, err := fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &cores)
				if err == nil {
					// Return cores per physical CPU
					return cores, nil
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading /proc/cpuinfo: %w", err)
	}

	// Fallback: if we couldn't get "cpu cores", return number of logical processors
	return len(processorIDs), nil
}

// GetCPUThreads retrieves the total number of CPU threads (logical processors)
func (c *CPUMetrics) GetCPUThreads() (int, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return 0, fmt.Errorf("failed to open /proc/cpuinfo: %w", err)
	}
	defer file.Close()

	threadCount := 0
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "processor") {
			threadCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading /proc/cpuinfo: %w", err)
	}

	if threadCount == 0 {
		return 0, fmt.Errorf("no processor information found in /proc/cpuinfo")
	}

	return threadCount, nil
}

// GetCPUFrequency retrieves CPU frequency in MHz (public method)
func (c *CPUMetrics) GetCPUFrequency() (float64, error) {
	return c.getCPUFrequency()
}
