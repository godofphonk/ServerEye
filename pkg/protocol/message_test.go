package protocol

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessage(t *testing.T) {
	payload := map[string]interface{}{
		"test": "data",
	}

	msg := NewMessage(TypeGetCPUTemp, payload)

	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, TypeGetCPUTemp, msg.Type)
	assert.Equal(t, "1.0", msg.Version)
	assert.Equal(t, payload, msg.Payload)
	assert.WithinDuration(t, time.Now(), msg.Timestamp, time.Second)
}

func TestMessage_ToJSON(t *testing.T) {
	payload := CPUTempPayload{
		Temperature: 45.5,
		Unit:        "celsius",
		Sensor:      "/sys/class/thermal/thermal_zone0/temp",
	}

	msg := NewMessage(TypeCPUTempResponse, payload)

	data, err := msg.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, string(TypeCPUTempResponse), result["type"])
	assert.Equal(t, "1.0", result["version"])
	assert.NotEmpty(t, result["id"])
	assert.NotEmpty(t, result["timestamp"])
	assert.NotNil(t, result["payload"])
}

func TestFromJSON(t *testing.T) {
	original := NewMessage(TypeGetContainers, nil)

	data, err := original.ToJSON()
	require.NoError(t, err)

	parsed, err := FromJSON(data)
	require.NoError(t, err)

	assert.Equal(t, original.ID, parsed.ID)
	assert.Equal(t, original.Type, parsed.Type)
	assert.Equal(t, original.Version, parsed.Version)
	assert.WithinDuration(t, original.Timestamp, parsed.Timestamp, time.Second)
}

func TestFromJSON_InvalidJSON(t *testing.T) {
	_, err := FromJSON([]byte("invalid json"))
	assert.Error(t, err)
}

func TestMessageTypes(t *testing.T) {
	// Test that all message types are properly defined
	types := []MessageType{
		TypeGetCPUTemp,
		TypeGetSystemInfo,
		TypeGetContainers,
		TypePing,
		TypeCPUTempResponse,
		TypeSystemInfoResponse,
		TypeContainersResponse,
		TypePong,
		TypeErrorResponse,
	}

	for _, msgType := range types {
		assert.NotEmpty(t, string(msgType), "Message type should not be empty")
	}
}

func TestPayloadStructs(t *testing.T) {
	t.Run("CPUTempPayload", func(t *testing.T) {
		payload := CPUTempPayload{
			Temperature: 42.5,
			Unit:        "celsius",
			Sensor:      "/sys/class/thermal/thermal_zone0/temp",
		}

		data, err := json.Marshal(payload)
		require.NoError(t, err)

		var unmarshaled CPUTempPayload
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, payload, unmarshaled)
	})

	t.Run("ContainerInfo", func(t *testing.T) {
		container := ContainerInfo{
			ID:     "abc123456789",
			Name:   "test-container",
			Image:  "nginx:latest",
			Status: "Up 2 hours",
			State:  "running",
			Ports:  []string{"80/tcp", "443/tcp"},
			Labels: map[string]string{"env": "production"},
		}

		data, err := json.Marshal(container)
		require.NoError(t, err)

		var unmarshaled ContainerInfo
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, container, unmarshaled)
	})

	t.Run("ContainersPayload", func(t *testing.T) {
		payload := ContainersPayload{
			Containers: []ContainerInfo{
				{
					ID:    "abc123",
					Name:  "container1",
					Image: "nginx",
					State: "running",
				},
				{
					ID:    "def456",
					Name:  "container2",
					Image: "redis",
					State: "running",
				},
			},
			Total: 2,
		}

		data, err := json.Marshal(payload)
		require.NoError(t, err)

		var unmarshaled ContainersPayload
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, payload, unmarshaled)
	})

	t.Run("ErrorPayload", func(t *testing.T) {
		payload := ErrorPayload{
			ErrorCode:    ErrorSensorNotFound,
			ErrorMessage: "Temperature sensor not found",
		}

		data, err := json.Marshal(payload)
		require.NoError(t, err)

		var unmarshaled ErrorPayload
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, payload, unmarshaled)
	})
}

func TestErrorCodes(t *testing.T) {
	errorCodes := []string{
		ErrorSensorNotFound,
		ErrorPermissionDenied,
		ErrorCommandTimeout,
		ErrorInvalidCommand,
	}

	for _, code := range errorCodes {
		assert.NotEmpty(t, code, "Error code should not be empty")
	}
}

func TestMessage_JSONRoundTrip(t *testing.T) {
	testCases := []struct {
		name    string
		msgType MessageType
		payload interface{}
	}{
		{
			name:    "CPU Temperature Response",
			msgType: TypeCPUTempResponse,
			payload: CPUTempPayload{
				Temperature: 55.5,
				Unit:        "celsius",
				Sensor:      "/sys/class/thermal/thermal_zone0/temp",
			},
		},
		{
			name:    "Containers Response",
			msgType: TypeContainersResponse,
			payload: ContainersPayload{
				Containers: []ContainerInfo{
					{
						ID:     "abc123",
						Name:   "test-container",
						Image:  "nginx:latest",
						Status: "Up 1 hour",
						State:  "running",
						Ports:  []string{"80/tcp"},
						Labels: map[string]string{"env": "test"},
					},
				},
				Total: 1,
			},
		},
		{
			name:    "Error Response",
			msgType: TypeErrorResponse,
			payload: ErrorPayload{
				ErrorCode:    ErrorSensorNotFound,
				ErrorMessage: "CPU temperature sensor not found",
			},
		},
		{
			name:    "Ping Command",
			msgType: TypePing,
			payload: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create message
			original := NewMessage(tc.msgType, tc.payload)

			// Convert to JSON
			data, err := original.ToJSON()
			require.NoError(t, err)

			// Parse back from JSON
			parsed, err := FromJSON(data)
			require.NoError(t, err)

			// Verify basic fields
			assert.Equal(t, original.ID, parsed.ID)
			assert.Equal(t, original.Type, parsed.Type)
			assert.Equal(t, original.Version, parsed.Version)
			assert.WithinDuration(t, original.Timestamp, parsed.Timestamp, time.Second)

			// Verify payload (this is a basic check since payload is interface{})
			if tc.payload != nil {
				assert.NotNil(t, parsed.Payload)
			} else {
				// Payload might be nil or an empty map depending on JSON unmarshaling
				// This is acceptable for commands without payload
			}
		})
	}
}
