package metrics

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// SystemMonitor provides system monitoring capabilities
type SystemMonitor struct {
	logger *logrus.Logger
	// Previous network stats for speed calculation
	prevNetStats map[string]*networkStats
	prevNetTime  time.Time
}

// networkStats stores previous network statistics for delta calculation
type networkStats struct {
	bytesSent uint64
	bytesRecv uint64
}

// NewSystemMonitor creates a new system monitor
func NewSystemMonitor(logger *logrus.Logger) *SystemMonitor {
	return &SystemMonitor{
		logger:       logger,
		prevNetStats: make(map[string]*networkStats),
		prevNetTime:  time.Now(),
	}
}

// GetMemoryInfo retrieves system memory information
func (s *SystemMonitor) GetMemoryInfo() (*protocol.MemoryInfo, error) {
	s.logger.Debug("Getting memory information")

	// Read /proc/meminfo for detailed memory stats
	cmd := exec.Command("cat", "/proc/meminfo")
	output, err := cmd.Output()
	if err != nil {
		s.logger.WithError(err).Error("Failed to read /proc/meminfo")
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	memInfo := &protocol.MemoryInfo{}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSuffix(parts[0], ":")
		valueStr := parts[1]
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			continue
		}

		// Convert from KB to bytes
		value *= 1024

		switch key {
		case "MemTotal":
			memInfo.Total = value
		case "MemAvailable":
			memInfo.Available = value
		case "MemFree":
			memInfo.Free = value
		case "Buffers":
			memInfo.Buffers = value
		case "Cached":
			memInfo.Cached = value
		}
	}

	// Calculate used memory
	memInfo.Used = memInfo.Total - memInfo.Available

	// Calculate used percentage
	if memInfo.Total > 0 {
		memInfo.UsedPercent = float64(memInfo.Used) / float64(memInfo.Total) * 100
	}

	s.logger.WithFields(logrus.Fields{
		"total_mb":     memInfo.Total / 1024 / 1024,
		"used_mb":      memInfo.Used / 1024 / 1024,
		"available_mb": memInfo.Available / 1024 / 1024,
		"used_percent": memInfo.UsedPercent,
	}).Debug("Memory info retrieved")

	return memInfo, nil
}

// GetDiskInfo retrieves disk usage information for all mounted filesystems
func (s *SystemMonitor) GetDiskInfo() (*protocol.DiskInfoPayload, error) {
	s.logger.Debug("Getting disk information")

	// Use df command to get disk usage
	cmd := exec.Command("df", "-h", "-x", "tmpfs", "-x", "devtmpfs", "-x", "squashfs")
	output, err := cmd.Output()
	if err != nil {
		s.logger.WithError(err).Error("Failed to execute df command")
		return nil, fmt.Errorf("failed to get disk info: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var disks []protocol.DiskInfo

	// Skip header line
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		filesystem := fields[0]
		totalStr := fields[1]
		usedStr := fields[2]
		availStr := fields[3]
		usedPercentStr := strings.TrimSuffix(fields[4], "%")
		mountPoint := fields[5]

		// Parse sizes (df -h gives human readable format)
		total := s.parseHumanSize(totalStr)
		used := s.parseHumanSize(usedStr)
		free := s.parseHumanSize(availStr)

		usedPercent, _ := strconv.ParseFloat(usedPercentStr, 64)

		diskInfo := protocol.DiskInfo{
			Path:        mountPoint,
			Total:       total,
			Used:        used,
			Free:        free,
			UsedPercent: usedPercent,
			Filesystem:  filesystem,
		}

		disks = append(disks, diskInfo)
	}

	payload := &protocol.DiskInfoPayload{
		Disks: disks,
	}

	s.logger.WithField("disks_count", len(disks)).Debug("Disk info retrieved")
	return payload, nil
}

// GetUptime retrieves system uptime information
func (s *SystemMonitor) GetUptime() (*protocol.UptimeInfo, error) {
	s.logger.Debug("Getting uptime information")

	// Read /proc/uptime
	cmd := exec.Command("cat", "/proc/uptime")
	output, err := cmd.Output()
	if err != nil {
		s.logger.WithError(err).Error("Failed to read /proc/uptime")
		return nil, fmt.Errorf("failed to get uptime: %w", err)
	}

	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 1 {
		return nil, fmt.Errorf("invalid uptime format")
	}

	uptimeFloat, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uptime: %w", err)
	}

	uptimeSeconds := uint64(uptimeFloat)
	now := time.Now().Unix()
	if now < 0 {
		now = 0 // Should never happen, but for gosec
	}
	bootTime := uint64(now) - uptimeSeconds

	// Format uptime in human readable format
	days := uptimeSeconds / 86400
	hours := (uptimeSeconds % 86400) / 3600
	minutes := (uptimeSeconds % 3600) / 60

	var formatted string
	if days > 0 {
		formatted = fmt.Sprintf("%d days, %d hours, %d minutes", days, hours, minutes)
	} else if hours > 0 {
		formatted = fmt.Sprintf("%d hours, %d minutes", hours, minutes)
	} else {
		formatted = fmt.Sprintf("%d minutes", minutes)
	}

	uptimeInfo := &protocol.UptimeInfo{
		Uptime:    uptimeSeconds,
		BootTime:  bootTime,
		Formatted: formatted,
	}

	s.logger.WithFields(logrus.Fields{
		"uptime_seconds": uptimeSeconds,
		"formatted":      formatted,
	}).Debug("Uptime info retrieved")

	return uptimeInfo, nil
}

