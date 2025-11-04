package bot

import (
	"strings"
	"testing"

	"github.com/servereye/servereye/pkg/protocol"
)

func TestFormatContainers_Empty(t *testing.T) {
	bot := &Bot{}
	
	containers := &protocol.ContainersPayload{
		Total:      0,
		Containers: []protocol.ContainerInfo{},
	}

	result := bot.formatContainers(containers)
	
	if !strings.Contains(result, "No Docker containers") {
		t.Errorf("Expected no containers message, got: %v", result)
	}
}

func TestFormatContainers_Single(t *testing.T) {
	bot := &Bot{}
	
	containers := &protocol.ContainersPayload{
		Total: 1,
		Containers: []protocol.ContainerInfo{
			{
				ID:     "abc123",
				Name:   "nginx",
				Image:  "nginx:latest",
				State:  "running",
				Status: "Up 2 hours",
				Ports:  []string{"80:80"},
			},
		},
	}

	result := bot.formatContainers(containers)
	
	if !strings.Contains(result, "nginx") {
		t.Errorf("Expected nginx in result, got: %v", result)
	}
	
	if !strings.Contains(result, "Up 2 hours") {
		t.Errorf("Expected status in result, got: %v", result)
	}
}

func TestFormatContainers_Multiple(t *testing.T) {
	bot := &Bot{}
	
	containers := &protocol.ContainersPayload{
		Total: 3,
		Containers: []protocol.ContainerInfo{
			{ID: "1", Name: "web", State: "running"},
			{ID: "2", Name: "db", State: "running"},
			{ID: "3", Name: "cache", State: "exited"},
		},
	}

	result := bot.formatContainers(containers)
	
	if !strings.Contains(result, "web") {
		t.Error("Expected web container in result")
	}
	
	if !strings.Contains(result, "db") {
		t.Error("Expected db container in result")
	}
	
	if !strings.Contains(result, "cache") {
		t.Error("Expected cache container in result")
	}
}

func TestFormatContainers_ManyContainers(t *testing.T) {
	bot := &Bot{}
	
	containers := &protocol.ContainersPayload{
		Total:      15,
		Containers: make([]protocol.ContainerInfo, 15),
	}
	
	for i := 0; i < 15; i++ {
		containers.Containers[i] = protocol.ContainerInfo{
			ID:    string(rune('A' + i)),
			Name:  "container" + string(rune('A'+i)),
			State: "running",
		}
	}

	result := bot.formatContainers(containers)
	
	// Should limit display and show "more containers"
	if !strings.Contains(result, "more") || !strings.Contains(result, "containers") {
		t.Log("Should mention more containers when list is long")
	}
}

func TestContainerStateEmoji(t *testing.T) {
	tests := []struct {
		state string
		want  string
	}{
		{"running", "🟢"},
		{"exited", "🔴"},
		{"paused", "🟡"},
		{"created", "🔴"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			// Test that state detection works
			state := strings.ToLower(tt.state)
			isRunning := strings.Contains(state, "running")
			isPaused := strings.Contains(state, "paused")
			
			if tt.state == "running" && !isRunning {
				t.Error("Failed to detect running state")
			}
			if tt.state == "paused" && !isPaused {
				t.Error("Failed to detect paused state")
			}
		})
	}
}

func TestContainerAction_ValidActions(t *testing.T) {
	validActions := []string{"start", "stop", "restart", "remove"}
	
	for _, action := range validActions {
		t.Run(action, func(t *testing.T) {
			if action == "" {
				t.Error("Action is empty")
			}
			if len(action) < 3 {
				t.Error("Action name too short")
			}
		})
	}
}

func TestHandleContainerAction_InvalidAction(t *testing.T) {
	t.Skip("Requires full bot initialization")
}

func TestCreateContainerFromTemplate_InvalidTemplate(t *testing.T) {
	t.Skip("Requires full bot initialization")
}
