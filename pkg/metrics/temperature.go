package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// TemperatureInfo represents detailed temperature information
type TemperatureInfo struct {
	CPUTemperature      float64              `json:"cpu_temperature"`
	GPUTemperature      float64              `json:"gpu_temperature"`
	SystemTemperature   float64              `json:"system_temperature"`
	StorageTemperatures []StorageTemperature `json:"storage_temperatures"`
	HighestTemperature  float64              `json:"highest_temperature"`
	TemperatureUnit     string               `json:"temperature_unit"`
}

// StorageTemperature represents storage device temperature
type StorageTemperature struct {
	Device      string  `json:"device"`
	Type        string  `json:"type"`
	Temperature float64 `json:"temperature"`
}

// TemperatureMetrics represents temperature metrics collector
type TemperatureMetrics struct {
	logger *logrus.Logger
}

// NewTemperatureMetrics creates a new temperature metrics collector
func NewTemperatureMetrics(logger *logrus.Logger) *TemperatureMetrics {
	return &TemperatureMetrics{
		logger: logger,
	}
}

// GetTemperatureInfo collects detailed temperature information
func (tm *TemperatureMetrics) GetTemperatureInfo() (*TemperatureInfo, error) {
	// Platform check - only Linux supported for now
	if runtime.GOOS != "linux" {
		tm.logger.Warnf("Temperature metrics not implemented for %s, returning zero values", runtime.GOOS)
		return &TemperatureInfo{
			CPUTemperature:      0,
			GPUTemperature:      0,
			SystemTemperature:   0,
			HighestTemperature:  0,
			TemperatureUnit:     "celsius",
			StorageTemperatures: []StorageTemperature{},
		}, nil
	}

	tempInfo := &TemperatureInfo{
		TemperatureUnit:     "celsius",
		StorageTemperatures: []StorageTemperature{},
	}

	// Get CPU temperature
	if cpuTemp, err := tm.getCPUTemperature(); err == nil {
		tempInfo.CPUTemperature = cpuTemp
		tempInfo.HighestTemperature = cpuTemp
	} else {
		tm.logger.WithError(err).Debug("Failed to get CPU temperature")
	}

	// Get GPU temperature
	if gpuTemp, err := tm.getGPUTemperature(); err == nil {
		tempInfo.GPUTemperature = gpuTemp
		if gpuTemp > tempInfo.HighestTemperature {
			tempInfo.HighestTemperature = gpuTemp
		}
	} else {
		tm.logger.WithError(err).Debug("Failed to get GPU temperature")
	}

	// Get System temperature
	if sysTemp, err := tm.getSystemTemperature(); err == nil {
		tempInfo.SystemTemperature = sysTemp
		if sysTemp > tempInfo.HighestTemperature {
			tempInfo.HighestTemperature = sysTemp
		}
	} else {
		tm.logger.WithError(err).Debug("Failed to get system temperature")
	}

	// Get storage temperatures
	if storageTemps, err := tm.getStorageTemperatures(); err == nil {
		tempInfo.StorageTemperatures = storageTemps
		for _, storage := range storageTemps {
			if storage.Temperature > tempInfo.HighestTemperature {
				tempInfo.HighestTemperature = storage.Temperature
			}
		}
	} else {
		tm.logger.WithError(err).Debug("Failed to get storage temperatures")
	}

	tm.logger.WithFields(map[string]interface{}{
		"cpu_temp":      tempInfo.CPUTemperature,
		"gpu_temp":      tempInfo.GPUTemperature,
		"system_temp":   tempInfo.SystemTemperature,
		"storage_count": len(tempInfo.StorageTemperatures),
		"highest_temp":  tempInfo.HighestTemperature,
	}).Debug("Temperature metrics collected")

	return tempInfo, nil
}

// getCPUTemperature gets CPU temperature from thermal zones
func (tm *TemperatureMetrics) getCPUTemperature() (float64, error) {
	// Try thermal zones first
	if temp, err := tm.getThermalZoneTemp(); err == nil {
		return temp, nil
	}

	// Try hwmon sensors directly
	if temp, err := tm.getHwmonCPUTemperature(); err == nil {
		return temp, nil
	}

	// Try sensors command as fallback
	if temp, err := tm.getSensorsTemperature("cpu"); err == nil {
		return temp, nil
	}

	// Try k10temp specifically for AMD CPUs
	if temp, err := tm.getSensorsTemperature("k10temp"); err == nil {
		return temp, nil
	}

	return 0, fmt.Errorf("could not read CPU temperature")
}

