package metrics

import (
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
		Chipset:      "Unknown",
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

// CollectMemoryModules collects detailed memory modules information
func (h *HardwareInfoCollector) CollectMemoryModules() ([]types.MemoryModule, error) {
	h.logger.Debug("Collecting memory modules information")

	// Get memory devices from dmidecode
	cmd := exec.Command("sudo", "dmidecode", "-t", "17")
	output, err := cmd.Output()
	if err != nil {
		return []types.MemoryModule{}, err
	}

	// Parse memory devices
	memoryDevices := h.parseDMIMemoryDevices(string(output))

	return memoryDevices, nil
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

// parseDMIDecodeLine parses a single line from dmidecode output
func (h *HardwareInfoCollector) parseDMIDecodeLine(line string, currentDevice *types.MemoryModule) bool {
	if strings.Contains(line, "Size:") {
		if size := h.parseMemorySize(line); size > 0 {
			currentDevice.SizeGB = size
		}
	} else if strings.Contains(line, "Manufacturer:") {
		currentDevice.Manufacturer = strings.TrimSpace(strings.TrimPrefix(line, "Manufacturer:"))
	} else if strings.Contains(line, "Type:") {
		memType := strings.TrimSpace(strings.TrimPrefix(line, "Type:"))
		if memType != "Unknown" {
			currentDevice.MemoryType = memType
		}
	} else if strings.Contains(line, "Speed:") {
		if speed := h.parseMemorySpeed(line); speed > 0 {
			currentDevice.FrequencyMHz = speed
		}
	} else if strings.Contains(line, "Locator:") {
		slotName := strings.TrimSpace(strings.TrimPrefix(line, "Locator:"))
		currentDevice.SlotName = slotName
	} else if strings.Contains(line, "Part Number:") {
		partNumber := strings.TrimSpace(strings.TrimPrefix(line, "Part Number:"))
		if partNumber != "Not Specified" && partNumber != "Unknown" {
			currentDevice.PartNumber = partNumber
		}
	} else if strings.Contains(line, "Configured Memory Speed:") {
		if speed := h.parseMemorySpeed(line); speed > 0 {
			currentDevice.SpeedMTS = speed * 2 // Convert MHz to MT/s for DDR
		}
	} else if strings.Contains(line, "Voltage:") {
		if voltage := h.parseVoltage(line); voltage > 0 {
			currentDevice.Voltage = voltage
		}
	} else if strings.Contains(line, "Error Correction Type:") {
		ecc := strings.TrimSpace(strings.TrimPrefix(line, "Error Correction Type:"))
		currentDevice.ECC = ecc == "Multi-bit ECC" || ecc == "Single-bit ECC"
	} else if strings.Contains(line, "Type Detail:") {
		detail := strings.TrimSpace(strings.TrimPrefix(line, "Type Detail:"))
		currentDevice.Registered = strings.Contains(detail, "Registered")
	} else {
		return false // Line not processed
	}
	return true // Line processed
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

		h.parseDMIDecodeLine(line, &currentDevice)
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

// parseVoltage parses voltage from dmidecode output
func (h *HardwareInfoCollector) parseVoltage(line string) float64 {
	re := regexp.MustCompile(`([\d.]+)\s*V`)
	matches := re.FindStringSubmatch(line)
	if len(matches) == 2 {
		voltage, _ := strconv.ParseFloat(matches[1], 64)
		return voltage
	}
	return 0
}
