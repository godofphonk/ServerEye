package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/servereye/servereye/internal/config"
	"github.com/sirupsen/logrus"
)

type mockRedisClient struct {
	mu                sync.Mutex
	publishedMessages []string
	publishedChannels []string
}

func (m *mockRedisClient) Subscribe(ctx context.Context, channel string) (SubscriptionInterface, error) {
	return nil, nil
}

func (m *mockRedisClient) Publish(ctx context.Context, channel string, message []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishedChannels = append(m.publishedChannels, channel)
	m.publishedMessages = append(m.publishedMessages, string(message))
	return nil
}

func (m *mockRedisClient) Close() error {
	return nil
}

func TestSendHeartbeat(t *testing.T) {
	mockRedis := &mockRedisClient{}
	logger := logrus.New()

	agent := &Agent{
		redisClient: mockRedis,
		logger:      logger,
		ctx:         context.Background(),
		config: &config.AgentConfig{
			Server: config.ServerConfig{
				SecretKey: "test-key",
				Name:      "test-server",
			},
		},
	}

	agent.sendHeartbeat()

	if len(mockRedis.publishedChannels) != 1 {
		t.Errorf("Expected 1 published message, got %d", len(mockRedis.publishedChannels))
	}

	if len(mockRedis.publishedChannels) > 0 {
		expectedChannel := "heartbeat:test-key"
		if mockRedis.publishedChannels[0] != expectedChannel {
			t.Errorf("Published to channel %v, want %v", mockRedis.publishedChannels[0], expectedChannel)
		}
	}

	if len(mockRedis.publishedMessages) > 0 {
		message := mockRedis.publishedMessages[0]
		if message == "" {
			t.Error("Published message is empty")
		}
	}
}

func TestStartHeartbeat_Cancellation(t *testing.T) {
	mockRedis := &mockRedisClient{}
	logger := logrus.New()
	ctx, cancel := context.WithCancel(context.Background())

	agent := &Agent{
		redisClient: mockRedis,
		logger:      logger,
		ctx:         ctx,
		config: &config.AgentConfig{
			Server: config.ServerConfig{
				SecretKey: "test-key",
				Name:      "test-server",
			},
		},
	}

	// Start heartbeat in goroutine
	go agent.startHeartbeat()

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for goroutine to finish
	time.Sleep(50 * time.Millisecond)

	// Should have sent at least one heartbeat or none if cancelled too fast
	// This is just to test it doesn't panic
}
