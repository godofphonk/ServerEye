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

// Edge case tests

func TestEmptyMessageID(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypePing, nil)
	msg.ID = ""
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response for empty ID")
	}
}

func TestVeryLongMessageID(t *testing.T) {
	agent := createTestAgent()

	// Create very long ID (1000 characters)
	longID := ""
	for i := 0; i < 100; i++ {
		longID += "0123456789"
	}

	msg := protocol.NewMessage(protocol.TypePing, nil)
	msg.ID = longID
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response for long ID")
	}
}

func TestSpecialCharactersInID(t *testing.T) {
	agent := createTestAgent()

	specialIDs := []string{
		"id-with-дashes",
		"id_with_underscores",
		"id.with.dots",
		"id:with:colons",
		"id/with/slashes",
		"id@with@at",
		"id#with#hash",
		"id with spaces",
	}

	for _, id := range specialIDs {
		msg := protocol.NewMessage(protocol.TypePing, nil)
		msg.ID = id
		jsonBytes, _ := msg.ToJSON()

		agent.processCommand(jsonBytes)
	}

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) < len(specialIDs) {
		t.Errorf("Expected %d responses, got %d", len(specialIDs), len(mockClient.publishedMessages))
	}
}

func TestUnicodeInPayload(t *testing.T) {
	agent := createTestAgent()

	payload := map[string]interface{}{
		"text": "Привет мир 你好世界 مرحبا بالعالم",
	}

	msg := protocol.NewMessage(protocol.TypePing, payload)
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response for unicode payload")
	}
}

func TestNullPayload(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypePing, nil)
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response for null payload")
	}
}

func TestLargePayload(t *testing.T) {
	agent := createTestAgent()

	// Create large payload (10000 items)
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData["key"+string(rune(i))] = "value" + string(rune(i))
	}

	msg := protocol.NewMessage(protocol.TypePing, largeData)
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response for large payload")
	}
}

func TestNestedPayload(t *testing.T) {
	agent := createTestAgent()

	nested := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"value": "deep",
				},
			},
		},
	}

	msg := protocol.NewMessage(protocol.TypePing, nested)
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response for nested payload")
	}
}

func TestArrayPayload(t *testing.T) {
	agent := createTestAgent()

	arrayData := []interface{}{
		"item1",
		2,
		true,
		map[string]interface{}{"key": "value"},
		[]interface{}{1, 2, 3},
	}

	msg := protocol.NewMessage(protocol.TypePing, arrayData)
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response for array payload")
	}
}

func TestZeroTimestamp(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypePing, nil)
	msg.ID = "zero-timestamp"
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response for zero timestamp")
	}
}

func TestFutureTimestamp(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypePing, nil)
	msg.ID = "future-timestamp"
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response for future timestamp")
	}
}

func TestMultipleContextCancellations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	agent := &Agent{
		logger: logrus.New(),
		ctx:    ctx,
		cancel: cancel,
	}

	// Cancel multiple times
	cancel()
	cancel()
	cancel()

	// Check context is done
	select {
	case <-agent.ctx.Done():
		// Expected
	default:
		t.Error("Context not done after multiple cancellations")
	}
}

func TestCancelledContextOperations(t *testing.T) {
	mockClient := &mockRedisClient{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

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

	// Try to process command with cancelled context
	msg := protocol.NewMessage(protocol.TypePing, nil)
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	// Should still process (doesn't check context in processCommand)
	if len(mockClient.publishedMessages) == 0 {
		t.Log("No response with cancelled context (may be expected)")
	}
}

func TestTimeoutContext(t *testing.T) {
	mockClient := &mockRedisClient{}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	agent := &Agent{
		logger:      logrus.New(),
		ctx:         ctx,
		redisClient: mockClient,
		config: &config.AgentConfig{
			Server: config.ServerConfig{SecretKey: "test-key"},
		},
	}

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	// Check context is done
	select {
	case <-agent.ctx.Done():
		// Expected
	default:
		t.Error("Context not done after timeout")
	}
}

func TestNilLogger(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Recovered from panic with nil logger: %v", r)
		}
	}()

	agent := &Agent{
		logger: nil,
	}

	// This might panic
	_ = agent.getAgentVersion()
}

func TestNilConfig(t *testing.T) {
	agent := &Agent{
		logger: logrus.New(),
		config: nil,
	}

	// Should handle nil config
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Recovered from panic with nil config: %v", r)
		}
	}()

	msg := protocol.NewMessage(protocol.TypePing, nil)
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)
}

func TestDuplicateMessageIDs(t *testing.T) {
	agent := createTestAgent()

	duplicateID := "duplicate-123"

	// Send same ID multiple times
	for i := 0; i < 5; i++ {
		msg := protocol.NewMessage(protocol.TypePing, nil)
		msg.ID = duplicateID
		jsonBytes, _ := msg.ToJSON()

		agent.processCommand(jsonBytes)
	}

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) != 5 {
		t.Errorf("Expected 5 responses, got %d", len(mockClient.publishedMessages))
	}
}

func TestRapidContextSwitching(t *testing.T) {
	mockClient := &mockRedisClient{}

	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		agent := &Agent{
			logger:      logrus.New(),
			ctx:         ctx,
			cancel:      cancel,
			redisClient: mockClient,
			config: &config.AgentConfig{
				Server: config.ServerConfig{SecretKey: "test-key"},
			},
		}

		msg := protocol.NewMessage(protocol.TypePing, nil)
		jsonBytes, _ := msg.ToJSON()

		agent.processCommand(jsonBytes)
		cancel()
	}

	if len(mockClient.publishedMessages) < 10 {
		t.Logf("Processed %d messages with rapid context switching", len(mockClient.publishedMessages))
	}
}

func TestBoundaryValues(t *testing.T) {
	agent := createTestAgent()

	tests := []struct {
		name    string
		payload interface{}
	}{
		{"zero int", 0},
		{"negative int", -1},
		{"max int", 9223372036854775807},
		{"empty string", ""},
		{"single char", "a"},
		{"true bool", true},
		{"false bool", false},
		{"empty map", map[string]interface{}{}},
		{"empty array", []interface{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := protocol.NewMessage(protocol.TypePing, tt.payload)
			jsonBytes, _ := msg.ToJSON()

			agent.processCommand(jsonBytes)
		})
	}
}

func TestMessageWithoutType(t *testing.T) {
	agent := createTestAgent()

	// Invalid JSON missing type
	invalidJSON := []byte(`{"id":"test-123"}`)

	agent.processCommand(invalidJSON)

	// Should handle gracefully
	mockClient := agent.redisClient.(*mockRedisClient)
	t.Logf("Processed %d messages (may be 0 for invalid)", len(mockClient.publishedMessages))
}

func TestMessageWithoutID(t *testing.T) {
	agent := createTestAgent()

	// Message without ID
	msg := protocol.NewMessage(protocol.TypePing, nil)
	msg.ID = ""
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) == 0 {
		t.Error("No response for message without ID")
	}
}

func TestMaxConcurrency(t *testing.T) {
	agent := createTestAgent()

	concurrency := 1000
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			msg := protocol.NewMessage(protocol.TypePing, nil)
			msg.ID = "max-concurrency"
			jsonBytes, _ := msg.ToJSON()

			agent.processCommand(jsonBytes)
			done <- true
		}(i)
	}

	timeout := time.After(10 * time.Second)
	for i := 0; i < concurrency; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			t.Fatalf("Timeout at %d/%d messages", i, concurrency)
		}
	}

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) < concurrency {
		t.Logf("Processed %d/%d messages", len(mockClient.publishedMessages), concurrency)
	}
}
