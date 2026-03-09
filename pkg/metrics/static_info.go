package metrics

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/godofphonk/ServerEye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// StaticInfoCollector collects static server information
type StaticInfoCollector struct {
	logger *logrus.Logger
}

// NewStaticInfoCollector creates a new static info collector
func NewStaticInfoCollector(logger *logrus.Logger) *StaticInfoCollector {
	return &StaticInfoCollector{
		logger: logger,
	}
}

// CollectStaticInfo collects all static server information
func (s *StaticInfoCollector) CollectStaticInfo() (*protocol.StaticServerInfo, error) {
	staticInfo := &protocol.StaticServerInfo{
		ServerInfo:        s.collectServerInfo(),
		HardwareInfo:      s.collectHardwareInfo(),
		NetworkInterfaces: s.collectNetworkInterfaces(),
		DiskInfo:          s.collectDiskInfo(),
	}

	return staticInfo, nil
}

// collectServerInfo collects basic server information
func (s *StaticInfoCollector) collectServerInfo() protocol.ServerInfo {
	hostname, _ := os.Hostname()

	osName, osVersion := s.getOSInfo()
	kernel := s.getKernelVersion()

	return protocol.ServerInfo{
		Hostname:     hostname,
		OS:           osName,
		OSVersion:    osVersion,
		Kernel:       kernel,
		Architecture: runtime.GOARCH,
	}
}

// getOSInfo retrieves OS name and version
func (s *StaticInfoCollector) getOSInfo() (string, string) {
	if runtime.GOOS != "linux" {
		return runtime.GOOS, "unknown"
	}

	// Read /etc/os-release
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "Linux", "unknown"
	}
	defer file.Close()

	var osName, osVersion string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "NAME=") {
			osName = strings.Trim(strings.TrimPrefix(line, "NAME="), "\"")
		} else if strings.HasPrefix(line, "VERSION_ID=") {
			osVersion = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
	}

	if osName == "" {
		osName = "Linux"
	}
	if osVersion == "" {
		osVersion = "unknown"
	}

	return osName, osVersion
}

// getKernelVersion retrieves kernel version
func (s *StaticInfoCollector) getKernelVersion() string {
	if runtime.GOOS != "linux" {
		return "unknown"
	}

	cmd := exec.Command("uname", "-r")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(output))
}

// collectHardwareInfo collects hardware specifications
func (s *StaticInfoCollector) collectHardwareInfo() protocol.HardwareInfo {
	hwInfo := protocol.HardwareInfo{
		CPUModel:        s.getCPUModel(),
		CPUCores:        s.getCPUCores(),
		CPUThreads:      runtime.NumCPU(),
		CPUFrequencyMHz: s.getCPUFrequency(),
		TotalMemoryGB:   s.getTotalMemory(),
	}

	// Try to get GPU info
	gpuModel, gpuDriver, gpuMemory := s.getGPUInfo()
	if gpuModel != "" {
		hwInfo.GPUModel = gpuModel
		hwInfo.GPUDriver = gpuDriver
		hwInfo.GPUMemoryGB = gpuMemory
	}

	return hwInfo
}

// getCPUModel retrieves CPU model name
func (s *StaticInfoCollector) getCPUModel() string {
	if runtime.GOOS != "linux" {
		return "unknown"
	}

	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return "unknown"
}

// getCPUCores retrieves physical CPU cores count
func (s *StaticInfoCollector) getCPUCores() int {
	if runtime.GOOS != "linux" {
		return runtime.NumCPU()
	}

	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return runtime.NumCPU()
	}
	defer file.Close()

	coreIDs := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "core id") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				coreIDs[strings.TrimSpace(parts[1])] = true
			}
		}
	}

	if len(coreIDs) > 0 {
		return len(coreIDs)
	}

	return runtime.NumCPU()
}

// getCPUFrequency retrieves CPU frequency in MHz
func (s *StaticInfoCollector) getCPUFrequency() float64 {
	if runtime.GOOS != "linux" {
		return 0
	}

	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu MHz") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				freq, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
				if err == nil {
					return freq
				}
			}
		}
	}

	return 0
}

// getTotalMemory retrieves total system memory in GB
func (s *StaticInfoCollector) getTotalMemory() float64 {
	if runtime.GOOS != "linux" {
		return 0
	}

	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				kb, err := strconv.ParseUint(parts[1], 10, 64)
				if err == nil {
					return float64(kb) / 1024 / 1024 // Convert KB to GB
				}
			}
		}
	}

	return 0
}

