package metrics

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// CPUMetrics provides methods for collecting CPU metrics
type CPUMetrics struct{}

// CPUUsageInfo represents detailed CPU usage statistics
type CPUUsageInfo struct {
	UsageTotal  float64          `json:"usage_total"`
	UsageUser   float64          `json:"usage_user"`
	UsageSystem float64          `json:"usage_system"`
	UsageIdle   float64          `json:"usage_idle"`
	LoadAverage *LoadAverageInfo `json:"load_average"`
	Cores       int              `json:"cores"`
	Frequency   float64          `json:"frequency"`
}

// LoadAverageInfo represents system load averages
type LoadAverageInfo struct {
	Load1Min  float64 `json:"load_1min"`
	Load5Min  float64 `json:"load_5min"`
	Load15Min float64 `json:"load_15min"`
}

// NewCPUMetrics creates a new CPUMetrics instance
func NewCPUMetrics() *CPUMetrics {
	return &CPUMetrics{}
}

// GetTemperature retrieves CPU temperature in Celsius
func (c *CPUMetrics) GetTemperature() (float64, error) {
	// Try different CPU temperature sources
	sources := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/hwmon/hwmon0/temp1_input",
		"/sys/class/hwmon/hwmon1/temp1_input",
		"/sys/class/hwmon/hwmon2/temp1_input",
	}

	for _, source := range sources {
		if temp, err := c.readTemperatureFromFile(source); err == nil {
			return temp, nil
		}
	}

	// If sysfs temperature reading failed, try other methods
	if temp, err := c.getTemperatureFromCoretemp(); err == nil {
		return temp, nil
	}

	return 0, fmt.Errorf("failed to get CPU temperature: sensors unavailable")
}

// readTemperatureFromFile reads temperature from system file
func (c *CPUMetrics) readTemperatureFromFile(filepath string) (float64, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, fmt.Errorf("failed to read data from %s", filepath)
	}

	tempStr := strings.TrimSpace(scanner.Text())
	tempMilliC, err := strconv.ParseFloat(tempStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse temperature: %v", err)
	}

	// Convert from millidegrees to degrees Celsius
	tempC := tempMilliC / 1000.0

	// Validate temperature range (-50 to 150 degrees)
	if tempC < -50 || tempC > 150 {
		return 0, fmt.Errorf("unreasonable temperature value: %.2f°C", tempC)
	}

	return tempC, nil
}

// getTemperatureFromCoretemp attempts to get temperature via coretemp
func (c *CPUMetrics) getTemperatureFromCoretemp() (float64, error) {
	// Look for coretemp sensors
	coretempPaths := []string{
		"/sys/devices/platform/coretemp.0/hwmon/hwmon*/temp1_input",
		"/sys/devices/platform/coretemp.0/temp1_input",
	}

	for _, pattern := range coretempPaths {
		// Simple implementation without glob - check several variants
		for i := 0; i < 10; i++ {
			path := strings.Replace(pattern, "*", fmt.Sprintf("%d", i), 1)
			if temp, err := c.readTemperatureFromFile(path); err == nil {
				return temp, nil
			}
		}
	}

	return 0, fmt.Errorf("coretemp sensors not found")
}

// GetSensorInfo returns information about available sensors
func (c *CPUMetrics) GetSensorInfo() string {
	sources := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/hwmon/hwmon0/temp1_input",
		"/sys/class/hwmon/hwmon1/temp1_input",
	}

	for _, source := range sources {
		if _, err := os.Stat(source); err == nil {
			return source
		}
	}

	return "unknown"
}

// GetDetailedUsage retrieves detailed CPU usage statistics
func (c *CPUMetrics) GetDetailedUsage() (*CPUUsageInfo, error) {
	cpuInfo := &CPUUsageInfo{
		Cores: runtime.NumCPU(),
	}

	// Get CPU usage from /proc/stat
	if usage, err := c.getCPUUsageFromProc(); err == nil {
		cpuInfo.UsageTotal = usage.UsageTotal
		cpuInfo.UsageUser = usage.UsageUser
		cpuInfo.UsageSystem = usage.UsageSystem
		cpuInfo.UsageIdle = usage.UsageIdle
	}

	// Get load average
	if loadAvg, err := c.getLoadAverage(); err == nil {
		cpuInfo.LoadAverage = loadAvg
	}

	// Get CPU frequency
	if freq, err := c.getCPUFrequency(); err == nil {
		cpuInfo.Frequency = freq
	}

	return cpuInfo, nil
}