// GetTopProcesses retrieves top processes by CPU and memory usage
func (s *SystemMonitor) GetTopProcesses(limit int) (*protocol.ProcessesPayload, error) {
	s.logger.WithField("limit", limit).Debug("Getting top processes")

	if limit <= 0 {
		limit = 10
	}

	// Use ps command to get process information
	cmd := exec.Command("ps", "aux", "--sort=-pcpu,-pmem")
	output, err := cmd.Output()
	if err != nil {
		s.logger.WithError(err).Error("Failed to execute ps command")
		return nil, fmt.Errorf("failed to get processes: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var processes []protocol.ProcessInfo

	// Skip header line and process up to limit
	count := 0
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" || count >= limit {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		pid, _ := strconv.ParseInt(fields[1], 10, 32)
		cpuPercent, _ := strconv.ParseFloat(fields[2], 64)
		memPercent, _ := strconv.ParseFloat(fields[3], 32)

		// Memory in KB, convert to MB
		memKB, _ := strconv.ParseUint(fields[5], 10, 64)
		memMB := memKB / 1024

		processInfo := protocol.ProcessInfo{
			PID:           int32(pid),
			Name:          fields[10], // Command name
			CPUPercent:    cpuPercent,
			MemoryMB:      memMB,
			MemoryPercent: float32(memPercent),
			Status:        fields[7],
			Username:      fields[0],
			CreateTime:    0, // Would need additional parsing
		}

		processes = append(processes, processInfo)
		count++
	}

	payload := &protocol.ProcessesPayload{
		Processes: processes,
		Total:     len(processes),
	}

	s.logger.WithField("processes_count", len(processes)).Debug("Top processes retrieved")
	return payload, nil
}

// GetNetworkInfo retrieves network interface statistics
func (s *SystemMonitor) GetNetworkInfo() (*protocol.NetworkInfo, error) {
	s.logger.Debug("Getting network information")

	// Read /proc/net/dev
	cmd := exec.Command("cat", "/proc/net/dev")
	output, err := cmd.Output()
	if err != nil {
		s.logger.WithError(err).Error("Failed to read /proc/net/dev")
		return nil, fmt.Errorf("failed to get network info: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var interfaces []protocol.NetworkInterfaceInfo
	var totalDownload, totalUpload uint64
	var downloadSpeed, uploadSpeed float64

	currentTime := time.Now()
	timeDelta := currentTime.Sub(s.prevNetTime).Seconds()

	// Skip first 2 header lines
	for i, line := range lines {
		if i < 2 || strings.TrimSpace(line) == "" {
			continue
		}

		// Parse interface line
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		ifName := strings.TrimSpace(parts[0])
		
		// Skip loopback
		if ifName == "lo" {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		// Parse statistics
		bytesRecv, _ := strconv.ParseUint(fields[0], 10, 64)
		packetsRecv, _ := strconv.ParseUint(fields[1], 10, 64)
		errorsIn, _ := strconv.ParseUint(fields[2], 10, 64)
		dropIn, _ := strconv.ParseUint(fields[3], 10, 64)

		bytesSent, _ := strconv.ParseUint(fields[8], 10, 64)
		packetsSent, _ := strconv.ParseUint(fields[9], 10, 64)
		errorsOut, _ := strconv.ParseUint(fields[10], 10, 64)
		dropOut, _ := strconv.ParseUint(fields[11], 10, 64)

		// Calculate speed (Mbps) if we have previous data
		var ifDownloadSpeed, ifUploadSpeed float64
		if prev, exists := s.prevNetStats[ifName]; exists && timeDelta > 0 {
			bytesRecvDelta := float64(bytesRecv - prev.bytesRecv)
			bytesSentDelta := float64(bytesSent - prev.bytesSent)
			
			// Convert bytes/sec to Mbps: (bytes/sec * 8) / 1,000,000
			ifDownloadSpeed = (bytesRecvDelta / timeDelta * 8) / 1000000
			ifUploadSpeed = (bytesSentDelta / timeDelta * 8) / 1000000

			downloadSpeed += ifDownloadSpeed
			uploadSpeed += ifUploadSpeed
		}

		// Store current stats for next calculation
		s.prevNetStats[ifName] = &networkStats{
			bytesSent: bytesSent,
			bytesRecv: bytesRecv,
		}

		totalDownload += bytesRecv
		totalUpload += bytesSent

		ifInfo := protocol.NetworkInterfaceInfo{
			Name:        ifName,
			BytesSent:   bytesSent,
			BytesRecv:   bytesRecv,
			PacketsSent: packetsSent,
			PacketsRecv: packetsRecv,
			ErrorsIn:    errorsIn,
			ErrorsOut:   errorsOut,
			DropIn:      dropIn,
			DropOut:     dropOut,
			SpeedMbps:   0, // Link speed not easily available from /proc/net/dev
		}

		interfaces = append(interfaces, ifInfo)
	}

	s.prevNetTime = currentTime

	// Convert total to GB
	totalDownloadGB := totalDownload / 1024 / 1024 / 1024
	totalUploadGB := totalUpload / 1024 / 1024 / 1024

	networkInfo := &protocol.NetworkInfo{
		Interfaces:    interfaces,
		DownloadSpeed: downloadSpeed,
		UploadSpeed:   uploadSpeed,
		TotalDownload: totalDownloadGB,
		TotalUpload:   totalUploadGB,
	}

	s.logger.WithFields(logrus.Fields{
		"interfaces_count": len(interfaces),
		"download_mbps":    downloadSpeed,
		"upload_mbps":      uploadSpeed,
	}).Debug("Network info retrieved")

	return networkInfo, nil
}

// parseHumanSize converts human readable size (like 1.5G, 512M) to bytes
func (s *SystemMonitor) parseHumanSize(sizeStr string) uint64 {
	if len(sizeStr) == 0 {
		return 0
	}

	// Get the last character (unit)
	unit := sizeStr[len(sizeStr)-1:]
	valueStr := sizeStr[:len(sizeStr)-1]

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0
	}

	switch strings.ToUpper(unit) {
	case "K":
		return uint64(value * 1024)
	case "M":
		return uint64(value * 1024 * 1024)
	case "G":
		return uint64(value * 1024 * 1024 * 1024)
	case "T":
		return uint64(value * 1024 * 1024 * 1024 * 1024)
	default:
		// Assume it's already in bytes
		return uint64(value)
	}
}
