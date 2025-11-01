package docker

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// CheckDockerAvailability checks if Docker is running and accessible
func (c *Client) CheckDockerAvailability(ctx context.Context) error {
	c.logger.Debug("Checking Docker availability")

	// Create a context with timeout for the health check
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to run 'docker version' command
	cmd := exec.CommandContext(healthCtx, "docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.CombinedOutput()

	if err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "cannot connect") ||
			strings.Contains(outputStr, "connection refused") ||
			strings.Contains(outputStr, "system cannot find the file") ||
			strings.Contains(outputStr, "pipe") {
			c.logger.Warn("Docker daemon is not running")
			return NewDockerUnavailableError("Docker daemon is not running. Please start Docker Desktop.")
		}

		c.logger.WithError(err).Error("Docker health check failed")
		return NewDockerUnavailableError("Docker is not available: " + err.Error())
	}

	c.logger.WithField("server_version", strings.TrimSpace(string(output))).Debug("Docker is available")
	return nil
}

// DockerUnavailableError represents an error when Docker is not available
type DockerUnavailableError struct {
	message string
}

func (e *DockerUnavailableError) Error() string {
	return e.message
}

func (e *DockerUnavailableError) IsDockerUnavailable() bool {
	return true
}

// NewDockerUnavailableError creates a new DockerUnavailableError
func NewDockerUnavailableError(message string) *DockerUnavailableError {
	return &DockerUnavailableError{message: message}
}

// IsDockerUnavailableError checks if an error is a DockerUnavailableError
func IsDockerUnavailableError(err error) bool {
	if dockerErr, ok := err.(*DockerUnavailableError); ok {
		return dockerErr.IsDockerUnavailable()
	}
	return false
}
