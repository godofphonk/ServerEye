package metrics

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/godofphonk/ServerEye/pkg/types"
	"github.com/sirupsen/logrus"
)

// HardwareInfoCollector collects detailed hardware information
type HardwareInfoCollector struct {
	logger *logrus.Logger
}

// NewHardwareInfoCollector creates a new hardware info collector
func NewHardwareInfoCollector(logger *logrus.Logger) *HardwareInfoCollector {
	return &HardwareInfoCollector{
		logger: logger,
	}
}

// CollectMotherboardInfo collects motherboard information
func (h *HardwareInfoCollector) CollectMotherboardInfo() (*types.MotherboardInfo, error) {
	h.logger.Debug("Collecting motherboard information")

	motherboard := &types.MotherboardInfo{
		Manufacturer: "Unknown",
		Model:        "Unknown",
	}

	// Try DMI (Linux)
	if runtime.GOOS == "linux" {
		if err := h.collectMotherboardFromDMI(motherboard); err != nil {
			h.logger.WithError(err).Warn("Failed to collect motherboard from DMI")
		}
	}

	// Fallback to dmidecode command
	if motherboard.Manufacturer == "Unknown" {
		if err := h.collectMotherboardFromDMIDecode(motherboard); err != nil {
			h.logger.WithError(err).Warn("Failed to collect motherboard from dmidecode")
		}
	}

	return motherboard, nil
}

// CollectMemoryInfo collects detailed memory information
func (h *HardwareInfoCollector) CollectMemoryInfo() (*types.MemoryInfo, error) {
	h.logger.Debug("Collecting memory information")

	memory := &types.MemoryInfo{
		MemoryType:  "Unknown",
		MemorySpeed: 0,
		SlotsTotal:  0,
		SlotsUsed:   0,
		Modules:     []types.MemoryModule{},
	}

	// Get total memory from /proc/meminfo
	if totalGB, err := h.getTotalMemory(); err == nil {
		memory.TotalMemoryGB = totalGB
	}

	// Collect detailed memory info
	if runtime.GOOS == "linux" {
		if err := h.collectMemoryFromDMI(memory); err != nil {
			h.logger.WithError(err).Warn("Failed to collect memory from DMI")
		}
	}

	// Fallback to dmidecode command
	if len(memory.Modules) == 0 {
		if err := h.collectMemoryFromDMIDecode(memory); err != nil {
			h.logger.WithError(err).Warn("Failed to collect memory from dmidecode")
		}
	}

	// If still no modules, create a generic one
	if len(memory.Modules) == 0 && memory.TotalMemoryGB > 0 {
		memory.Modules = append(memory.Modules, types.MemoryModule{
			Manufacturer: "Unknown",
			SizeGB:       uint64(memory.TotalMemoryGB),
			Speed:        memory.MemorySpeed,
			Type:         memory.MemoryType,
			Slot:         0,
		})
		memory.SlotsUsed = 1
		memory.SlotsTotal = 1
	}

	return memory, nil
}

// collectMotherboardFromDMI collects motherboard info from /sys/class/dmi
func (h *HardwareInfoCollector) collectMotherboardFromDMI(motherboard *types.MotherboardInfo) error {
	// Read board vendor
	if vendor, err := os.ReadFile("/sys/class/dmi/id/board_vendor"); err == nil {
		motherboard.Manufacturer = strings.TrimSpace(string(vendor))
	}

	// Read board name
	if name, err := os.ReadFile("/sys/class/dmi/id/board_name"); err == nil {
		motherboard.Model = strings.TrimSpace(string(name))
	}

	return nil
}

// collectMotherboardFromDMIDecode uses dmidecode command
func (h *HardwareInfoCollector) collectMotherboardFromDMIDecode(motherboard *types.MotherboardInfo) error {
	cmd := exec.Command("sudo", "dmidecode", "-t", "2")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Manufacturer:") {
			motherboard.Manufacturer = strings.TrimSpace(strings.TrimPrefix(line, "Manufacturer:"))
		} else if strings.HasPrefix(line, "Product Name:") {
			motherboard.Model = strings.TrimSpace(strings.TrimPrefix(line, "Product Name:"))
		}
	}

	return nil
}

// collectMemoryFromDMI collects memory info from /sys/class/dmi
func (h *HardwareInfoCollector) collectMemoryFromDMI(memory *types.MemoryInfo) error {
	// Try to get memory type from /sys/class/dmi/id/memory_device
	if memType, err := os.ReadFile("/sys/class/dmi/id/memory_device"); err == nil {
		memory.MemoryType = strings.TrimSpace(string(memType))
	}

	// Get memory speed
	if speed, err := os.ReadFile("/sys/class/dmi/id/memory_speed"); err == nil {
		if s, err := strconv.Atoi(strings.TrimSpace(string(speed))); err == nil {
			memory.MemorySpeed = s
		}
	}

	return nil
}

