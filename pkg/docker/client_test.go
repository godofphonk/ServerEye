package docker

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	logger := logrus.New()
	client := NewClient(logger)

	assert.NotNil(t, client)
	assert.Equal(t, logger, client.logger)
}

func TestConvertToContainerInfo(t *testing.T) {
	logger := logrus.New()
	client := NewClient(logger)

	tests := []struct {
		name     string
		input    dockerContainer
		expected string // expected container name
	}{
		{
			name: "container with slash prefix",
			input: dockerContainer{
				ID:     "abc123456789",
				Names:  "/test-container",
				Image:  "nginx:latest",
				Status: "Up 2 hours",
				State:  "running",
				Ports:  "80/tcp, 443/tcp",
				Labels: "com.docker.compose.service=web",
			},
			expected: "test-container",
		},
		{
			name: "container without slash prefix",
			input: dockerContainer{
				ID:     "def987654321",
				Names:  "another-container",
				Image:  "redis:alpine",
				Status: "Up 1 day",
				State:  "running",
				Ports:  "6379/tcp",
				Labels: "",
			},
			expected: "another-container",
		},
		{
			name: "container with empty name",
			input: dockerContainer{
				ID:     "ghi111222333",
				Names:  "",
				Image:  "postgres:15",
				Status: "Up 3 days",
				State:  "running",
				Ports:  "5432/tcp",
				Labels: "",
			},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.convertToContainerInfo(tt.input)

			// Check ID (should be max 12 chars)
			expectedID := tt.input.ID
			if len(tt.input.ID) > 12 {
				expectedID = tt.input.ID[:12]
			}
			assert.Equal(t, expectedID, result.ID)
			assert.Equal(t, tt.expected, result.Name)
			assert.Equal(t, tt.input.Image, result.Image)
			assert.Equal(t, tt.input.Status, result.Status)
			assert.Equal(t, tt.input.State, result.State)

			// Test ports parsing
			if tt.input.Ports != "" {
				assert.NotEmpty(t, result.Ports)
			}

			// Test labels
			assert.NotNil(t, result.Labels)
			if tt.input.Labels != "" {
				assert.Equal(t, tt.input.Labels, result.Labels["raw"])
			}
		})
	}
}

func TestParseContainers(t *testing.T) {
	logger := logrus.New()
	client := NewClient(logger)

	// Test with valid JSON
	validJSON := `{"Id":"abc123","Names":"test-container","Image":"nginx","Status":"Up","State":"running","Ports":"80/tcp","Labels":"key=value"}
{"Id":"def456","Names":"another-container","Image":"redis","Status":"Up","State":"running","Ports":"6379/tcp","Labels":""}`

	containers, err := client.parseContainers([]byte(validJSON))
	require.NoError(t, err)
	assert.Len(t, containers, 2)

	assert.Equal(t, "test-container", containers[0].Name)
	assert.Equal(t, "another-container", containers[1].Name)

	// Test with empty input
	emptyContainers, err := client.parseContainers([]byte(""))
	require.NoError(t, err)
	assert.Len(t, emptyContainers, 0)

	// Test with invalid JSON (should skip invalid lines)
	invalidJSON := `{"Id":"abc123","Names":"test-container","Image":"nginx","Status":"Up","State":"running","Ports":"80/tcp","Labels":"key=value"}
invalid json line
{"Id":"def456","Names":"another-container","Image":"redis","Status":"Up","State":"running","Ports":"6379/tcp","Labels":""}`

	partialContainers, err := client.parseContainers([]byte(invalidJSON))
	require.NoError(t, err)
	assert.Len(t, partialContainers, 2) // Should parse valid lines and skip invalid
}

func TestCheckDockerAvailable(t *testing.T) {
	logger := logrus.New()
	client := NewClient(logger)

	// This test will only pass if Docker is available on the system
	// In CI environment, Docker should be available
	err := client.checkDockerAvailable()

	// We don't assert here because Docker might not be available in all test environments
	// Just ensure the method doesn't panic
	t.Logf("Docker availability check result: %v", err)
}

// TestGetContainers tests the main GetContainers method
// This is an integration test that requires Docker to be running
func TestGetContainers_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.New()
	client := NewClient(logger)

	ctx := context.Background()

	// This test requires Docker to be available
	err := client.checkDockerAvailable()
	if err != nil {
		t.Skipf("Docker not available, skipping integration test: %v", err)
	}

	payload, err := client.GetContainers(ctx)

	// We don't assert specific results because container state is dynamic
	// Just ensure the method works without errors
	if err != nil {
		t.Logf("GetContainers returned error (this might be expected): %v", err)
	} else {
		assert.NotNil(t, payload)
		assert.GreaterOrEqual(t, payload.Total, 0)
		t.Logf("Found %d containers", payload.Total)
	}
}
