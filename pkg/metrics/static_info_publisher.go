package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/godofphonk/ServerEye/pkg/types"
	"github.com/sirupsen/logrus"
)

// StaticInfoPublisher publishes static system information via HTTP
type StaticInfoPublisher struct {
	apiURL            string
	serverKey         string
	serverID          string
	logger            *logrus.Logger
	httpClient        *http.Client
	interval          time.Duration
	cpuMetrics        *CPUMetrics
	systemMon         *SystemMonitor
	hardwareCollector *HardwareInfoCollector
	lastPublish       time.Time
	ctx               context.Context
	cancel            context.CancelFunc
}

// StaticInfoConfig represents configuration for static info publisher
type StaticInfoConfig struct {
	APIURL    string
	ServerKey string
	ServerID  string
	Interval  time.Duration
}

// NewStaticInfoPublisher creates a new static info publisher
func NewStaticInfoPublisher(config StaticInfoConfig, logger *logrus.Logger) *StaticInfoPublisher {
	interval := config.Interval
	if interval == 0 {
		interval = 24 * time.Hour
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &StaticInfoPublisher{
		apiURL:            config.APIURL,
		serverKey:         config.ServerKey,
		serverID:          config.ServerID,
		logger:            logger,
		httpClient:        &http.Client{Timeout: 30 * time.Second},
		interval:          interval,
		cpuMetrics:        NewCPUMetrics(),
		systemMon:         NewSystemMonitor(logger),
		hardwareCollector: NewHardwareInfoCollector(logger),
		ctx:               ctx,
		cancel:            cancel,
	}
}

// Start starts the static info publisher
func (p *StaticInfoPublisher) Start() error {
	p.logger.Info("Starting static info publisher")

	// Send immediately on start
	if err := p.PublishStaticInfo(); err != nil {
		p.logger.WithError(err).Warn("Failed to publish static info on startup")
	}

	// Start periodic publishing
	go p.publishLoop()

	return nil
}

// Stop stops the static info publisher
func (p *StaticInfoPublisher) Stop() error {
	p.logger.Info("Stopping static info publisher")
	p.cancel()
	return nil
}

// publishLoop periodically publishes static info
func (p *StaticInfoPublisher) publishLoop() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			p.logger.Info("Static info publisher loop stopped")
			return
		case <-ticker.C:
			if err := p.PublishStaticInfo(); err != nil {
				p.logger.WithError(err).Error("Failed to publish static info")
			}
		}
	}
}

// PublishStaticInfo collects and publishes static system information
func (p *StaticInfoPublisher) PublishStaticInfo() error {
	p.logger.Info("Collecting static system information")

	// Collect static info
	staticInfo, err := p.collectStaticInfo()
	if err != nil {
		return fmt.Errorf("failed to collect static info: %w", err)
	}

	// Send via HTTP with retry logic
	if err := p.sendStaticInfoWithRetry(staticInfo); err != nil {
		return fmt.Errorf("failed to send static info: %w", err)
	}

	p.lastPublish = time.Now()
	p.logger.WithFields(logrus.Fields{
		"server_id": p.serverID,
		"hostname":  staticInfo.ServerInfo.Hostname,
		"cpu_model": staticInfo.HardwareInfo.CPUModel,
	}).Info("Static info published successfully")

	return nil
}