// getThermalZoneTemp reads from /sys/class/thermal/thermal_zone*/
func (tm *TemperatureMetrics) getThermalZoneTemp() (float64, error) {
	// Try to find CPU thermal zone
	thermalZones := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/thermal/thermal_zone1/temp",
		"/sys/class/thermal/thermal_zone2/temp",
	}

	for _, zone := range thermalZones {
		if data, err := os.ReadFile(zone); err == nil {
			tempStr := strings.TrimSpace(string(data))
			if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
				// Convert from millidegrees to degrees
				return temp / 1000, nil
			}
		}
	}

	return 0, fmt.Errorf("no thermal zone found")
}

// getHwmonCPUTemperature reads CPU temperature from hwmon devices
func (tm *TemperatureMetrics) getHwmonCPUTemperature() (float64, error) {
	// Try common hwmon paths for CPU temperature
	hwmonPaths := []string{
		"/sys/devices/pci0000:00/0000:00:18.3/hwmon/hwmon1/temp1_input", // AMD CPU
		"/sys/class/hwmon/hwmon*/temp1_input",                           // Generic
		"/sys/devices/platform/coretemp.0/hwmon/hwmon*/temp*_input",     // Intel CPU
	}

	for _, path := range hwmonPaths {
		matches, err := filepath.Glob(path)
		if err == nil && len(matches) > 0 {
			for _, match := range matches {
				if data, err := os.ReadFile(match); err == nil {
					tempStr := strings.TrimSpace(string(data))
					if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
						// Convert from millidegrees to degrees
						return temp / 1000, nil
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("no CPU hwmon temperature found")
}

// getGPUTemperature gets GPU temperature
func (tm *TemperatureMetrics) getGPUTemperature() (float64, error) {
	// Try NVIDIA GPU
	if temp, err := tm.getNVIDIAGPUTemperature(); err == nil {
		return temp, nil
	}

	// Try AMD GPU
	if temp, err := tm.getAMDGPUTemperature(); err == nil {
		return temp, nil
	}

	// Try Intel GPU
	if temp, err := tm.getIntelGPUTemperature(); err == nil {
		return temp, nil
	}

	return 0, fmt.Errorf("no GPU temperature found")
}

// getNVIDIAGPUTemperature gets NVIDIA GPU temperature
func (tm *TemperatureMetrics) getNVIDIAGPUTemperature() (float64, error) {
	cmd := exec.Command("nvidia-smi", "--query-gpu=temperature.gpu", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	tempStr := strings.TrimSpace(string(output))
	temp, err := strconv.ParseFloat(tempStr, 64)
	if err != nil {
		return 0, err
	}

	return temp, nil
}

// getAMDGPUTemperature gets AMD GPU temperature
func (tm *TemperatureMetrics) getAMDGPUTemperature() (float64, error) {
	cmd := exec.Command("aticonfig", "--od-gettemperature")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// Parse aticonfig output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Temperature") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "Temperature" && i+2 < len(fields) {
					tempStr := strings.TrimSuffix(fields[i+2], "C")
					return strconv.ParseFloat(tempStr, 64)
				}
			}
		}
	}

	return 0, fmt.Errorf("could not parse AMD GPU temperature")
}

// getIntelGPUTemperature gets Intel GPU temperature
func (tm *TemperatureMetrics) getIntelGPUTemperature() (float64, error) {
	cmd := exec.Command("intel_gpu_top", "-J")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// Parse JSON output from intel_gpu_top
	var data map[string]interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		return 0, err
	}

	if gpuTemp, ok := data["GPU temperature"].(float64); ok {
		return gpuTemp, nil
	}

	return 0, fmt.Errorf("could not parse Intel GPU temperature")
}

// getSystemTemperature gets system temperature from sensors
func (tm *TemperatureMetrics) getSystemTemperature() (float64, error) {
	return tm.getSensorsTemperature("sys")
}