// getCPUUsageFromProc parses /proc/stat for CPU usage
func (c *CPUMetrics) getCPUUsageFromProc() (*CPUUsageInfo, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 8 {
				continue
			}

			// Parse CPU times: user, nice, system, idle, iowait, irq, softirq, steal
			user, _ := strconv.ParseFloat(fields[1], 64)
			nice, _ := strconv.ParseFloat(fields[2], 64)
			system, _ := strconv.ParseFloat(fields[3], 64)
			idle, _ := strconv.ParseFloat(fields[4], 64)
			iowait, _ := strconv.ParseFloat(fields[5], 64)
			irq, _ := strconv.ParseFloat(fields[6], 64)
			softirq, _ := strconv.ParseFloat(fields[7], 64)
			steal, _ := strconv.ParseFloat(fields[8], 64)

			total := user + nice + system + idle + iowait + irq + softirq + steal
			used := user + nice + system + iowait + irq + softirq + steal

			usageTotal := 0.0
			if total > 0 {
				usageTotal = (used / total) * 100
			}

			return &CPUUsageInfo{
				UsageTotal:  usageTotal,
				UsageUser:   (user / total) * 100,
				UsageSystem: (system / total) * 100,
				UsageIdle:   (idle / total) * 100,
			}, nil
		}
	}

	return nil, fmt.Errorf("cpu stats not found in /proc/stat")
}

// getLoadAverage retrieves system load averages
func (c *CPUMetrics) getLoadAverage() (*LoadAverageInfo, error) {
	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 {
			load1, _ := strconv.ParseFloat(fields[0], 64)
			load5, _ := strconv.ParseFloat(fields[1], 64)
			load15, _ := strconv.ParseFloat(fields[2], 64)

			return &LoadAverageInfo{
				Load1Min:  load1,
				Load5Min:  load5,
				Load15Min: load15,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to parse load average")
}

// getCPUFrequency retrieves CPU frequency in MHz
func (c *CPUMetrics) getCPUFrequency() (float64, error) {
	// Try to read from /proc/cpuinfo
	cmd := exec.Command("cat", "/proc/cpuinfo")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "cpu MHz") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				freq, err := strconv.ParseFloat(fields[2], 64)
				if err == nil {
					return freq, nil
				}
			}
		}
	}

	// Fallback: try reading from sysfs
	if freq, err := c.getFrequencyFromSysfs(); err == nil {
		return freq, nil
	}

	return 0, fmt.Errorf("CPU frequency not found")
}

// getFrequencyFromSysfs tries to get CPU frequency from sysfs
func (c *CPUMetrics) getFrequencyFromSysfs() (float64, error) {
	// Try to read from cpuinfo_cur_freq on first CPU
	freqPath := "/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_cur_freq"
	if freq, err := c.readFrequencyFromFile(freqPath); err == nil {
		return freq, nil
	}

	// Try base_freq as fallback
	baseFreqPath := "/sys/devices/system/cpu/cpu0/cpufreq/base_freq"
	if freq, err := c.readFrequencyFromFile(baseFreqPath); err == nil {
		return freq, nil
	}

	return 0, fmt.Errorf("sysfs frequency not available")
}

// readFrequencyFromFile reads frequency from sysfs file (in KHz, converts to MHz)
func (c *CPUMetrics) readFrequencyFromFile(filepath string) (float64, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		freqKHz, err := strconv.ParseFloat(strings.TrimSpace(scanner.Text()), 64)
		if err != nil {
			return 0, err
		}
		// Convert from KHz to MHz
		return freqKHz / 1000.0, nil
	}

	return 0, fmt.Errorf("failed to read frequency from %s", filepath)
}
