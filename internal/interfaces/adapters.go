package interfaces

import (
	"context"

	"github.com/godofphonk/ServerEye/pkg/docker"
	"github.com/godofphonk/ServerEye/pkg/metrics"
	"github.com/godofphonk/ServerEye/pkg/protocol"
)

// SystemMonitorAdapter adapts metrics.SystemMonitor to interfaces.SystemMonitor
type SystemMonitorAdapter struct {
	*metrics.SystemMonitor
}

// NewSystemMonitorAdapter creates a new system monitor adapter
func NewSystemMonitorAdapter(monitor *metrics.SystemMonitor) SystemMonitor {
	return &SystemMonitorAdapter{SystemMonitor: monitor}
}

// GetProcesses implements the missing method
func (s *SystemMonitorAdapter) GetProcesses() (*protocol.ProcessesPayload, error) {
	// Placeholder implementation - would need to be added to metrics.SystemMonitor
	return &protocol.ProcessesPayload{
		Processes: []protocol.ProcessInfo{},
		Total:     0,
	}, nil
}

// GetSystemInfo implements the missing method
func (s *SystemMonitorAdapter) GetSystemInfo() (*protocol.SystemInfoPayload, error) {
	// Call the original method if available
	if s.SystemMonitor != nil {
		// This would need to be implemented in metrics.SystemMonitor
		return &protocol.SystemInfoPayload{}, nil
	}
	return &protocol.SystemInfoPayload{}, nil
}

// DockerManagerAdapter adapts docker.Client to interfaces.DockerManager
type DockerManagerAdapter struct {
	*docker.Client
}

// NewDockerManagerAdapter creates a new Docker manager adapter
func NewDockerManagerAdapter(client *docker.Client) DockerManager {
	return &DockerManagerAdapter{Client: client}
}

// CreateContainer adapts the return type
func (d *DockerManagerAdapter) CreateContainer(ctx context.Context, config *protocol.CreateContainerPayload) error {
	// Call original method and ignore response
	_, err := d.Client.CreateContainer(ctx, config)
	return err
}

// StartContainer implements missing method
func (d *DockerManagerAdapter) StartContainer(ctx context.Context, containerID string) error {
	// Placeholder implementation
	return nil
}

// StopContainer implements missing method
func (d *DockerManagerAdapter) StopContainer(ctx context.Context, containerID string) error {
	// Placeholder implementation
	return nil
}

// RestartContainer implements missing method
func (d *DockerManagerAdapter) RestartContainer(ctx context.Context, containerID string) error {
	// Placeholder implementation
	return nil
}

// RemoveContainer implements missing method
func (d *DockerManagerAdapter) RemoveContainer(ctx context.Context, containerID string) error {
	// Placeholder implementation
	return nil
}

// IsAvailable implements missing method
func (d *DockerManagerAdapter) IsAvailable() bool {
	// Placeholder implementation
	return true
}