// getSensorsTemperature gets temperature from lm-sensors
func (tm *TemperatureMetrics) getSensorsTemperature(prefix string) (float64, error) {
	cmd := exec.Command("sensors", "-j")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		return 0, err
	}

	// Parse sensors output for temperature
	for chipName, chipData := range data {
		if chip, ok := chipData.(map[string]interface{}); ok {
			// Look for temperature readings
			for key, value := range chip {
				if strings.Contains(key, "temp") && strings.Contains(key, "input") {
					if temp, ok := value.(float64); ok {
						// Check if this matches our prefix
						if strings.Contains(strings.ToLower(chipName), prefix) {
							return temp, nil
						}
					}
				}
				// Special handling for k10temp Tctl
				if prefix == "k10temp" && strings.Contains(strings.ToLower(chipName), "k10temp") {
					if key == "Tctl" {
						if temp, ok := value.(float64); ok {
							return temp, nil
						}
					}
				}
				// Special handling for CPU sensors
				if prefix == "cpu" && (strings.Contains(strings.ToLower(chipName), "coretemp") ||
					strings.Contains(strings.ToLower(chipName), "k10temp")) {
					if key == "Tctl" || key == "Package id" || strings.Contains(key, "temp") {
						if temp, ok := value.(float64); ok {
							return temp, nil
						}
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("no %s temperature found in sensors", prefix)
}

// getStorageTemperatures gets storage device temperatures
func (tm *TemperatureMetrics) getStorageTemperatures() ([]StorageTemperature, error) {
	var storageTemps []StorageTemperature

	// Get NVMe temperatures
	if nvmeTemps, err := tm.getNVMeTemperatures(); err == nil {
		storageTemps = append(storageTemps, nvmeTemps...)
	}

	// Get HDD/SSD temperatures via smartctl
	if smartTemps, err := tm.getSmartctlTemperatures(); err == nil {
		storageTemps = append(storageTemps, smartTemps...)
	}

	return storageTemps, nil
}

// getNVMeTemperatures gets NVMe drive temperatures
func (tm *TemperatureMetrics) getNVMeTemperatures() ([]StorageTemperature, error) {
	var temps []StorageTemperature

	// Find NVMe devices
	cmd := exec.Command("lsblk", "-d", "-o", "NAME,ROTA")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines[1:] { // Skip header
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "0" { // Non-rotating = SSD/NVMe
			device := fields[0]
			if strings.HasPrefix(device, "nvme") {
				// Get NVMe temperature
				cmd := exec.Command("smartctl", "-A", "/dev/"+device)
				output, err := cmd.Output()
				if err != nil {
					continue
				}

				lines := strings.Split(string(output), "\n")
				for _, line := range lines {
					if strings.Contains(line, "Temperature:") {
						fields := strings.Fields(line)
						for i, field := range fields {
							if field == "Temperature:" && i+1 < len(fields) {
								tempStr := strings.TrimSuffix(fields[i+1], "C")
								if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
									temps = append(temps, StorageTemperature{
										Device:      "/dev/" + device,
										Type:        "NVMe",
										Temperature: temp,
									})
								}
								break
							}
						}
					}
				}
			}
		}
	}

	return temps, nil
}

// getSmartctlTemperatures gets HDD/SSD temperatures via smartctl
func (tm *TemperatureMetrics) getSmartctlTemperatures() ([]StorageTemperature, error) {
	var temps []StorageTemperature

	// Find block devices
	cmd := exec.Command("lsblk", "-d", "-o", "NAME")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines[1:] { // Skip header
		device := strings.TrimSpace(line)
		if device != "" {
			// Try to get temperature via smartctl
			cmd := exec.Command("smartctl", "-A", "/dev/"+device)
			output, err := cmd.Output()
			if err != nil {
				continue
			}

			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Temperature") || strings.Contains(line, "temp") {
					fields := strings.Fields(line)
					for i, field := range fields {
						if (strings.Contains(field, "Temperature") || strings.Contains(field, "temp")) && i+1 < len(fields) {
							tempStr := strings.TrimSuffix(fields[i+1], "C")
							if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
								deviceType := "SSD"
								if strings.Contains(strings.ToLower(line), "spin") || strings.Contains(strings.ToLower(line), "rotation") {
									deviceType = "HDD"
								}

								temps = append(temps, StorageTemperature{
									Device:      "/dev/" + device,
									Type:        deviceType,
									Temperature: temp,
								})
							}
							break
						}
					}
				}
			}
		}
	}

	return temps, nil
}