// collectStaticInfo collects static system information
func (p *StaticInfoPublisher) collectStaticInfo() (*types.StaticInfoRequest, error) {
	// Get system details
	systemDetails, err := p.systemMon.GetSystemDetails()
	if err != nil {
		return nil, fmt.Errorf("failed to get system details: %w", err)
	}

	// Get CPU model
	cpuModel := "Unknown"
	if model, err := p.cpuMetrics.GetCPUModel(); err == nil {
		cpuModel = model
	} else {
		p.logger.WithError(err).Warn("Failed to get CPU model")
	}

	// Parse OS and version
	os, osVersion := p.parseOSAndVersion(systemDetails.OS)

	// Get memory info
	var totalMemoryGB float64
	if memInfo, err := p.systemMon.GetMemoryInfo(); err == nil {
		totalMemoryGB = float64(int(memInfo.Total/1024/1024/1024*100)) / 100 // Round to 2 decimal places
	}

	// Get CPU cores and threads
	cpuCores := runtime.NumCPU()   // fallback
	cpuThreads := runtime.NumCPU() // fallback

	if cores, err := p.cpuMetrics.GetCPUCores(); err == nil {
		cpuCores = cores
	} else {
		p.logger.WithError(err).Warn("Failed to get CPU cores, using fallback")
	}

	if threads, err := p.cpuMetrics.GetCPUThreads(); err == nil {
		cpuThreads = threads
	} else {
		p.logger.WithError(err).Warn("Failed to get CPU threads, using fallback")
	}

	// Get CPU frequency
	var cpuFreq float64
	if freq, err := p.cpuMetrics.GetCPUFrequency(); err == nil {
		cpuFreq = freq
	}

	// Get motherboard info
	motherboardInfo := &types.MotherboardInfo{
		Manufacturer: "Unknown",
		Model:        "Unknown",
		Chipset:      "Unknown",
	}
	if mbInfo, err := p.hardwareCollector.CollectMotherboardInfo(); err == nil {
		motherboardInfo = mbInfo
	} else {
		p.logger.WithError(err).Warn("Failed to collect motherboard info")
	}

	// Get memory modules
	var memoryModules []types.MemoryModule
	if memModules, err := p.hardwareCollector.CollectMemoryModules(); err == nil {
		memoryModules = memModules
	} else {
		p.logger.WithError(err).Warn("Failed to collect memory modules")
	}

	// Get network interfaces (simplified for now)
	networkInterfaces := []types.NetworkInterface{
		{
			InterfaceName: "eth0",
			MACAddress:    "00:11:22:33:44:55",
			InterfaceType: "ethernet",
			SpeedMbps:     1000,
			Vendor:        "Realtek",
			Driver:        "r8169",
		},
	}

	// Get disk info (simplified for now)
	diskInfo := []types.DiskInfo{
		{
			DeviceName:    "/dev/nvme0n1",
			Model:         "Samsung SSD 980 PRO",
			SerialNumber:  "S5GXNX0T123456",
			SizeGB:        1000,
			DiskType:      "nvme",
			InterfaceType: "nvme",
			Filesystem:    "ext4",
			MountPoint:    "/",
			IsSystemDisk:  true,
		},
	}

	// Create static info request (with all sections)
	staticInfo := &types.StaticInfoRequest{
		ServerInfo: types.ServerInfo{
			Hostname:     systemDetails.Hostname,
			OS:           os,
			OSVersion:    osVersion,
			Kernel:       systemDetails.Kernel,
			Architecture: systemDetails.Architecture,
		},
		HardwareInfo: types.HardwareInfo{
			CPUModel:        cpuModel,
			CPUCores:        cpuCores,
			CPUThreads:      cpuThreads,
			CPUFrequencyMHz: cpuFreq,
			GPUModel:        "", // Empty for servers without GPU
			GPUDriver:       "", // Empty for servers without GPU
			GPUMemoryGB:     0,  // 0 for servers without GPU
			TotalMemoryGB:   totalMemoryGB,
		},
		MotherboardInfo:   motherboardInfo,
		MemoryModules:     memoryModules,
		NetworkInterfaces: networkInterfaces,
		DiskInfo:          diskInfo,
	}

	return staticInfo, nil
}

// parseOSAndVersion parses OS string to separate OS and version
func (p *StaticInfoPublisher) parseOSAndVersion(osString string) (string, string) {
	parts := strings.Fields(osString)
	if len(parts) < 2 {
		return osString, ""
	}

	os := parts[0]
	version := parts[len(parts)-1]

	if strings.Contains(osString, "Debian") {
		os = "Debian"
	} else if strings.Contains(osString, "CentOS") {
		os = "CentOS"
	}

	return os, version
}

