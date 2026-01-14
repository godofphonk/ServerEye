package metrics

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// NetworkInfo represents detailed network statistics
type NetworkInfo struct {
	Interfaces  []NetworkInterface `json:"interfaces"`
	TotalRxMbps float64            `json:"total_rx_mbps"`
	TotalTxMbps float64            `json:"total_tx_mbps"`
}

// NetworkInterface represents network interface statistics
type NetworkInterface struct {
	Name        string  `json:"name"`
	RxBytes     uint64  `json:"rx_bytes"`
	TxBytes     uint64  `json:"tx_bytes"`
	RxPackets   uint64  `json:"rx_packets"`
	TxPackets   uint64  `json:"tx_packets"`
	RxSpeedMbps float64 `json:"rx_speed_mbps"`
	TxSpeedMbps float64 `json:"tx_speed_mbps"`
	Status      string  `json:"status"`
}

// NetworkMetrics represents network metrics collector
type NetworkMetrics struct {
	previousStats map[string]*NetworkInterface
	lastUpdate    time.Time
	logger        *logrus.Logger
}

// NewNetworkMetrics creates a new network metrics collector
func NewNetworkMetrics(logger *logrus.Logger) *NetworkMetrics {
	return &NetworkMetrics{
		previousStats: make(map[string]*NetworkInterface),
		lastUpdate:    time.Now(),
		logger:        logger,
	}
}

// GetNetworkInfo collects detailed network information
func (nm *NetworkMetrics) GetNetworkInfo() (*NetworkInfo, error) {
	// Read network statistics from /proc/net/dev
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/net/dev: %w", err)
	}
	defer file.Close()

	var interfaces []NetworkInterface
	var totalRxSpeed, totalTxSpeed float64
	currentTime := time.Now()
	timeDelta := currentTime.Sub(nm.lastUpdate).Seconds()

	scanner := bufio.NewScanner(file)

	// Skip header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 17 {
			continue
		}

		// Parse interface name and statistics
		ifaceName := strings.TrimSuffix(fields[0], ":")

		// Skip loopback interface
		if ifaceName == "lo" {
			continue
		}

		// Parse network statistics
		rxBytes, _ := strconv.ParseUint(fields[1], 10, 64)
		rxPackets, _ := strconv.ParseUint(fields[2], 10, 64)
		txBytes, _ := strconv.ParseUint(fields[9], 10, 64)
		txPackets, _ := strconv.ParseUint(fields[10], 10, 64)

		// Calculate speed if we have previous data
		var rxSpeedMbps, txSpeedMbps float64
		if prev, exists := nm.previousStats[ifaceName]; exists && timeDelta > 0 {
			rxDelta := float64(rxBytes - prev.RxBytes)
			txDelta := float64(txBytes - prev.TxBytes)

			// Convert bytes to megabits per second
			rxSpeedMbps = (rxDelta * 8) / (timeDelta * 1024 * 1024)
			txSpeedMbps = (txDelta * 8) / (timeDelta * 1024 * 1024)
		}

		// Determine interface status (simple check if interface has traffic)
		status := "up"
		if rxBytes == 0 && txBytes == 0 {
			status = "down"
		}

		iface := NetworkInterface{
			Name:        ifaceName,
			RxBytes:     rxBytes,
			TxBytes:     txBytes,
			RxPackets:   rxPackets,
			TxPackets:   txPackets,
			RxSpeedMbps: rxSpeedMbps,
			TxSpeedMbps: txSpeedMbps,
			Status:      status,
		}

		interfaces = append(interfaces, iface)
		totalRxSpeed += rxSpeedMbps
		totalTxSpeed += txSpeedMbps

		// Update previous stats
		nm.previousStats[ifaceName] = &iface
	}

	nm.lastUpdate = currentTime

	networkInfo := &NetworkInfo{
		Interfaces:  interfaces,
		TotalRxMbps: totalRxSpeed,
		TotalTxMbps: totalTxSpeed,
	}

	nm.logger.WithFields(map[string]interface{}{
		"interfaces_count": len(interfaces),
		"total_rx_mbps":    totalRxSpeed,
		"total_tx_mbps":    totalTxSpeed,
	}).Debug("Network metrics collected")

	return networkInfo, nil
}

// GetNetworkSpeed returns total network speed in Mbps
func (nm *NetworkMetrics) GetNetworkSpeed() (float64, float64, error) {
	networkInfo, err := nm.GetNetworkInfo()
	if err != nil {
		return 0, 0, err
	}

	return networkInfo.TotalRxMbps, networkInfo.TotalTxMbps, nil
}
