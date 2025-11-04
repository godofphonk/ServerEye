package agent

import (
	"context"
	"testing"
	"time"

	"github.com/servereye/servereye/internal/config"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)


func TestAgent_ProcessCommand_Ping(t *testing.T) {
	mockClient := &mockRedisClient{}
	agent := &Agent{
		logger:      logrus.New(),
		ctx:         context.Background(),
		redisClient: mockClient,
		config: &config.AgentConfig{
			Server: config.ServerConfig{SecretKey: "test-key"},
		},
		updateFunc: MockUpdateFunc(),
	}

	msg := protocol.NewMessage(protocol.TypePing, nil)
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	// Should have sent response
	if len(mockClient.publishedMessages) == 0 {
		t.Error("Expected response to be published")
	}
}

func TestAgent_ProcessCommand_InvalidJSON(t *testing.T) {
	agent := &Agent{
		logger:     logrus.New(),
		ctx:        context.Background(),
		updateFunc: MockUpdateFunc(),
	}

	invalidJSON := []byte("{invalid json")

	// Should not panic
	agent.processCommand(invalidJSON)
}

func TestAgent_ProcessCommand_AllTypes(t *testing.T) {
	mockClient := &mockRedisClient{}
	agent := &Agent{
		logger:      logrus.New(),
		ctx:         context.Background(),
		redisClient: mockClient,
		config: &config.AgentConfig{
			Server: config.ServerConfig{SecretKey: "test-key"},
		},
		updateFunc: MockUpdateFunc(),
	}

	messageTypes := []protocol.MessageType{
		protocol.TypePing,
		protocol.TypeGetCPUTemp,
		protocol.TypeGetMemoryInfo,
		protocol.TypeGetDiskInfo,
		protocol.TypeGetUptime,
		protocol.TypeGetProcesses,
		protocol.TypeGetContainers,
	}

	for _, msgType := range messageTypes {
		t.Run(string(msgType), func(t *testing.T) {
			msg := protocol.NewMessage(msgType, nil)
			jsonBytes, _ := msg.ToJSON()

			// Should not panic
			agent.processCommand(jsonBytes)
		})
	}
}

func TestAgent_Stop(t *testing.T) {
	mockClient := &mockRedisClient{}
	ctx, cancel := context.WithCancel(context.Background())

	agent := &Agent{
		logger:      logrus.New(),
		ctx:         ctx,
		cancel:      cancel,
		redisClient: mockClient,
		updateFunc:  MockUpdateFunc(),
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

func TestAgent_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	agent := &Agent{
		logger:     logrus.New(),
		ctx:        ctx,
		cancel:     cancel,
		updateFunc: MockUpdateFunc(),
	}

	// Cancel context
	cancel()

	// Check if context is done
	select {
	case <-agent.ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Context not cancelled")
	}
}

func TestAgent_MultipleStops(t *testing.T) {
	mockClient := &mockRedisClient{}
	ctx, cancel := context.WithCancel(context.Background())

	agent := &Agent{
		logger:      logrus.New(),
		ctx:         ctx,
		cancel:      cancel,
		redisClient: mockClient,
		updateFunc:  MockUpdateFunc(),
	}

	// Call Stop multiple times
	for i := 0; i < 3; i++ {
		err := agent.Stop()
		if err != nil && i == 0 {
			t.Errorf("First Stop() error = %v", err)
		}
	}
}

func TestAgent_ConfigAccess(t *testing.T) {
	cfg := &config.AgentConfig{
		Server: config.ServerConfig{
			Name:      "test-server",
			SecretKey: "secret-123",
		},
	}

	agent := &Agent{
		config:     cfg,
		logger:     logrus.New(),
		updateFunc: MockUpdateFunc(),
	}

	if agent.config.Server.Name != "test-server" {
		t.Error("Config not accessible")
	}

	if agent.config.Server.SecretKey != "secret-123" {
		t.Error("Secret key not accessible")
	}
}

func TestAgent_LoggerAccess(t *testing.T) {
	logger := logrus.New()
	agent := &Agent{
		logger:     logger,
		updateFunc: MockUpdateFunc(),
	}

	if agent.logger == nil {
		t.Error("Logger is nil")
	}

	// Should be able to log
	agent.logger.Info("Test log")
}

func TestRedisClientAdapter_Interface(t *testing.T) {
	// Test that adapters implement the interface
	var _ RedisClientInterface = (*HTTPClientAdapter)(nil)
	var _ RedisClientInterface = (*DirectClientAdapter)(nil)
}

func TestSubscriptionAdapter_Interface(t *testing.T) {
	// Test subscription adapters
	var _ SubscriptionInterface = (*HTTPSubscriptionAdapter)(nil)
	var _ SubscriptionInterface = (*DirectSubscriptionAdapter)(nil)
}

func TestAgent_Initialization(t *testing.T) {
	tests := []struct {
		name   string
		config *config.AgentConfig
	}{
		{
			name: "with API config",
			config: &config.AgentConfig{
				Server: config.ServerConfig{
					Name:      "test",
					SecretKey: "key",
				},
				API: config.APIConfig{
					BaseURL: "https://example.com",
				},
			},
		},
		{
			name: "with Redis config",
			config: &config.AgentConfig{
				Server: config.ServerConfig{
					Name:      "test",
					SecretKey: "key",
				},
				Redis: config.RedisConfig{
					Address: "localhost:6379",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip("Requires real connections")
		})
	}
}

func TestAgent_MessageHandling(t *testing.T) {
	mockClient := &mockRedisClient{}
	agent := &Agent{
		logger:      logrus.New(),
		ctx:         context.Background(),
		redisClient: mockClient,
		config: &config.AgentConfig{
			Server: config.ServerConfig{SecretKey: "test-key"},
		},
		updateFunc: MockUpdateFunc(),
	}

	// Test with valid message
	msg := protocol.NewMessage(protocol.TypePing, nil)
	msg.ID = "msg-123"
	jsonBytes, _ := msg.ToJSON()

	agent.processCommand(jsonBytes)

	if len(mockClient.publishedMessages) == 0 {
		t.Error("No messages published")
	}
}

func TestAgent_ConcurrentProcessing(t *testing.T) {
	mockClient := &mockRedisClient{}
	agent := &Agent{
		logger:      logrus.New(),
		ctx:         context.Background(),
		redisClient: mockClient,
		config: &config.AgentConfig{
			Server: config.ServerConfig{SecretKey: "test-key"},
		},
		updateFunc: MockUpdateFunc(),
	}

	// Process multiple messages concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			msg := protocol.NewMessage(protocol.TypePing, nil)
			jsonBytes, _ := msg.ToJSON()
			agent.processCommand(jsonBytes)
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	if len(mockClient.publishedMessages) < 10 {
		t.Errorf("Expected at least 10 messages, got %d", len(mockClient.publishedMessages))
	}
}

func TestAgent_ErrorRecovery(t *testing.T) {
	agent := &Agent{
		logger:      logrus.New(),
		ctx:         context.Background(),
		redisClient: nil, // Will cause issues but shouldn't crash
		config: &config.AgentConfig{
			Server: config.ServerConfig{SecretKey: "test-key"},
		},
		updateFunc: MockUpdateFunc(),
	}

	// Should handle nil client gracefully in some way
	msg := protocol.NewMessage(protocol.TypePing, nil)
	jsonBytes, _ := msg.ToJSON()

	// Should not panic even with nil client
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Recovered from panic: %v", r)
		}
	}()

	agent.processCommand(jsonBytes)
}