// getGPUInfo retrieves GPU information using nvidia-smi
func (s *StaticInfoCollector) getGPUInfo() (model string, driver string, memoryGB float64) {
	// Try nvidia-smi for NVIDIA GPUs
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,driver_version,memory.total", "--format=csv,noheader")
	output, err := cmd.Output()
	if err == nil {
		parts := strings.Split(strings.TrimSpace(string(output)), ",")
		if len(parts) >= 3 {
			model = strings.TrimSpace(parts[0])
			driver = strings.TrimSpace(parts[1])
			memStr := strings.TrimSpace(strings.TrimSuffix(parts[2], " MiB"))
			if mem, err := strconv.ParseFloat(memStr, 64); err == nil {
				memoryGB = mem / 1024 // Convert MiB to GB
			}
			return
		}
	}

	return "", "", 0
}

// collectNetworkInterfaces collects static network interface information
func (s *StaticInfoCollector) collectNetworkInterfaces() []protocol.StaticNetworkInterface {
	var interfaces []protocol.StaticNetworkInterface

	if runtime.GOOS != "linux" {
		return interfaces
	}

	// Read network interfaces from /sys/class/net
	netDir := "/sys/class/net"
	entries, err := os.ReadDir(netDir)
	if err != nil {
		return interfaces
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		ifName := entry.Name()

		// Skip loopback
		if ifName == "lo" {
			continue
		}

		iface := protocol.StaticNetworkInterface{
			InterfaceName: ifName,
			MACAddress:    s.getMACAddress(ifName),
			InterfaceType: s.getInterfaceType(ifName),
			SpeedMbps:     s.getInterfaceSpeed(ifName),
			IsPhysical:    s.isPhysicalInterface(ifName),
		}

		interfaces = append(interfaces, iface)
	}

	return interfaces
}

// getMACAddress retrieves MAC address for an interface
func (s *StaticInfoCollector) getMACAddress(ifName string) string {
	macFile := filepath.Join("/sys/class/net", ifName, "address")
	data, err := os.ReadFile(macFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// getInterfaceType determines interface type
func (s *StaticInfoCollector) getInterfaceType(ifName string) string {
	// Check if wireless
	wirelessDir := filepath.Join("/sys/class/net", ifName, "wireless")
	if _, err := os.Stat(wirelessDir); err == nil {
		return "wireless"
	}

	// Check if virtual
	if strings.HasPrefix(ifName, "veth") || strings.HasPrefix(ifName, "br-") ||
		strings.HasPrefix(ifName, "docker") || strings.Contains(ifName, "tun") {
		return "virtual"
	}

	return "ethernet"
}

// getInterfaceSpeed retrieves interface speed in Mbps
func (s *StaticInfoCollector) getInterfaceSpeed(ifName string) int {
	speedFile := filepath.Join("/sys/class/net", ifName, "speed")
	data, err := os.ReadFile(speedFile)
	if err != nil {
		return 0
	}

	speed, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}

	return speed
}

// isPhysicalInterface checks if interface is physical
func (s *StaticInfoCollector) isPhysicalInterface(ifName string) bool {
	// Virtual interfaces
	if strings.HasPrefix(ifName, "veth") || strings.HasPrefix(ifName, "br-") ||
		strings.HasPrefix(ifName, "docker") || strings.Contains(ifName, "tun") ||
		strings.HasPrefix(ifName, "virbr") {
		return false
	}

	return true
}

// collectDiskInfo collects static disk information
func (s *StaticInfoCollector) collectDiskInfo() []protocol.DiskStaticInfo {
	var disks []protocol.DiskStaticInfo

	if runtime.GOOS != "linux" {
		return disks
	}

	// Read /proc/mounts
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return disks
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		device := fields[0]
		mountPoint := fields[1]
		fsType := fields[2]

		// Skip non-physical filesystems
		if !strings.HasPrefix(device, "/dev/") {
			continue
		}

		// Skip special mounts
		if mountPoint == "/boot/efi" || strings.Contains(mountPoint, "efi") {
			continue
		}

		// Get total size
		totalGB := s.getDiskTotalSize(mountPoint)
		if totalGB == 0 {
			continue
		}

		disk := protocol.DiskStaticInfo{
			DeviceName:     device,
			FilesystemType: fsType,
			MountPoint:     mountPoint,
			TotalGB:        totalGB,
		}

		disks = append(disks, disk)
	}

	return disks
}

// getDiskTotalSize retrieves total disk size in GB
func (s *StaticInfoCollector) getDiskTotalSize(mountPoint string) int {
	cmd := exec.Command("df", "-BG", mountPoint)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return 0
	}

	totalStr := strings.TrimSuffix(fields[1], "G")
	total, err := strconv.Atoi(totalStr)
	if err != nil {
		return 0
	}

	return total
}
