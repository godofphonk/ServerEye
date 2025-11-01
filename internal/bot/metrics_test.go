package bot

import (
	"errors"
	"testing"
	"time"
)

func TestNewInMemoryMetrics(t *testing.T) {
	metrics := NewInMemoryMetrics()

	if metrics == nil {
		t.Fatal("NewInMemoryMetrics() returned nil")
	}
	if metrics.commandCounts == nil {
		t.Error("commandCounts not initialized")
	}
	if metrics.errorCounts == nil {
		t.Error("errorCounts not initialized")
	}
	if metrics.latencies == nil {
		t.Error("latencies not initialized")
	}
	if metrics.startTime.IsZero() {
		t.Error("startTime not set")
	}
}

func TestIncrementCommand(t *testing.T) {
	metrics := NewInMemoryMetrics()

	metrics.IncrementCommand("/start")
	metrics.IncrementCommand("/start")
	metrics.IncrementCommand("/help")

	counts := metrics.GetCommandCounts()

	if counts["/start"] != 2 {
		t.Errorf("/start count = %d, want 2", counts["/start"])
	}
	if counts["/help"] != 1 {
		t.Errorf("/help count = %d, want 1", counts["/help"])
	}
}

func TestIncrementError(t *testing.T) {
	metrics := NewInMemoryMetrics()

	metrics.IncrementError("VALIDATION_ERROR")
	metrics.IncrementError("VALIDATION_ERROR")
	metrics.IncrementError("DATABASE_ERROR")

	counts := metrics.GetErrorCounts()

	if counts["VALIDATION_ERROR"] != 2 {
		t.Errorf("VALIDATION_ERROR count = %d, want 2", counts["VALIDATION_ERROR"])
	}
	if counts["DATABASE_ERROR"] != 1 {
		t.Errorf("DATABASE_ERROR count = %d, want 1", counts["DATABASE_ERROR"])
	}
}

func TestRecordLatency(t *testing.T) {
	metrics := NewInMemoryMetrics()

	metrics.RecordLatency("get_temp", 0.5)
	metrics.RecordLatency("get_temp", 1.5)

	avg := metrics.GetAverageLatency("get_temp")
	expected := 1.0

	if avg != expected {
		t.Errorf("Average latency = %f, want %f", avg, expected)
	}
}

func TestRecordLatency_MemoryLimit(t *testing.T) {
	metrics := NewInMemoryMetrics()

	// Record 1001 latencies to test memory limit
	for i := 0; i < 1001; i++ {
		metrics.RecordLatency("operation", float64(i))
	}

	metrics.mu.RLock()
	latencyCount := len(metrics.latencies["operation"])
	metrics.mu.RUnlock()

	if latencyCount > 1000 {
		t.Errorf("Latency count = %d, should be limited to 1000", latencyCount)
	}
}

func TestRecordActiveUsers(t *testing.T) {
	metrics := NewInMemoryMetrics()

	metrics.RecordActiveUsers(42)

	if got := metrics.GetActiveUsers(); got != 42 {
		t.Errorf("GetActiveUsers() = %d, want 42", got)
	}

	metrics.RecordActiveUsers(100)

	if got := metrics.GetActiveUsers(); got != 100 {
		t.Errorf("GetActiveUsers() = %d, want 100", got)
	}
}

func TestGetCommandCounts(t *testing.T) {
	metrics := NewInMemoryMetrics()

	metrics.IncrementCommand("/start")
	metrics.IncrementCommand("/help")

	counts := metrics.GetCommandCounts()

	// Verify it's a copy (modifying shouldn't affect original)
	counts["/start"] = 999

	originalCounts := metrics.GetCommandCounts()
	if originalCounts["/start"] != 1 {
		t.Error("GetCommandCounts should return a copy, not a reference")
	}
}

func TestGetErrorCounts(t *testing.T) {
	metrics := NewInMemoryMetrics()

	metrics.IncrementError("ERROR_1")
	metrics.IncrementError("ERROR_2")

	counts := metrics.GetErrorCounts()

	// Verify it's a copy
	counts["ERROR_1"] = 999

	originalCounts := metrics.GetErrorCounts()
	if originalCounts["ERROR_1"] != 1 {
		t.Error("GetErrorCounts should return a copy, not a reference")
	}
}

func TestGetAverageLatency_NoData(t *testing.T) {
	metrics := NewInMemoryMetrics()

	avg := metrics.GetAverageLatency("nonexistent")

	if avg != 0 {
		t.Errorf("Average latency for nonexistent operation = %f, want 0", avg)
	}
}

func TestGetAverageLatency_MultipleValues(t *testing.T) {
	metrics := NewInMemoryMetrics()

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	for _, v := range values {
		metrics.RecordLatency("test_op", v)
	}

	avg := metrics.GetAverageLatency("test_op")
	expected := 3.0

	if avg != expected {
		t.Errorf("Average latency = %f, want %f", avg, expected)
	}
}

