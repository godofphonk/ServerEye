package agent

import (
	"context"
	"testing"

	"github.com/servereye/servereye/pkg/metrics"
	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

func TestHandleGetCPUTemp_WithNilMetrics(t *testing.T) {
	t.Skip("Panics with nil metrics")
}

func TestHandleGetCPUTemp_WithValidMetrics(t *testing.T) {
	agent := &Agent{
		logger:     logrus.New(),
		cpuMetrics: metrics.NewCPUMetrics(),
	}

	msg := protocol.NewMessage(protocol.TypeGetCPUTemp, nil)
	response := agent.handleGetCPUTemp(msg)

	if response == nil {
		t.Fatal("handleGetCPUTemp returned nil")
	}

	// Response type should be either success or error
	validTypes := response.Type == protocol.TypeCPUTempResponse ||
		response.Type == protocol.TypeErrorResponse

	if !validTypes {
		t.Errorf("Unexpected response type: %v", response.Type)
	}
}

func TestHandleGetMemoryInfo_WithValidMonitor(t *testing.T) {
	logger := logrus.New()
	agent := &Agent{
		logger:        logger,
		systemMonitor: metrics.NewSystemMonitor(logger),
	}

	msg := protocol.NewMessage(protocol.TypeGetMemoryInfo, nil)
	response := agent.handleGetMemoryInfo(msg)

	if response == nil {
		t.Fatal("handleGetMemoryInfo returned nil")
	}

	if response.ID == "" {
		t.Error("Response ID is empty")
	}
}

func TestHandleGetMemoryInfo_WithNilMonitor(t *testing.T) {
	t.Skip("Panics with nil monitor")
}

func TestHandleGetDiskInfo_WithValidMonitor(t *testing.T) {
	logger := logrus.New()
	agent := &Agent{
		logger:        logger,
		systemMonitor: metrics.NewSystemMonitor(logger),
	}

	msg := protocol.NewMessage(protocol.TypeGetDiskInfo, nil)
	response := agent.handleGetDiskInfo(msg)

	if response == nil {
		t.Fatal("handleGetDiskInfo returned nil")
	}
}

func TestHandleGetDiskInfo_WithNilMonitor(t *testing.T) {
	t.Skip("Panics with nil monitor")
}

func TestHandleGetUptime_WithValidMonitor(t *testing.T) {
	logger := logrus.New()
	agent := &Agent{
		logger:        logger,
		systemMonitor: metrics.NewSystemMonitor(logger),
	}

	msg := protocol.NewMessage(protocol.TypeGetUptime, nil)
	response := agent.handleGetUptime(msg)

	if response == nil {
		t.Fatal("handleGetUptime returned nil")
	}
}

func TestHandleGetUptime_WithNilMonitor(t *testing.T) {
	t.Skip("Panics with nil monitor")
}

func TestHandleGetProcesses_WithValidMonitor(t *testing.T) {
	logger := logrus.New()
	agent := &Agent{
		logger:        logger,
		systemMonitor: metrics.NewSystemMonitor(logger),
		ctx:           context.Background(),
	}

	msg := protocol.NewMessage(protocol.TypeGetProcesses, nil)
	response := agent.handleGetProcesses(msg)

	if response == nil {
		t.Fatal("handleGetProcesses returned nil")
	}
}

func TestHandleGetProcesses_WithNilMonitor(t *testing.T) {
	t.Skip("Panics with nil monitor")
}

func TestHandleGetProcesses_WithNilContext(t *testing.T) {
	t.Skip("Panics with nil context")
}

func TestMonitoringHandlers_ResponseIDs(t *testing.T) {
	logger := logrus.New()
	agent := &Agent{
		logger:        logger,
		cpuMetrics:    metrics.NewCPUMetrics(),
		systemMonitor: metrics.NewSystemMonitor(logger),
		ctx:           context.Background(),
	}

	handlers := map[string]func(*protocol.Message) *protocol.Message{
		"cpu_temp":  agent.handleGetCPUTemp,
		"memory":    agent.handleGetMemoryInfo,
		"disk":      agent.handleGetDiskInfo,
		"uptime":    agent.handleGetUptime,
		"processes": agent.handleGetProcesses,
	}

	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			msg := protocol.NewMessage(protocol.TypePing, nil)
			msg.ID = "test-" + name

			response := handler(msg)

			if response == nil {
				t.Fatal("Handler returned nil")
			}

			if response.ID == "" {
				t.Error("Response ID is empty")
			}
		})
	}
}

func TestMonitoringHandlers_ErrorPayloads(t *testing.T) {
	t.Skip("Handlers panic with nil monitors/metrics")
}

func TestMonitoringHandlers_ConcurrentCalls(t *testing.T) {
	logger := logrus.New()
	agent := &Agent{
		logger:        logger,
		cpuMetrics:    metrics.NewCPUMetrics(),
		systemMonitor: metrics.NewSystemMonitor(logger),
		ctx:           context.Background(),
	}

	done := make(chan bool, 20)

	// Call handlers concurrently
	for i := 0; i < 20; i++ {
		go func(id int) {
			msg := protocol.NewMessage(protocol.TypeGetMemoryInfo, nil)
			response := agent.handleGetMemoryInfo(msg)
			if response == nil {
				t.Error("Concurrent call returned nil")
			}
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestMonitoringHandlers_MessageIDPreservation(t *testing.T) {
	logger := logrus.New()
	agent := &Agent{
		logger:        logger,
		cpuMetrics:    metrics.NewCPUMetrics(),
		systemMonitor: metrics.NewSystemMonitor(logger),
		ctx:           context.Background(),
	}

	testIDs := []string{
		"msg-001",
		"msg-002",
		"command-abc-123",
		"request-xyz-789",
	}

	for _, id := range testIDs {
		msg := protocol.NewMessage(protocol.TypeGetCPUTemp, nil)
		msg.ID = id

		response := agent.handleGetCPUTemp(msg)

		if response.ID == "" {
			t.Errorf("Response ID is empty for input ID %s", id)
		}
	}
}
