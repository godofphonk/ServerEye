package metrics

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// CPUMetrics provides methods for collecting CPU metrics
type CPUMetrics struct{}

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
