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

// RemoveContainer removes a Docker container
func (c *Client) RemoveContainer(ctx context.Context, containerID string) (*protocol.ContainerActionResponse, error) {
	c.logger.WithField("container_id", containerID).Info("Removing Docker container")

	// Check Docker availability first
	if err := c.CheckDockerAvailability(ctx); err != nil {
		return &protocol.ContainerActionResponse{
			ContainerID: containerID,
			Action:      "remove",
			Success:     false,
			Message:     err.Error(),
		}, nil
	}

	// Use -f flag to force removal
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerID)
	output, err := cmd.CombinedOutput()

	response := &protocol.ContainerActionResponse{
		ContainerID: containerID,
		Action:      "remove",
		Success:     err == nil,
		Message:     string(output),
	}

	if err != nil {
		c.logger.WithError(err).Error("Failed to remove container")
		response.Message = fmt.Sprintf("Failed to remove container: %v", err)
		return response, nil
	}

	response.NewState = "removed"
	c.logger.Info("Container removed successfully")
	return response, nil
}

// CreateContainer creates and starts a new Docker container
func (c *Client) CreateContainer(ctx context.Context, payload *protocol.CreateContainerPayload) (*protocol.ContainerActionResponse, error) {
	c.logger.WithField("container_name", payload.Name).Info("Creating Docker container")

	// Check Docker availability first
	if err := c.CheckDockerAvailability(ctx); err != nil {
		return &protocol.ContainerActionResponse{
			ContainerName: payload.Name,
			Action:        "create",
			Success:       false,
			Message:       err.Error(),
		}, nil
	}

	// Build docker run command
	args := []string{"run", "-d", "--name", payload.Name}

	// Add port mappings
	for containerPort, hostPort := range payload.Ports {
		args = append(args, "-p", fmt.Sprintf("%s:%s", hostPort, containerPort))
	}

	// Add environment variables
	for key, value := range payload.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add volume mappings
	for hostPath, containerPath := range payload.Volumes {
		args = append(args, "-v", fmt.Sprintf("%s:%s", hostPath, containerPath))
	}

	// Add image name
	args = append(args, payload.Image)

	// Execute docker run command
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()

	response := &protocol.ContainerActionResponse{
		ContainerName: payload.Name,
		Action:        "create",
		Success:       err == nil,
		Message:       string(output),
	}

	if err != nil {
		c.logger.WithError(err).Error("Failed to create container")
		outputStr := string(output)
		
		// Check for common errors and provide helpful messages
		if strings.Contains(outputStr, "address already in use") {
			// Extract port from error message
			response.Message = "⚠️ Port is already in use. Please choose a different port or use random port assignment."
		} else if strings.Contains(outputStr, "name is already in use") {
			response.Message = "⚠️ Container name already exists. Please choose a different name."
		} else {
			response.Message = fmt.Sprintf("Failed to create container: %v\nOutput: %s", err, outputStr)
		}
		return response, nil
	}

	// Get container ID from output (first line)
	containerID := strings.TrimSpace(string(output))
	response.ContainerID = containerID
	response.NewState = "running"
	response.Message = fmt.Sprintf("Container created and started successfully (ID: %s)", containerID[:12])

	c.logger.Info("Container created successfully")
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
