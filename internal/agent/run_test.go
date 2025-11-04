package agent

import (
	"context"
	"testing"
	"time"

	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

func TestHandleCommands_NilMessage(t *testing.T) {
	agent := createTestAgent()

	msgChan := make(chan []byte, 1)
	msgChan <- nil
	close(msgChan)

	// Should exit gracefully
	agent.handleCommands(msgChan)
}

func TestHandleCommands_ValidMessage(t *testing.T) {
	agent := createTestAgent()

	msg := protocol.NewMessage(protocol.TypePing, nil)
	jsonBytes, _ := msg.ToJSON()

	msgChan := make(chan []byte, 1)
	msgChan <- jsonBytes

	// Start handling in goroutine
	done := make(chan bool)
	go func() {
		agent.handleCommands(msgChan)
		done <- true
	}()

	// Close channel to exit handler
	time.Sleep(50 * time.Millisecond)
	close(msgChan)

	// Wait for completion
	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("handleCommands did not exit")
	}
}

func TestHandleCommands_MultipleMessages(t *testing.T) {
	agent := createTestAgent()

	msgChan := make(chan []byte, 10)

	// Send multiple messages
	for i := 0; i < 5; i++ {
		msg := protocol.NewMessage(protocol.TypePing, nil)
		msg.ID = "msg-" + string(rune('A'+i))
		jsonBytes, _ := msg.ToJSON()
		msgChan <- jsonBytes
	}

	// Start handling
	done := make(chan bool)
	go func() {
		agent.handleCommands(msgChan)
		done <- true
	}()

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)
	close(msgChan)

	// Wait for completion
	select {
	case <-done:
		mockClient := agent.redisClient.(*mockRedisClient)
		if len(mockClient.publishedMessages) < 5 {
			t.Errorf("Expected 5 responses, got %d", len(mockClient.publishedMessages))
		}
	case <-time.After(2 * time.Second):
		t.Error("handleCommands did not exit")
	}
}

func TestHandleCommands_ContextCancellation(t *testing.T) {
	mockClient := &mockRedisClient{}
	ctx, cancel := context.WithCancel(context.Background())

	agent := &Agent{
		logger:      logrus.New(),
		ctx:         ctx,
		redisClient: mockClient,
		config: &config.AgentConfig{
			Server: config.ServerConfig{SecretKey: "test-key"},
		},
	}

	msgChan := make(chan []byte)

	// Start handling
	go agent.handleCommands(msgChan)

	// Cancel context
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Should exit
	time.Sleep(100 * time.Millisecond)
}

