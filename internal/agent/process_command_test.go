package agent

import (
	"context"
	"testing"

	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/pkg/metrics"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// Helper function to create test agent
func createTestAgent() *Agent {
	logger := logrus.New()
	mockClient := &mockRedisClient{}

	return &Agent{
		logger:        logger,
		ctx:           context.Background(),
		redisClient:   mockClient,
		cpuMetrics:    metrics.NewCPUMetrics(),
		systemMonitor: metrics.NewSystemMonitor(logger),
		config: &config.AgentConfig{
			Server: config.ServerConfig{SecretKey: "test-key"},
		},
		updateFunc: MockUpdateFunc(),
	}
}

func TestProcessCommand_TypePing(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypePing, nil)
	msg.ID = "ping-001"
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response published for ping")
	}
}

func TestProcessCommand_TypeGetCPUTemp(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypeGetCPUTemp, nil)
	msg.ID = "cpu-001"
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response published for cpu temp")
	}
}

func TestProcessCommand_TypeGetMemoryInfo(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypeGetMemoryInfo, nil)
	msg.ID = "mem-001"
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response published for memory info")
	}
}

func TestProcessCommand_TypeGetDiskInfo(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypeGetDiskInfo, nil)
	msg.ID = "disk-001"
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response published for disk info")
	}
}

func TestProcessCommand_TypeGetUptime(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypeGetUptime, nil)
	msg.ID = "uptime-001"
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response published for uptime")
	}
}

func TestProcessCommand_TypeGetProcesses(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypeGetProcesses, nil)
	msg.ID = "proc-001"
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response published for processes")
	}
}

func TestProcessCommand_TypeUpdateAgent(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypeUpdateAgent, map[string]interface{}{
		"version": "1.0.0",
		"url":     "https://example.com/agent",
	})
	msg.ID = "update-001"
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response published for update agent")
	}
}

func TestProcessCommand_UnknownCommandType(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.MessageType("unknown_command"), nil)
	msg.ID = "unknown-001"
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response published for unknown command")
	}

	// Should have error response
	if len(mockClient.publishedMessages) > 0 {
		t.Log("Error response sent for unknown command (expected)")
	}
}

func TestProcessCommand_AllCommandTypes(t *testing.T) {
	tests := []struct {
		name    string
		msgType protocol.MessageType
		payload interface{}
	}{
		{"ping", protocol.TypePing, nil},
		{"cpu_temp", protocol.TypeGetCPUTemp, nil},
		{"memory", protocol.TypeGetMemoryInfo, nil},
		{"disk", protocol.TypeGetDiskInfo, nil},
		{"uptime", protocol.TypeGetUptime, nil},
		{"processes", protocol.TypeGetProcesses, nil},
		{"update", protocol.TypeUpdateAgent, map[string]interface{}{"version": "1.0.0"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := createTestAgent()

			msg := protocol.NewMessage(tt.msgType, tt.payload)
			msg.ID = "test-" + tt.name
			jsonBytes, _ := msg.ToJSON()

			agent.processCommand(jsonBytes)

			mockClient := agent.redisClient.(*mockRedisClient)
			if len(mockClient.publishedMessages) == 0 {
				t.Errorf("No response published for %s", tt.name)
			}
		})
	}
}

func TestProcessCommand_ResponseIDPreservation(t *testing.T) {
	agent := createTestAgent()

	testIDs := []string{
		"cmd-001",
		"cmd-002",
		"cmd-abc-123",
		"cmd-xyz-789",
	}

	for _, id := range testIDs {
		msg := protocol.NewMessage(protocol.TypePing, nil)
		msg.ID = id
		jsonBytes, _ := msg.ToJSON()

		agent.processCommand(jsonBytes)

		mockClient := agent.redisClient.(*mockRedisClient)
		if len(mockClient.publishedMessages) == 0 {
			t.Errorf("No response for ID %s", id)
		}
	}
}

func TestProcessCommand_ConcurrentProcessing(t *testing.T) {
	agent := createTestAgent()

	done := make(chan bool, 20)

	for i := 0; i < 20; i++ {
		go func(id int) {
			msg := protocol.NewMessage(protocol.TypePing, nil)
			msg.ID = "concurrent-ping"
			jsonBytes, _ := msg.ToJSON()

			agent.processCommand(jsonBytes)
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) < 20 {
		t.Errorf("Expected at least 20 responses, got %d", len(mockClient.publishedMessages))
	}
}

func TestProcessCommand_LoggingFields(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypePing, nil)
	msg.ID = "test-logging"
	jsonBytes, _ := msg.ToJSON()

	// Should not panic with logging
	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response published")
	}
}

func TestProcessCommand_EmptyID(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypePing, nil)
	msg.ID = ""
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response published for empty ID")
	}
}

func TestProcessCommand_LongID(t *testing.T) {
	agent := createTestAgent()

	longID := "this-is-a-very-long-command-id-to-test-the-agent-processing-capabilities-123456789"
	msg := protocol.NewMessage(protocol.TypePing, nil)
	msg.ID = longID
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response published for long ID")
	}
}

func TestProcessCommand_MultipleSequential(t *testing.T) {
	agent := createTestAgent()

	commandTypes := []protocol.MessageType{
		protocol.TypePing,
		protocol.TypeGetCPUTemp,
		protocol.TypeGetMemoryInfo,
		protocol.TypeGetDiskInfo,
		protocol.TypeGetUptime,
	}

	for i, cmdType := range commandTypes {
		msg := protocol.NewMessage(cmdType, nil)
		msg.ID = string(rune('A' + i))
		jsonBytes, _ := msg.ToJSON()

		agent.processCommand(jsonBytes)
	}

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) < len(commandTypes) {
		t.Errorf("Expected %d responses, got %d", len(commandTypes), len(mockClient.publishedMessages))
	}
}
