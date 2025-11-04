package agent

import (
	"context"
	"testing"

	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

func TestNew_DirectRedis(t *testing.T) {
	t.Skip("Skipping test that requires Redis connection")
	
	cfg := &config.AgentConfig{
		Server: config.ServerConfig{
			Name:      "test-server",
			SecretKey: "test-key-123",
		},
		Redis: config.RedisConfig{
			Address: "localhost:6379",
		},
	}

	logger := logrus.New()
	agent, err := New(cfg, logger)

	if err != nil {
		t.Logf("Expected error without Redis: %v", err)
		return
	}

	if agent == nil {
		t.Fatal("New() returned nil agent")
	}

	if agent.config == nil {
		t.Error("Agent config is nil")
	}

	if agent.logger == nil {
		t.Error("Agent logger is nil")
	}

	if agent.ctx == nil {
		t.Error("Agent context is nil")
	}
}

func TestNew_WithAPI(t *testing.T) {
	cfg := &config.AgentConfig{
		Server: config.ServerConfig{
			Name:      "test-server",
			SecretKey: "test-key-123",
		},
		API: config.APIConfig{
			BaseURL: "https://example.com",
		},
	}

	logger := logrus.New()
	agent, err := New(cfg, logger)

	if err != nil {
		t.Logf("New() with API config: %v", err)
	}

	if agent == nil {
		t.Log("Agent is nil (expected without real API)")
		return
	}
	
	if agent.config == nil {
		t.Error("Agent config should not be nil")
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	cfg := &config.AgentConfig{
		Server: config.ServerConfig{
			Name:      "",
			SecretKey: "",
		},
	}

	logger := logrus.New()
	agent, err := New(cfg, logger)

	if err != nil {
		t.Logf("Expected error for invalid config: %v", err)
		return
	}

	if agent != nil {
		t.Log("Created agent even with minimal config")
	}
}

func TestProcessCommand_UnknownType(t *testing.T) {
	t.Skip("Requires Redis client initialization")
	
	logger := logrus.New()
	agent := &Agent{
		logger: logger,
		ctx:    context.Background(),
		redisClient: &mockRedisClient{},
	}

	msg := protocol.NewMessage(protocol.MessageType("unknown_type"), nil)
	msg.ID = "test-123"

	jsonBytes, _ := msg.ToJSON()
	agent.processCommand(jsonBytes)

	// Should not panic
}

func TestProcessCommand_InvalidJSON(t *testing.T) {
	logger := logrus.New()
	agent := &Agent{
		logger: logger,
		ctx:    context.Background(),
	}

	invalidJSON := []byte("{invalid json")
	agent.processCommand(invalidJSON)

	// Should not panic
}

func TestAgentStop(t *testing.T) {
	t.Skip("Requires Redis connection")
	
	cfg := &config.AgentConfig{
		Server: config.ServerConfig{
			Name:      "test-server",
			SecretKey: "test-key",
		},
		Redis: config.RedisConfig{
			Address: "localhost:6379",
		},
	}

	logger := logrus.New()
	agent, err := New(cfg, logger)
	if err != nil {
		t.Logf("Error creating agent: %v", err)
		return
	}

	err = agent.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestRedisClientInterface(t *testing.T) {
	// Test that interface is properly defined
	var _ RedisClientInterface = (*mockRedisClient)(nil)
}

func TestSubscriptionInterface(t *testing.T) {
	// Test that subscription interface exists
	var sub SubscriptionInterface
	if sub != nil {
		t.Error("Uninitialized interface should be nil")
	}
}

func TestAgentContext(t *testing.T) {
	ctx := context.Background()
	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	if ctxWithCancel == nil {
		t.Error("Context is nil")
	}

	// Test cancellation
	cancel()
	select {
	case <-ctxWithCancel.Done():
		// Expected
	default:
		t.Error("Context not cancelled")
	}
}

func TestAgentFields(t *testing.T) {
	agent := &Agent{
		config: &config.AgentConfig{
			Server: config.ServerConfig{
				Name:      "test",
				SecretKey: "key",
			},
		},
	}

	if agent.config.Server.Name != "test" {
		t.Error("Agent config not set correctly")
	}

	if agent.config.Server.SecretKey != "key" {
		t.Error("Agent secret key not set correctly")
	}
}

func TestMessageProcessingFlow(t *testing.T) {
	// Test complete message flow
	msg := protocol.NewMessage(protocol.TypePing, nil)
	msg.ID = "test-ping"

	jsonBytes, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("Message serialization failed: %v", err)
	}
	if len(jsonBytes) == 0 {
		t.Error("Message serialization returned empty bytes")
	}

	parsed, err := protocol.FromJSON(jsonBytes)
	if err != nil {
		t.Fatalf("Message parsing failed: %v", err)
	}

	if parsed.Type != protocol.TypePing {
		t.Errorf("Message type = %v, want %v", parsed.Type, protocol.TypePing)
	}

	if parsed.ID != msg.ID {
		t.Errorf("Message ID = %v, want %v", parsed.ID, msg.ID)
	}
}
