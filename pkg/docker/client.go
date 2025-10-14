package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// Client represents a Docker client
type Client struct {
	logger *logrus.Logger
}

// NewClient creates a new Docker client
func NewClient(logger *logrus.Logger) *Client {
	return &Client{
		logger: logger,
	}
}

// dockerContainer represents Docker container JSON output
type dockerContainer struct {
	ID     string       `json:"Id"`
	Names  string       `json:"Names"`
	Image  string       `json:"Image"`
	Status string       `json:"Status"`
	State  string       `json:"State"`
	Ports  string       `json:"Ports"`
	Labels string       `json:"Labels"`
}

// dockerPort represents Docker port mapping
type dockerPort struct {
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort,omitempty"`
	Type        string `json:"Type"`
	IP          string `json:"IP,omitempty"`
}

// GetContainers retrieves information about Docker containers
func (c *Client) GetContainers(ctx context.Context) (*protocol.ContainersPayload, error) {
	c.logger.Debug("Getting Docker containers information")

	// Check if Docker is available
	if err := c.checkDockerAvailable(); err != nil {
		return nil, fmt.Errorf("Docker not available: %v", err)
	}

	// Execute docker ps command with JSON format
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		c.logger.WithError(err).Error("Failed to execute docker ps command")
		return nil, fmt.Errorf("failed to get containers: %v", err)
	}

	containers, err := c.parseContainers(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse containers: %v", err)
	}

	payload := &protocol.ContainersPayload{
		Containers: containers,
		Total:      len(containers),
	}

	c.logger.WithField("containers_count", len(containers)).Info("Successfully retrieved Docker containers")
	return payload, nil
}

// checkDockerAvailable checks if Docker is available and accessible
func (c *Client) checkDockerAvailable() error {
	cmd := exec.Command("docker", "version", "--format", "json")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker command failed: %v", err)
	}
	return nil
}

// parseContainers parses Docker containers from JSON output
func (c *Client) parseContainers(output []byte) ([]protocol.ContainerInfo, error) {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	containers := make([]protocol.ContainerInfo, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		var dockerContainer dockerContainer
		if err := json.Unmarshal([]byte(line), &dockerContainer); err != nil {
			c.logger.WithError(err).WithField("line", line).Warn("Failed to parse container JSON")
			continue
		}

		container := c.convertToContainerInfo(dockerContainer)
		containers = append(containers, container)
	}

	return containers, nil
}

// convertToContainerInfo converts Docker container to protocol ContainerInfo
func (c *Client) convertToContainerInfo(dc dockerContainer) protocol.ContainerInfo {
	// Clean container name (remove leading slash)
	name := "unknown"
	if dc.Names != "" {
		name = strings.TrimPrefix(dc.Names, "/")
	}

	// Format ports - Docker returns ports as string like "8080/tcp" or "0.0.0.0:6379->6379/tcp"
	var ports []string
	if dc.Ports != "" {
		// Split multiple ports by comma if needed
		portStrings := strings.Split(dc.Ports, ", ")
		for _, portStr := range portStrings {
			if strings.TrimSpace(portStr) != "" {
				ports = append(ports, strings.TrimSpace(portStr))
			}
		}
	}

	// Use short ID (max 12 chars)
	shortID := dc.ID
	if len(dc.ID) > 12 {
		shortID = dc.ID[:12]
	}
	
	return protocol.ContainerInfo{
		ID:     shortID,
		Name:   name,
		Image:  dc.Image,
		Status: dc.Status,
		State:  dc.State,
		Ports:  ports,
		Labels: map[string]string{"raw": dc.Labels},
	}
}