// collectMemoryFromDMIDecode uses dmidecode command for detailed memory info
func (h *HardwareInfoCollector) collectMemoryFromDMIDecode(memory *types.MemoryInfo) error {
	// First get memory array info (type 16)
	cmd := exec.Command("sudo", "dmidecode", "-t", "16")
	output, err := cmd.Output()
	if err == nil {
		h.parseMemoryArrayInfo(string(output), memory)
	}

	// Then get memory devices (type 17)
	cmd = exec.Command("sudo", "dmidecode", "-t", "17")
	output, err = cmd.Output()
	if err != nil {
		return err
	}

	// Parse memory devices
	memoryDevices := h.parseDMIMemoryDevices(string(output))

	for _, device := range memoryDevices {
		if device.SizeGB > 0 { // Only include populated slots
			memory.Modules = append(memory.Modules, device)
			memory.SlotsUsed++
		}
	}

	// Set memory type and speed from first module if available
	if len(memory.Modules) > 0 {
		if memory.MemoryType == "Unknown" {
			memory.MemoryType = memory.Modules[0].Type
		}
		if memory.MemorySpeed == 0 {
			memory.MemorySpeed = memory.Modules[0].Speed
		}
	}

	return nil
}

// parseMemoryArrayInfo parses memory array information from dmidecode type 16
func (h *HardwareInfoCollector) parseMemoryArrayInfo(output string, memory *types.MemoryInfo) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Number Of Devices:") {
			if slots := h.extractNumber(line); slots > 0 {
				memory.SlotsTotal = slots
			}
		}
	}
}

// extractNumber extracts number from string
func (h *HardwareInfoCollector) extractNumber(line string) int {
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) == 2 {
		num, _ := strconv.Atoi(matches[1])
		return num
	}
	return 0
}

// parseDMIMemoryDevices parses dmidecode output for memory devices
func (h *HardwareInfoCollector) parseDMIMemoryDevices(output string) []types.MemoryModule {
	var devices []types.MemoryModule
	var currentDevice types.MemoryModule
	var inDevice bool

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Memory Device") {
			if inDevice && currentDevice.SizeGB > 0 {
				devices = append(devices, currentDevice)
			}
			currentDevice = types.MemoryModule{}
			inDevice = true
			continue
		}

		if !inDevice {
			continue
		}

		if strings.Contains(line, "Size:") {
			if size := h.parseMemorySize(line); size > 0 {
				currentDevice.SizeGB = size
			}
		} else if strings.Contains(line, "Manufacturer:") {
			currentDevice.Manufacturer = strings.TrimSpace(strings.TrimPrefix(line, "Manufacturer:"))
		} else if strings.Contains(line, "Type:") {
			memType := strings.TrimSpace(strings.TrimPrefix(line, "Type:"))
			if memType != "Unknown" {
				currentDevice.Type = memType
			}
		} else if strings.Contains(line, "Speed:") {
			if speed := h.parseMemorySpeed(line); speed > 0 {
				currentDevice.Speed = speed
			}
		} else if strings.Contains(line, "Locator:") {
			// Extract slot number from locator
			if slot := h.extractSlotNumber(line); slot >= 0 {
				currentDevice.Slot = slot
			}
		}
	}

	// Add last device
	if inDevice && currentDevice.SizeGB > 0 {
		devices = append(devices, currentDevice)
	}

	return devices
}

// parseMemorySize parses size from dmidecode output
func (h *HardwareInfoCollector) parseMemorySize(line string) uint64 {
	re := regexp.MustCompile(`(\d+)\s*(MB|GB|TB)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) == 3 {
		size, _ := strconv.ParseUint(matches[1], 10, 64)
		unit := strings.ToUpper(matches[2])

		switch unit {
		case "MB":
			return size / 1024
		case "GB":
			return size
		case "TB":
			return size * 1024
		}
	}
	return 0
}

// parseMemorySpeed parses speed from dmidecode output
func (h *HardwareInfoCollector) parseMemorySpeed(line string) int {
	// Handle both "MHz" and "MT/s"
	re := regexp.MustCompile(`(\d+)\s*(MHz|MT/s)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) == 3 {
		speed, _ := strconv.Atoi(matches[1])
		return speed
	}
	return 0
}

// extractSlotNumber extracts slot number from locator string
func (h *HardwareInfoCollector) extractSlotNumber(line string) int {
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) == 2 {
		slot, _ := strconv.Atoi(matches[1])
		return slot
	}
	return -1
}

// getTotalMemory gets total memory from /proc/meminfo
func (h *HardwareInfoCollector) getTotalMemory() (float64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseUint(fields[1], 10, 64)
				if err == nil {
					return float64(kb) / 1024 / 1024, nil // Convert to GB
				}
			}
		}
	}

	return 0, fmt.Errorf("MemTotal not found in /proc/meminfo")
}
