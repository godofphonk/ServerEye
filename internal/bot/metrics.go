package bot

import (
	"errors"
	"sync"
	"time"
)

// InMemoryMetrics implements the Metrics interface with in-memory storage
type InMemoryMetrics struct {
	mu sync.RWMutex
	
	commandCounts map[string]int64
	errorCounts   map[string]int64
	latencies     map[string][]float64
	activeUsers   int64
	startTime     time.Time
}

// NewInMemoryMetrics creates a new in-memory metrics collector
func NewInMemoryMetrics() *InMemoryMetrics {
	return &InMemoryMetrics{
		commandCounts: make(map[string]int64),
		errorCounts:   make(map[string]int64),
		latencies:     make(map[string][]float64),
		startTime:     time.Now(),
	}
}

// IncrementCommand increments the counter for a specific command
func (m *InMemoryMetrics) IncrementCommand(command string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commandCounts[command]++
}

// IncrementError increments the counter for a specific error type
func (m *InMemoryMetrics) IncrementError(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCounts[errorType]++
}

// RecordLatency records the latency for a specific operation
func (m *InMemoryMetrics) RecordLatency(operation string, duration float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.latencies[operation] == nil {
		m.latencies[operation] = make([]float64, 0)
	}
	
	// Keep only last 1000 measurements to prevent memory leak
	if len(m.latencies[operation]) >= 1000 {
		m.latencies[operation] = m.latencies[operation][1:]
	}
	
	m.latencies[operation] = append(m.latencies[operation], duration)
}

// RecordActiveUsers records the current number of active users
func (m *InMemoryMetrics) RecordActiveUsers(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeUsers = count
}

// GetCommandCounts returns a copy of command counts
func (m *InMemoryMetrics) GetCommandCounts() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	counts := make(map[string]int64, len(m.commandCounts))
	for k, v := range m.commandCounts {
		counts[k] = v
	}
	return counts
}

// GetErrorCounts returns a copy of error counts
func (m *InMemoryMetrics) GetErrorCounts() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	counts := make(map[string]int64, len(m.errorCounts))
	for k, v := range m.errorCounts {
		counts[k] = v
	}
	return counts
}

// GetAverageLatency returns the average latency for an operation
func (m *InMemoryMetrics) GetAverageLatency(operation string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	latencies, exists := m.latencies[operation]
	if !exists || len(latencies) == 0 {
		return 0
	}
	
	var sum float64
	for _, latency := range latencies {
		sum += latency
	}
	
	return sum / float64(len(latencies))
}

// GetActiveUsers returns the current number of active users
func (m *InMemoryMetrics) GetActiveUsers() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeUsers
}

// GetUptime returns the bot uptime
func (m *InMemoryMetrics) GetUptime() time.Duration {
	return time.Since(m.startTime)
}

// GetStats returns comprehensive statistics
func (m *InMemoryMetrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := make(map[string]interface{})
	
	// Command statistics
	totalCommands := int64(0)
	for _, count := range m.commandCounts {
		totalCommands += count
	}
	stats["total_commands"] = totalCommands
	stats["command_counts"] = m.GetCommandCounts()
	
	// Error statistics
	totalErrors := int64(0)
	for _, count := range m.errorCounts {
		totalErrors += count
	}
	stats["total_errors"] = totalErrors
	stats["error_counts"] = m.GetErrorCounts()
	
	// Latency statistics
	avgLatencies := make(map[string]float64)
	for operation := range m.latencies {
		avgLatencies[operation] = m.GetAverageLatency(operation)
	}
	stats["average_latencies"] = avgLatencies
	
	// General statistics
	stats["active_users"] = m.activeUsers
	stats["uptime_seconds"] = m.GetUptime().Seconds()
	
	return stats
}

// MetricsMiddleware wraps command execution with metrics collection
func (m *InMemoryMetrics) MetricsMiddleware(command string, handler func() error) error {
	start := time.Now()
	
	// Increment command counter
	m.IncrementCommand(command)
	
	// Execute handler
	err := handler()
	
	// Record latency
	duration := time.Since(start).Seconds()
	m.RecordLatency(command, duration)
	
	// Record error if occurred
	if err != nil {
		var botErr *BotError
		if errors.As(err, &botErr) {
			m.IncrementError(botErr.Code)
		} else {
			m.IncrementError("UNKNOWN_ERROR")
		}
	}
	
	return err
}
