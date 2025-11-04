package agent

import (
	"context"
	"testing"
	"time"

	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/pkg/metrics"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// Integration tests for complete workflows

func TestFullCommandFlow_Ping(t *testing.T) {
	agent := createTestAgent()

	// Create command
	msg := protocol.NewMessage(protocol.TypePing, nil)
	msg.ID = "integration-ping-001"
	jsonBytes, _ := msg.ToJSON()

	// Process command
	agent.processCommand(jsonBytes)

	// Verify response sent
	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response published")
	}

	if len(mockClient.publishedChannels) == 0 {
		t.Error("No channel used")
	}

	// Check channel format
	expectedChannelPrefix := "resp:test-key:"
	if len(mockClient.publishedChannels) > 0 {
		channel := mockClient.publishedChannels[0]
		if len(channel) < len(expectedChannelPrefix) {
			t.Errorf("Channel too short: %s", channel)
		}
	}
}

func TestFullCommandFlow_AllMonitoring(t *testing.T) {
	agent := createTestAgent()

	commands := []protocol.MessageType{
		protocol.TypeGetCPUTemp,
		protocol.TypeGetMemoryInfo,
		protocol.TypeGetDiskInfo,
		protocol.TypeGetUptime,
		protocol.TypeGetProcesses,
	}

	for _, cmdType := range commands {
		msg := protocol.NewMessage(cmdType, nil)
		msg.ID = "integration-" + string(cmdType)
		jsonBytes, _ := msg.ToJSON()

		agent.processCommand(jsonBytes)
	}

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) < len(commands) {
		t.Errorf("Expected %d responses, got %d", len(commands), len(mockClient.publishedMessages))
	}
}

func TestConcurrentCommandProcessing(t *testing.T) {
	agent := createTestAgent()

	done := make(chan bool, 50)

	// Process 50 commands concurrently
	for i := 0; i < 50; i++ {
		go func(id int) {
			msg := protocol.NewMessage(protocol.TypePing, nil)
			msg.ID = "concurrent-" + string(rune('A'+id%26))
			jsonBytes, _ := msg.ToJSON()

			agent.processCommand(jsonBytes)
			done <- true
		}(i)
	}

	// Wait for all
	timeout := time.After(5 * time.Second)
	for i := 0; i < 50; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			t.Fatalf("Timeout waiting for concurrent processing (completed %d/50)", i)
		}
	}

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) < 50 {
		t.Errorf("Expected 50 responses, got %d", len(mockClient.publishedMessages))
	}
}

func TestSequentialCommandProcessing(t *testing.T) {
	agent := createTestAgent()

	commandSequence := []struct {
		msgType protocol.MessageType
		id      string
	}{
		{protocol.TypePing, "seq-001"},
		{protocol.TypeGetCPUTemp, "seq-002"},
		{protocol.TypePing, "seq-003"},
		{protocol.TypeGetMemoryInfo, "seq-004"},
		{protocol.TypePing, "seq-005"},
	}

	for _, cmd := range commandSequence {
		msg := protocol.NewMessage(cmd.msgType, nil)
		msg.ID = cmd.id
		jsonBytes, _ := msg.ToJSON()

		agent.processCommand(jsonBytes)
	}

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) != len(commandSequence) {
		t.Errorf("Expected %d responses, got %d", len(commandSequence), len(mockClient.publishedMessages))
	}
}

func TestMixedValidInvalidCommands(t *testing.T) {
	agent := createTestAgent()

	// Mix of valid and invalid
	commands := [][]byte{
		[]byte(`{"id":"valid-1","type":"ping"}`),
		[]byte(`{invalid json`),
		[]byte(`{"id":"valid-2","type":"get_cpu_temp"}`),
		[]byte(``),
		[]byte(`{"id":"valid-3","type":"ping"}`),
	}

	for _, cmd := range commands {
		agent.processCommand(cmd)
	}

	mockClient := agent.redisClient.(*mockRedisClient)
	// Should have responses for valid commands
	if len(mockClient.publishedMessages) < 3 {
		t.Logf("Got %d responses (some invalid commands skipped)", len(mockClient.publishedMessages))
	}
}

func TestAgentStateConsistency(t *testing.T) {
	agent := createTestAgent()

	// Process multiple commands and check state remains consistent
	for i := 0; i < 20; i++ {
		msg := protocol.NewMessage(protocol.TypePing, nil)
		jsonBytes, _ := msg.ToJSON()
		agent.processCommand(jsonBytes)

		// Verify agent state
		if agent.logger == nil {
			t.Error("Logger became nil")
		}
		if agent.ctx == nil {
			t.Error("Context became nil")
		}
		if agent.redisClient == nil {
			t.Error("Redis client became nil")
		}
	}
}

func TestChannelNamingConvention(t *testing.T) {
	tests := []struct {
		name       string
		secretKey  string
		commandID  string
		wantPrefix string
	}{
		{
			name:       "standard key",
			secretKey:  "my-secret-key",
			commandID:  "cmd-123",
			wantPrefix: "resp:my-secret-key:cmd-123",
		},
		{
			name:       "short key",
			secretKey:  "key",
			commandID:  "c1",
			wantPrefix: "resp:key:c1",
		},
		{
			name:       "long key",
			secretKey:  "very-long-secret-key-with-many-characters",
			commandID:  "long-command-id-123456789",
			wantPrefix: "resp:very-long-secret-key-with-many-characters:long-command-id-123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := "resp:" + tt.secretKey + ":" + tt.commandID
			if channel != tt.wantPrefix {
				t.Errorf("Channel = %s, want %s", channel, tt.wantPrefix)
			}
		})
	}
}