func TestHandleCommands_InvalidJSON(t *testing.T) {
	agent := createTestAgent()

	msgChan := make(chan []byte, 1)
	msgChan <- []byte("{invalid json}")

	done := make(chan bool)
	go func() {
		agent.handleCommands(msgChan)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	close(msgChan)

	select {
	case <-done:
		// Expected - should handle invalid JSON gracefully
	case <-time.After(1 * time.Second):
		t.Error("handleCommands did not exit")
	}
}

func TestHandleCommands_EmptyMessage(t *testing.T) {
	agent := createTestAgent()

	msgChan := make(chan []byte, 1)
	msgChan <- []byte("")

	done := make(chan bool)
	go func() {
		agent.handleCommands(msgChan)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	close(msgChan)

	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("handleCommands did not exit")
	}
}

func TestHandleCommands_RapidMessages(t *testing.T) {
	agent := createTestAgent()

	msgChan := make(chan []byte, 100)

	// Send many messages rapidly
	for i := 0; i < 50; i++ {
		msg := protocol.NewMessage(protocol.TypePing, nil)
		msg.ID = "rapid-" + string(rune('0'+i%10))
		jsonBytes, _ := msg.ToJSON()
		msgChan <- jsonBytes
	}

	done := make(chan bool)
	go func() {
		agent.handleCommands(msgChan)
		done <- true
	}()

	time.Sleep(200 * time.Millisecond)
	close(msgChan)

	select {
	case <-done:
		mockClient := agent.redisClient.(*mockRedisClient)
		if len(mockClient.publishedMessages) < 50 {
			t.Logf("Processed %d out of 50 messages", len(mockClient.publishedMessages))
		}
	case <-time.After(3 * time.Second):
		t.Error("handleCommands did not exit")
	}
}

func TestHandleCommands_MixedMessageTypes(t *testing.T) {
	agent := createTestAgent()

	msgChan := make(chan []byte, 10)

	messageTypes := []protocol.MessageType{
		protocol.TypePing,
		protocol.TypeGetCPUTemp,
		protocol.TypeGetMemoryInfo,
		protocol.TypeGetDiskInfo,
		protocol.TypeGetUptime,
	}

	for _, msgType := range messageTypes {
		msg := protocol.NewMessage(msgType, nil)
		jsonBytes, _ := msg.ToJSON()
		msgChan <- jsonBytes
	}

	done := make(chan bool)
	go func() {
		agent.handleCommands(msgChan)
		done <- true
	}()

	time.Sleep(200 * time.Millisecond)
	close(msgChan)

	select {
	case <-done:
		mockClient := agent.redisClient.(*mockRedisClient)
		if len(mockClient.publishedMessages) < len(messageTypes) {
			t.Errorf("Expected %d responses, got %d", len(messageTypes), len(mockClient.publishedMessages))
		}
	case <-time.After(2 * time.Second):
		t.Error("handleCommands did not exit")
	}
}

func TestStop_Basic(t *testing.T) {
	mockClient := &mockRedisClient{}
	ctx, cancel := context.WithCancel(context.Background())

	agent := &Agent{
		logger:      logrus.New(),
		ctx:         ctx,
		cancel:      cancel,
		redisClient: mockClient,
	}

	err := agent.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	// Context should be cancelled
	select {
	case <-agent.ctx.Done():
		// Expected
	default:
		t.Error("Context not cancelled after Stop()")
	}
}

func TestStop_MultipleCalls(t *testing.T) {
	mockClient := &mockRedisClient{}
	ctx, cancel := context.WithCancel(context.Background())

	agent := &Agent{
		logger:      logrus.New(),
		ctx:         ctx,
		cancel:      cancel,
		redisClient: mockClient,
	}

	// Call Stop multiple times
	for i := 0; i < 3; i++ {
		err := agent.Stop()
		if err != nil && i == 0 {
			t.Errorf("First Stop() error = %v", err)
		}
	}
}

func TestRun_Integration(t *testing.T) {
	t.Skip("Run() requires real Redis/HTTP connection")
}

func TestMessageProcessing_Pipeline(t *testing.T) {
	agent := createTestAgent()

	// Create a pipeline of messages
	messages := []struct {
		msgType protocol.MessageType
		payload interface{}
	}{
		{protocol.TypePing, nil},
		{protocol.TypeGetCPUTemp, nil},
		{protocol.TypeGetMemoryInfo, nil},
	}

	for _, m := range messages {
		msg := protocol.NewMessage(m.msgType, m.payload)
		jsonBytes, _ := msg.ToJSON()
		agent.processCommand(jsonBytes)
	}

	mockClient := agent.redisClient.(*mockRedisClient)
	if len(mockClient.publishedMessages) < len(messages) {
		t.Errorf("Expected %d responses, got %d", len(messages), len(mockClient.publishedMessages))
	}
}

func TestChannelClosed_Handling(t *testing.T) {
	agent := createTestAgent()

	msgChan := make(chan []byte)
	close(msgChan) // Close immediately

	// Should exit without hanging
	done := make(chan bool)
	go func() {
		agent.handleCommands(msgChan)
		done <- true
	}()

	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("handleCommands did not exit when channel closed")
	}
}

func TestSelectCase_ContextAndMessage(t *testing.T) {
	agent := createTestAgent()

	msgChan := make(chan []byte, 1)
	msg := protocol.NewMessage(protocol.TypePing, nil)
	jsonBytes, _ := msg.ToJSON()
	msgChan <- jsonBytes

	done := make(chan bool)
	go func() {
		select {
		case m := <-msgChan:
			agent.processCommand(m)
		case <-agent.ctx.Done():
			// Context cancelled
		}
		done <- true
	}()

	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("Select did not complete")
	}
}
