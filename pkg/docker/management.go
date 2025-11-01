package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/servereye/servereye/pkg/protocol"
)

// StartContainer starts a Docker container
func (c *Client) StartContainer(ctx context.Context, containerID string) (*protocol.ContainerActionResponse, error) {
	c.logger.WithField("container_id", containerID).Info("Starting Docker container")

	// Check Docker availability first
	if err := c.CheckDockerAvailability(ctx); err != nil {
		return &protocol.ContainerActionResponse{
			ContainerID: containerID,
			Action:      "start",
			Success:     false,
			Message:     err.Error(),
		}, nil
	}

	cmd := exec.CommandContext(ctx, "docker", "start", containerID)
	output, err := cmd.CombinedOutput()

	response := &protocol.ContainerActionResponse{
		ContainerID: containerID,
		Action:      "start",
		Success:     err == nil,
		Message:     string(output),
	}

	if err != nil {
		c.logger.WithError(err).Error("Failed to start container")
		response.Message = fmt.Sprintf("Failed to start container: %v", err)
		return response, nil
	}

	// Get updated container state
	if state, stateErr := c.getContainerState(ctx, containerID); stateErr == nil {
		response.NewState = state
	}

	c.logger.Info("Container started successfully")
	return response, nil
}

// StopContainer stops a Docker container
func (c *Client) StopContainer(ctx context.Context, containerID string) (*protocol.ContainerActionResponse, error) {
	c.logger.WithField("container_id", containerID).Info("Stopping Docker container")

	// Check Docker availability first
	if err := c.CheckDockerAvailability(ctx); err != nil {
		return &protocol.ContainerActionResponse{
			ContainerID: containerID,
			Action:      "stop",
			Success:     false,
			Message:     err.Error(),
		}, nil
	}

	cmd := exec.CommandContext(ctx, "docker", "stop", containerID)
	output, err := cmd.CombinedOutput()

	response := &protocol.ContainerActionResponse{
		ContainerID: containerID,
		Action:      "stop",
		Success:     err == nil,
		Message:     string(output),
	}

	if err != nil {
		c.logger.WithError(err).Error("Failed to stop container")
		response.Message = fmt.Sprintf("Failed to stop container: %v", err)
		return response, nil
	}

	// Get updated container state
	if state, stateErr := c.getContainerState(ctx, containerID); stateErr == nil {
		response.NewState = state
	}

	c.logger.Info("Container stopped successfully")
	return response, nil
}

// RestartContainer restarts a Docker container
func (c *Client) RestartContainer(ctx context.Context, containerID string) (*protocol.ContainerActionResponse, error) {
	c.logger.WithField("container_id", containerID).Info("Restarting Docker container")

	// Check Docker availability first
	if err := c.CheckDockerAvailability(ctx); err != nil {
		return &protocol.ContainerActionResponse{
			ContainerID: containerID,
			Action:      "restart",
			Success:     false,
			Message:     err.Error(),
		}, nil
	}

	cmd := exec.CommandContext(ctx, "docker", "restart", containerID)
	output, err := cmd.CombinedOutput()

	response := &protocol.ContainerActionResponse{
		ContainerID: containerID,
		Action:      "restart",
		Success:     err == nil,
		Message:     string(output),
	}

	if err != nil {
		c.logger.WithError(err).Error("Failed to restart container")
		response.Message = fmt.Sprintf("Failed to restart container: %v", err)
		return response, nil
	}

	// Get updated container state
	if state, stateErr := c.getContainerState(ctx, containerID); stateErr == nil {
		response.NewState = state
	}

	c.logger.Info("Container restarted successfully")
	return response, nil
}

// getContainerState gets the current state of a container
func (c *Client) getContainerState(ctx context.Context, containerID string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Status}}", containerID)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
