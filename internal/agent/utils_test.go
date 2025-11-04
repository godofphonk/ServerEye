package agent

import (
	"testing"

	"github.com/servereye/servereye/pkg/protocol"
)

func TestParsePayload_ContainerAction(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
		wantErr bool
	}{
		{
			name: "valid map",
			payload: map[string]interface{}{
				"container_id":   "abc123",
				"container_name": "nginx",
			},
			wantErr: false,
		},
		{
			name: "struct payload",
			payload: protocol.ContainerActionPayload{
				ContainerID:   "xyz789",
				ContainerName: "redis",
			},
			wantErr: false,
		},
		{
			name:    "string payload",
			payload: "invalid",
			wantErr: true,
		},
		{
			name:    "int payload",
			payload: 123,
			wantErr: true,
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: false, // parsePayload handles nil gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result protocol.ContainerActionPayload
			err := parsePayload(tt.payload, &result)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParsePayload_CreateContainer(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
		wantErr bool
	}{
		{
			name: "valid minimal",
			payload: map[string]interface{}{
				"name":  "test-container",
				"image": "nginx:latest",
			},
			wantErr: false,
		},
		{
			name: "with ports",
			payload: map[string]interface{}{
				"name":  "web-server",
				"image": "nginx:latest",
				"ports": map[string]string{"80/tcp": "8080"},
			},
			wantErr: false,
		},
		{
			name: "with volumes",
			payload: map[string]interface{}{
				"name":    "data-server",
				"image":   "postgres:14",
				"volumes": map[string]string{"/data": "/var/lib/postgresql"},
			},
			wantErr: false,
		},
		{
			name:    "invalid string",
			payload: "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result protocol.CreateContainerPayload
			err := parsePayload(tt.payload, &result)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParsePayload_UpdateAgent(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
		wantErr bool
	}{
		{
			name: "valid",
			payload: map[string]interface{}{
				"version": "1.2.3",
				"url":     "https://example.com/agent",
			},
			wantErr: false,
		},
		{
			name: "missing version",
			payload: map[string]interface{}{
				"url": "https://example.com/agent",
			},
			wantErr: false, // Still valid, just incomplete
		},
		{
			name:    "invalid type",
			payload: []string{"invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result protocol.UpdateAgentPayload
			err := parsePayload(tt.payload, &result)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParsePayload_EmptyMaps(t *testing.T) {
	tests := []struct {
		name    string
		target  interface{}
		wantErr bool
	}{
		{
			name:    "ContainerAction",
			target:  &protocol.ContainerActionPayload{},
			wantErr: false,
		},
		{
			name:    "CreateContainer",
			target:  &protocol.CreateContainerPayload{},
			wantErr: false,
		},
		{
			name:    "UpdateAgent",
			target:  &protocol.UpdateAgentPayload{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{}
			err := parsePayload(payload, tt.target)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParsePayload_NestedStructures(t *testing.T) {
	payload := map[string]interface{}{
		"name":  "complex-container",
		"image": "nginx:latest",
		"ports": map[string]string{
			"80/tcp":   "8080",
			"443/tcp":  "8443",
			"3000/tcp": "3000",
		},
		"volumes": map[string]string{
			"/var/www":  "/app",
			"/var/log":  "/logs",
			"/var/data": "/data",
		},
	}

	var result protocol.CreateContainerPayload
	err := parsePayload(payload, &result)
	
	if err != nil {
		t.Fatalf("parsePayload() error = %v", err)
	}

	if result.Name != "complex-container" {
		t.Errorf("Name = %v, want complex-container", result.Name)
	}

	if len(result.Ports) != 3 {
		t.Errorf("Ports length = %d, want 3", len(result.Ports))
	}

	if len(result.Volumes) != 3 {
		t.Errorf("Volumes length = %d, want 3", len(result.Volumes))
	}
}

func TestParsePayload_TypeConversions(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
		field   string
		value   interface{}
	}{
		{
			name:    "string to string",
			payload: map[string]interface{}{"container_id": "abc123"},
			field:   "container_id",
			value:   "abc123",
		},
		{
			name:    "int to string attempt",
			payload: map[string]interface{}{"container_id": 12345},
			field:   "container_id",
			value:   nil, // Will fail conversion
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result protocol.ContainerActionPayload
			err := parsePayload(tt.payload, &result)
			
			// Just test that it doesn't panic
			if err != nil {
				t.Logf("parsePayload() returned error (may be expected): %v", err)
			}
		})
	}
}

func TestParsePayload_LargePayloads(t *testing.T) {
	// Create payload with multiple ports
	ports := map[string]string{
		"80/tcp":   "8080",
		"443/tcp":  "8443",
		"3000/tcp": "3000",
		"5432/tcp": "5432",
		"6379/tcp": "6379",
		"8000/tcp": "8000",
		"9000/tcp": "9000",
		"3306/tcp": "3306",
		"27017/tcp": "27017",
		"5000/tcp": "5000",
	}
	
	payload := map[string]interface{}{
		"name":  "large-container",
		"image": "nginx:latest",
		"ports": ports,
	}

	var result protocol.CreateContainerPayload
	err := parsePayload(payload, &result)
	
	if err != nil {
		t.Fatalf("parsePayload() error = %v", err)
	}

	if len(result.Ports) != 10 {
		t.Errorf("Ports length = %d, want 10", len(result.Ports))
	}
}

func TestParsePayload_SpecialCharacters(t *testing.T) {
	payload := map[string]interface{}{
		"name":  "test-container-with-дashes-and-кириллица",
		"image": "nginx:latest-α-β-γ",
	}

	var result protocol.CreateContainerPayload
	err := parsePayload(payload, &result)
	
	if err != nil {
		t.Fatalf("parsePayload() error = %v", err)
	}

	if result.Name == "" {
		t.Error("Name is empty")
	}

	if result.Image == "" {
		t.Error("Image is empty")
	}
}

func TestParsePayload_BooleanValues(t *testing.T) {
	payload := map[string]interface{}{
		"container_id": "abc123",
		"auto_remove":  true,
		"privileged":   false,
	}

	var result map[string]interface{}
	err := parsePayload(payload, &result)
	
	if err != nil {
		t.Logf("parsePayload() error: %v", err)
	}
}

func TestParsePayload_NumericValues(t *testing.T) {
	payload := map[string]interface{}{
		"name":       "test",
		"cpu_shares": 1024,
		"memory":     512000000,
	}

	var result map[string]interface{}
	err := parsePayload(payload, &result)
	
	if err != nil {
		t.Logf("parsePayload() error: %v", err)
	}
}