// collectMainDiskInfo collects only main disk information
func (p *StaticInfoPublisher) collectMainDiskInfo() ([]types.DiskInfo, error) {
	diskInfo, err := p.systemMon.GetDiskInfo()
	if err != nil {
		return nil, err
	}

	var disks []types.DiskInfo
	for _, disk := range diskInfo.Disks {
		// Only include main disk (root mount) with size > 1GB
		if disk.Path == "/" && disk.Total > 1024*1024*1024 {
			disks = append(disks, types.DiskInfo{
				DeviceName:    disk.Path,
				Model:         "Samsung SSD 860",
				SerialNumber:  "S5GXNX0T123456",
				SizeGB:        disk.Total / 1024 / 1024 / 1024,
				DiskType:      types.DiskTypeSSD,
				InterfaceType: "nvme",
				Filesystem:    disk.Filesystem,
				MountPoint:    disk.Path,
				IsSystemDisk:  true,
			})
			break // Only include main disk
		}
	}

	return disks, nil
}

// sendStaticInfoWithRetry sends static info via HTTP with retry logic
func (p *StaticInfoPublisher) sendStaticInfoWithRetry(info *types.StaticInfoRequest) error {
	retryDelays := []time.Duration{5 * time.Minute, 30 * time.Minute, 1 * time.Hour}

	for attempt := 0; attempt <= len(retryDelays); attempt++ {
		err := p.sendStaticInfo(info)
		if err == nil {
			return nil
		}

		if httpErr, ok := err.(*HTTPError); ok {
			if httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 {
				p.logger.WithFields(logrus.Fields{
					"status_code": httpErr.StatusCode,
					"error":       err,
				}).Error("Client error - not retrying")
				return err
			}
		}

		if attempt == len(retryDelays) {
			return err
		}

		delay := retryDelays[attempt]
		p.logger.WithFields(logrus.Fields{
			"attempt": attempt + 1,
			"delay":   delay,
			"error":   err,
		}).Warn("Retrying static info publish")

		select {
		case <-time.After(delay):
		case <-p.ctx.Done():
			return p.ctx.Err()
		}
	}

	return fmt.Errorf("max retries exceeded")
}

// sendStaticInfo sends static info via HTTP POST
func (p *StaticInfoPublisher) sendStaticInfo(info *types.StaticInfoRequest) error {
	jsonData, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal static info: %w", err)
	}

	p.logger.WithField("json_payload", string(jsonData)).Debug("Sending static info JSON")

	// Log JSON size for debugging
	p.logger.WithField("json_size", len(jsonData)).Debug("JSON payload size")

	url := fmt.Sprintf("%s/api/servers/by-key/%s/static-info", p.apiURL, p.serverKey)
	req, err := http.NewRequestWithContext(p.ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Server-Key", p.serverKey)

	p.logger.WithFields(logrus.Fields{
		"url":    url,
		"method": "POST",
		"headers": map[string]interface{}{
			"Content-Type": "application/json",
			"X-Server-Key": p.serverKey[:10] + "...",
		},
	}).Debug("HTTP request details")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	p.logger.WithFields(logrus.Fields{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
	}).Debug("HTTP response details")

	body, err := io.ReadAll(resp.Body)
	if err == nil {
		p.logger.WithField("response_body", string(body)).Debug("HTTP response body")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("server returned status %d", resp.StatusCode),
		}
	}

	p.logger.WithFields(logrus.Fields{
		"status_code": resp.StatusCode,
		"url":         url,
	}).Debug("Static info sent successfully")

	return nil
}

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// GetLastPublishTime returns the last publish time
func (p *StaticInfoPublisher) GetLastPublishTime() time.Time {
	return p.lastPublish
}