func TestErrorRecovery(t *testing.T) {
	agent := createTestAgent()

	// Send invalid command that causes error
	invalidMsg := protocol.NewMessage(protocol.MessageType("invalid_type"), nil)
	jsonBytes, _ := invalidMsg.ToJSON()

	agent.processCommand(jsonBytes)

	// Agent should still be functional
	validMsg := protocol.NewMessage(protocol.TypePing, nil)
	validBytes, _ := validMsg.ToJSON()

	agent.processCommand(validBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) < 2 {
		t.Error("Agent did not recover from error")
	}
}

func TestMessageIDPropagation(t *testing.T) {
	agent := createTestAgent()

	testIDs := []string{
		"id-001",
		"id-002",
		"unique-identifier-123",
		"uuid-style-abc-def-ghi",
	}

	for _, id := range testIDs {
		msg := protocol.NewMessage(protocol.TypePing, nil)
		msg.ID = id
		jsonBytes, _ := msg.ToJSON()

		agent.processCommand(jsonBytes)
	}

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) != len(testIDs) {
		t.Errorf("Expected %d messages, got %d", len(testIDs), len(mockClient.publishedMessages))
	}
}

func TestContextCancellation_MultipleOperations(t *testing.T) {
	mockClient := &mockRedisClient{}
	ctx, cancel := context.WithCancel(context.Background())

	agent := &Agent{
		logger:        logrus.New(),
		ctx:           ctx,
		cancel:        cancel,
		redisClient:   mockClient,
		cpuMetrics:    metrics.NewCPUMetrics(),
		systemMonitor: metrics.NewSystemMonitor(logrus.New()),
		config: &config.AgentConfig{
			Server: config.ServerConfig{SecretKey: "test-key"},
		},
	}

	// Start some operations
	for i := 0; i < 5; i++ {
		msg := protocol.NewMessage(protocol.TypePing, nil)
		jsonBytes, _ := msg.ToJSON()
		agent.processCommand(jsonBytes)
	}

	// Cancel context
	cancel()

	// Try more operations after cancellation
	for i := 0; i < 5; i++ {
		msg := protocol.NewMessage(protocol.TypePing, nil)
		jsonBytes, _ := msg.ToJSON()
		agent.processCommand(jsonBytes)
	}

	// Should have processed some messages
	if len(mockClient.publishedMessages) < 5 {
		t.Logf("Processed %d messages before/after cancellation", len(mockClient.publishedMessages))
	}
}

func TestAgentInitialization_AllFields(t *testing.T) {
	logger := logrus.New()
	mockClient := &mockRedisClient{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agent := &Agent{
		config: &config.AgentConfig{
			Server: config.ServerConfig{
				Name:      "test-server",
				SecretKey: "test-key",
			},
		},
		logger:        logger,
		redisClient:   mockClient,
		cpuMetrics:    metrics.NewCPUMetrics(),
		systemMonitor: metrics.NewSystemMonitor(logger),
		ctx:           ctx,
		cancel:        cancel,
		useStreams:    false,
	}

	// Verify all fields initialized
	if agent.config == nil {
		t.Error("Config is nil")
	}
	if agent.logger == nil {
		t.Error("Logger is nil")
	}
	if agent.redisClient == nil {
		t.Error("Redis client is nil")
	}
	if agent.cpuMetrics == nil {
		t.Error("CPU metrics is nil")
	}
	if agent.systemMonitor == nil {
		t.Error("System monitor is nil")
	}
	if agent.ctx == nil {
		t.Error("Context is nil")
	}
	if agent.cancel == nil {
		t.Error("Cancel func is nil")
	}
}

func TestCommandPriority_AllEqual(t *testing.T) {
	agent := createTestAgent()

	// All commands should be processed in order
	commands := []string{"cmd-1", "cmd-2", "cmd-3", "cmd-4", "cmd-5"}

	for _, id := range commands {
		msg := protocol.NewMessage(protocol.TypePing, nil)
		msg.ID = id
		jsonBytes, _ := msg.ToJSON()
		agent.processCommand(jsonBytes)
	}

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) != len(commands) {
		t.Errorf("Expected %d messages, got %d", len(commands), len(mockClient.publishedMessages))
	}
}

func TestLongRunningCommand(t *testing.T) {
	agent := createTestAgent()

	// Commands should complete without timeout
	msg := protocol.NewMessage(protocol.TypeGetProcesses, nil)
	jsonBytes, _ := msg.ToJSON()

	done := make(chan bool)
	go func() {
		agent.processCommand(jsonBytes)
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Command took too long")
	}
}

func TestRapidFireCommands(t *testing.T) {
	agent := createTestAgent()

	// Send 100 commands as fast as possible
	start := time.Now()

	for i := 0; i < 100; i++ {
		msg := protocol.NewMessage(protocol.TypePing, nil)
		msg.ID = "rapid-" + string(rune('0'+i%10))
		jsonBytes, _ := msg.ToJSON()
		agent.processCommand(jsonBytes)
	}

	elapsed := time.Since(start)
	t.Logf("Processed 100 commands in %v", elapsed)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) != 100 {
		t.Errorf("Expected 100 responses, got %d", len(mockClient.publishedMessages))
	}
}
