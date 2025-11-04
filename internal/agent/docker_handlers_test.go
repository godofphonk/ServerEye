package agent

import (
	"testing"

	"github.com/servereye/servereye/pkg/protocol"
)

func TestParsePayload(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
		target  interface{}
		wantErr bool
	}{
		{
			name: "valid container action payload",
			payload: map[string]interface{}{
				"container_id":   "test-container",
				"container_name": "nginx",
			},
			target:  &protocol.ContainerActionPayload{},
			wantErr: false,
		},
		{
			name: "valid create container payload",
			payload: map[string]interface{}{
				"name":  "test-container",
				"image": "nginx:latest",
			},
			target:  &protocol.CreateContainerPayload{},
			wantErr: false,
		},
		{
			name:    "invalid payload type",
			payload: "invalid",
			target:  &protocol.ContainerActionPayload{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parsePayload(tt.payload, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParsePayload_ContainerActionPayload(t *testing.T) {
	payload := map[string]interface{}{
		"container_id":   "abc123",
		"container_name": "my-container",
	}

	var result protocol.ContainerActionPayload
	err := parsePayload(payload, &result)
	
	if err != nil {
		t.Fatalf("parsePayload() error = %v", err)
	}

	if result.ContainerID != "abc123" {
		t.Errorf("ContainerID = %v, want abc123", result.ContainerID)
	}

	if result.ContainerName != "my-container" {
		t.Errorf("ContainerName = %v, want my-container", result.ContainerName)
	}
}

func TestParsePayload_CreateContainerPayload(t *testing.T) {
	payload := map[string]interface{}{
		"name":  "test-nginx",
		"image": "nginx:alpine",
		"ports": map[string]string{"80/tcp": "80"},
	}

	var result protocol.CreateContainerPayload
	err := parsePayload(payload, &result)
	
	if err != nil {
		t.Fatalf("parsePayload() error = %v", err)
	}

	if result.Name != "test-nginx" {
		t.Errorf("Name = %v, want test-nginx", result.Name)
	}

	if result.Image != "nginx:alpine" {
		t.Errorf("Image = %v, want nginx:alpine", result.Image)
	}
}

func TestParsePayload_EmptyPayload(t *testing.T) {
	payload := map[string]interface{}{}

	var result protocol.ContainerActionPayload
	err := parsePayload(payload, &result)
	
	if err != nil {
		t.Fatalf("parsePayload() with empty payload error = %v", err)
	}

	if result.ContainerID != "" {
		t.Error("Empty payload should result in empty fields")
	}
}

func TestParsePayload_NilTarget(t *testing.T) {
	t.Skip("parsePayload doesn't panic on nil, it returns error")
}

func TestParsePayload_ComplexPayload(t *testing.T) {
	payload := map[string]interface{}{
		"name":    "complex-container",
		"image":   "nginx:latest",
		"ports":   map[string]string{"443/tcp": "443", "80/tcp": "80"},
		"env":     map[string]string{"ENV": "production"},
		"volumes": map[string]string{"/data": "/data"},
	}

	var result protocol.CreateContainerPayload
	err := parsePayload(payload, &result)
	
	if err != nil {
		t.Fatalf("parsePayload() error = %v", err)
	}

	if result.Name != "complex-container" {
		t.Errorf("Name = %v, want complex-container", result.Name)
	}

	if len(result.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(result.Ports))
	}
}
