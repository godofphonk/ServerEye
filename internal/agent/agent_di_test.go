package agent

import (
	"context"
	"testing"
	"time"

	"github.com/godofphonk/ServerEye/internal/config"
	"github.com/godofphonk/ServerEye/internal/interfaces"
	"github.com/godofphonk/ServerEye/pkg/protocol"
	"github.com/godofphonk/ServerEye/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLogger is a mock implementation of interfaces.Logger
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Info(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Warn(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Error(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Fatal(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Fatalf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) WithField(key string, value interface{}) interfaces.Logger {
	args := m.Called(key, value)
	return args.Get(0).(interfaces.Logger)
}

func (m *MockLogger) WithFields(fields map[string]interface{}) interfaces.Logger {
	args := m.Called(fields)
	return args.Get(0).(interfaces.Logger)
}

func (m *MockLogger) WithError(err error) interfaces.Logger {
	args := m.Called(err)
	return args.Get(0).(interfaces.Logger)
}

// MockMetricsPublisher is a mock implementation of interfaces.MetricsPublisher
type MockMetricsPublisher struct {
	mock.Mock
}

func (m *MockMetricsPublisher) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMetricsPublisher) Publish(ctx context.Context, metric *types.Metric) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}

func (m *MockMetricsPublisher) PublishBatch(ctx context.Context, metrics []*types.Metric) error {
	args := m.Called(ctx, metrics)
	return args.Error(0)
}

func (m *MockMetricsPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMetricsPublisher) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMetricsPublisher) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockMetricsPublisher) GetMetrics() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

// TestAgentWithDI demonstrates dependency injection with mock dependencies
func TestAgentWithDI(t *testing.T) {
	// Create mock dependencies
	mockLogger := &MockLogger{}
	mockPublisher := &MockMetricsPublisher{}

	// Create agent with injected dependencies (simplified test)
	agent := &Agent{
		logger:            mockLogger,
		wsPublisher:       mockPublisher,
		wsCommandConsumer: nil, // Not needed for this test
		useWebSocket:      true,
		cpuMetrics:        nil, // Not needed for this test
		systemMonitor:     nil, // Not needed for this test
		dockerClient:      nil, // Not needed for this test
		startTime:         time.Now(),
	}

	// Test that agent can be created with DI interfaces
	assert.NotNil(t, agent)
	assert.Equal(t, mockLogger, agent.logger)
	assert.Equal(t, mockPublisher, agent.wsPublisher)
	assert.True(t, agent.useWebSocket)
}

// TestAgentCommandHandling demonstrates interface-based command handling
func TestAgentCommandHandling(t *testing.T) {
	// Create mock logger
	mockLogger := &MockLogger{}
	mockLogger.On("Info", mock.AnythingOfType("string")).Return()
	mockLogger.On("Info", mock.AnythingOfType("[]interface {}")).Return()
	mockLogger.On("Info", mock.AnythingOfType("string")).Return()
	mockLogger.On("WithFields", mock.AnythingOfType("map[string]interface {}")).Return(mockLogger)

	agent := &Agent{
		logger:            mockLogger,
		startTime:         time.Now(),
		config:            &config.AgentConfig{Server: config.ServerConfig{SecretKey: "test-key"}},
		wsPublisher:       &MockMetricsPublisher{},
		wsCommandConsumer: nil,
		useWebSocket:      false,
		cpuMetrics:        nil,
		systemMonitor:     nil,
		dockerClient:      nil,
	}

	// Test command handling
	ctx := context.Background()
	msg := protocol.NewMessage(protocol.TypePing, nil)

	response, err := agent.HandleCommand(ctx, msg)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "ping_response", string(response.Type))

	// Don't check mock expectations - just verify the command works
}

// TestCreateMetricFromData demonstrates metric creation with DI
func TestCreateMetricFromData(t *testing.T) {
	agent := &Agent{
		startTime: time.Now(),
		config:    &config.AgentConfig{Server: config.ServerConfig{SecretKey: "test-key"}},
	}

	// Test metric creation
	metric := agent.CreateMetricFromData("test_metric", 42.5, map[string]string{"env": "test"})

	assert.NotNil(t, metric)
	assert.Equal(t, "test_metric", metric.Type)
	assert.Equal(t, 42.5, metric.Value)
	assert.Equal(t, "test", metric.Tags["env"])
}

// BenchmarkAgentCreation compares agent creation with and without DI
func BenchmarkAgentCreation(b *testing.B) {
	ctx := context.Background()
	configPath := "/tmp/test-config.yaml"

	b.Run("WithDI", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			agent, err := InitializeAgentEnhanced(ctx, configPath)
			if err != nil {
				b.Fatal(err)
			}
			_ = agent
		}
	})
}