func TestGetUptime(t *testing.T) {
	metrics := NewInMemoryMetrics()

	time.Sleep(10 * time.Millisecond)

	uptime := metrics.GetUptime()

	if uptime < 10*time.Millisecond {
		t.Errorf("Uptime = %v, should be at least 10ms", uptime)
	}
}

func TestGetStats(t *testing.T) {
	metrics := NewInMemoryMetrics()

	// Add some test data
	metrics.IncrementCommand("/start")
	metrics.IncrementCommand("/start")
	metrics.IncrementCommand("/help")
	metrics.IncrementError("ERROR_1")
	metrics.RecordLatency("op1", 1.5)
	metrics.RecordActiveUsers(50)

	stats := metrics.GetStats()

	// Verify total commands
	if total, ok := stats["total_commands"].(int64); !ok || total != 3 {
		t.Errorf("total_commands = %v, want 3", stats["total_commands"])
	}

	// Verify total errors
	if total, ok := stats["total_errors"].(int64); !ok || total != 1 {
		t.Errorf("total_errors = %v, want 1", stats["total_errors"])
	}

	// Verify active users
	if users, ok := stats["active_users"].(int64); !ok || users != 50 {
		t.Errorf("active_users = %v, want 50", stats["active_users"])
	}

	// Verify command counts exist
	if _, ok := stats["command_counts"]; !ok {
		t.Error("command_counts not in stats")
	}

	// Verify error counts exist
	if _, ok := stats["error_counts"]; !ok {
		t.Error("error_counts not in stats")
	}

	// Verify average latencies exist
	if _, ok := stats["average_latencies"]; !ok {
		t.Error("average_latencies not in stats")
	}

	// Verify uptime exists
	if uptime, ok := stats["uptime_seconds"].(float64); !ok || uptime < 0 {
		t.Error("uptime_seconds invalid")
	}
}

func TestMetricsMiddleware_Success(t *testing.T) {
	metrics := NewInMemoryMetrics()

	err := metrics.MetricsMiddleware("/test", func() error {
		time.Sleep(1 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("MetricsMiddleware() error = %v, want nil", err)
	}

	// Verify command was counted
	counts := metrics.GetCommandCounts()
	if counts["/test"] != 1 {
		t.Errorf("/test count = %d, want 1", counts["/test"])
	}

	// Verify latency was recorded
	avg := metrics.GetAverageLatency("/test")
	if avg <= 0 {
		t.Error("Latency should be recorded")
	}

	// Verify no errors recorded
	errorCounts := metrics.GetErrorCounts()
	if len(errorCounts) != 0 {
		t.Error("No errors should be recorded on success")
	}
}

func TestMetricsMiddleware_WithError(t *testing.T) {
	metrics := NewInMemoryMetrics()

	testErr := errors.New("test error")

	err := metrics.MetricsMiddleware("/test", func() error {
		return testErr
	})

	if err != testErr {
		t.Errorf("MetricsMiddleware() error = %v, want %v", err, testErr)
	}

	// Verify command was counted
	counts := metrics.GetCommandCounts()
	if counts["/test"] != 1 {
		t.Errorf("/test count = %d, want 1", counts["/test"])
	}

	// Verify error was recorded
	errorCounts := metrics.GetErrorCounts()
	if errorCounts["UNKNOWN_ERROR"] != 1 {
		t.Errorf("UNKNOWN_ERROR count = %d, want 1", errorCounts["UNKNOWN_ERROR"])
	}
}

func TestMetricsMiddleware_WithBotError(t *testing.T) {
	metrics := NewInMemoryMetrics()

	testErr := NewValidationError("test validation error", nil)

	err := metrics.MetricsMiddleware("/test", func() error {
		return testErr
	})

	if err != testErr {
		t.Errorf("MetricsMiddleware() error = %v, want %v", err, testErr)
	}

	// Verify error was recorded with correct code
	errorCounts := metrics.GetErrorCounts()
	if errorCounts[ErrCodeValidation] != 1 {
		t.Errorf("VALIDATION_ERROR count = %d, want 1", errorCounts[ErrCodeValidation])
	}
}

func TestMetrics_Concurrency(t *testing.T) {
	metrics := NewInMemoryMetrics()

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			metrics.IncrementCommand("/test")
			metrics.IncrementError("ERROR")
			metrics.RecordLatency("op", float64(id))
			metrics.RecordActiveUsers(int64(id))
			_ = metrics.GetCommandCounts()
			_ = metrics.GetErrorCounts()
			_ = metrics.GetAverageLatency("op")
			_ = metrics.GetStats()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify counts
	counts := metrics.GetCommandCounts()
	if counts["/test"] != 10 {
		t.Errorf("/test count = %d, want 10", counts["/test"])
	}

	errorCounts := metrics.GetErrorCounts()
	if errorCounts["ERROR"] != 10 {
		t.Errorf("ERROR count = %d, want 10", errorCounts["ERROR"])
	}
}
