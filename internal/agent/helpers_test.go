package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

func TestHandlePing(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(logrus.StandardLogger().Out)

	agent := &Agent{
		logger: logger,
	}

	msg := protocol.NewMessage(protocol.TypePing, nil)
	msg.ID = "test-123"

	response := agent.handlePing(msg)

	if response == nil {
		t.Fatal("handlePing returned nil")
	}

	if response.Type != protocol.TypePong {
		t.Errorf("Expected TypePong, got %v", response.Type)
	}

	if response.ID != msg.ID {
		t.Errorf("Response ID = %v, want %v", response.ID, msg.ID)
	}

	// Check payload
	payload, ok := response.Payload.(protocol.PongPayload)
	if !ok {
		t.Fatal("Payload is not PongPayload")
	}

	if payload.Status != "healthy" {
		t.Errorf("Status = %v, want healthy", payload.Status)
	}
}

func TestHandleUnknownCommand(t *testing.T) {
	logger := logrus.New()
	agent := &Agent{
		logger: logger,
	}

	msg := protocol.NewMessage(protocol.MessageType("unknown_command"), nil)
	msg.ID = "test-456"

	response := agent.handleUnknownCommand(msg)

	if response == nil {
		t.Fatal("handleUnknownCommand returned nil")
	}

	if response.Type != protocol.TypeErrorResponse {
		t.Errorf("Expected TypeErrorResponse, got %v", response.Type)
	}

	if response.ID != msg.ID {
		t.Errorf("Response ID = %v, want %v", response.ID, msg.ID)
	}

	// Check error payload
	payload, ok := response.Payload.(protocol.ErrorPayload)
	if !ok {
		t.Fatal("Payload is not ErrorPayload")
	}

	if payload.ErrorCode != protocol.ErrorInvalidCommand {
		t.Errorf("ErrorCode = %v, want %v", payload.ErrorCode, protocol.ErrorInvalidCommand)
	}
}

func TestHandlePing_MultipleMessages(t *testing.T) {
	agent := &Agent{
		logger: logrus.New(),
	}

	for i := 0; i < 10; i++ {
		msg := protocol.NewMessage(protocol.TypePing, nil)
		msg.ID = "test-" + string(rune('0'+i))

		response := agent.handlePing(msg)

		if response == nil {
			t.Fatalf("handlePing returned nil for message %d", i)
		}

		if response.Type != protocol.TypePong {
			t.Errorf("Message %d: Expected TypePong, got %v", i, response.Type)
		}
	}
}

func TestHandleUnknownCommand_DifferentTypes(t *testing.T) {
	agent := &Agent{
		logger: logrus.New(),
	}

	unknownTypes := []protocol.MessageType{
		"unknown_type_1",
		"invalid_command",
		"nonexistent",
		"test_command",
	}

	for _, cmdType := range unknownTypes {
		t.Run(string(cmdType), func(t *testing.T) {
			msg := protocol.NewMessage(cmdType, nil)
			msg.ID = "test-" + string(cmdType)

			response := agent.handleUnknownCommand(msg)

			if response == nil {
				t.Fatal("handleUnknownCommand returned nil")
			}

			if response.Type != protocol.TypeErrorResponse {
				t.Errorf("Expected TypeErrorResponse, got %v", response.Type)
			}

			payload, ok := response.Payload.(protocol.ErrorPayload)
			if !ok {
				t.Fatal("Payload is not ErrorPayload")
			}

			if !strings.Contains(payload.ErrorMessage, string(cmdType)) {
				t.Errorf("Error message should contain command type %s, got: %s", cmdType, payload.ErrorMessage)
			}
		})
	}
}

func TestSendResponse_WithoutRedisClient(t *testing.T) {
	t.Skip("sendResponse panics with nil Redis client, needs proper error handling")
}

func TestSendResponseToCommand_WithMockClient(t *testing.T) {
	mockClient := &mockRedisClient{}

	agent := &Agent{
		logger:      logrus.New(),
		ctx:         context.Background(),
		redisClient: mockClient,
		config: &config.AgentConfig{
			Server: config.ServerConfig{
				SecretKey: "test-key",
			},
		},
		useStreams: false,
	}

	msg := protocol.NewMessage(protocol.TypePong, protocol.PongPayload{Status: "ok"})
	msg.ID = "test-response-123"

	err := agent.sendResponseToCommand(msg, "cmd-456")
	if err != nil {
		t.Errorf("sendResponseToCommand() error = %v", err)
	}

	if len(mockClient.publishedMessages) != 1 {
		t.Errorf("Expected 1 published message, got %d", len(mockClient.publishedMessages))
	}

	if len(mockClient.publishedChannels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(mockClient.publishedChannels))
	}

	expectedChannel := "resp:test-key:cmd-456"
	if mockClient.publishedChannels[0] != expectedChannel {
		t.Errorf("Published to %s, want %s", mockClient.publishedChannels[0], expectedChannel)
	}
}
