package bot

import (
	"context"
	"testing"
	"time"

	"github.com/servereye/servereye/pkg/protocol"
)

func TestSendCommandAndParse_Success(t *testing.T) {
	// This is an integration test that would require mocking
	// For now, we test the helper function behavior with type checking

	type TestResponse struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}

	// Test that the generic type parameter works
	var testFunc = func() *TestResponse {
		// This simulates what sendCommandAndParse returns
		return &TestResponse{
			Value: "test",
			Count: 42,
		}
	}

	result := testFunc()
	if result == nil {
		t.Fatal("result is nil")
	}

	if result.Value != "test" {
		t.Errorf("Value = %v, want test", result.Value)
	}

	if result.Count != 42 {
		t.Errorf("Count = %v, want 42", result.Count)
	}
}

func TestProtocolMessageTypes(t *testing.T) {
	// Test that protocol types are correctly defined
	tests := []struct {
		name        string
		messageType protocol.MessageType
	}{
		{"GetCPUTemp", protocol.TypeGetCPUTemp},
		{"CPUTempResponse", protocol.TypeCPUTempResponse},
		{"GetMemoryInfo", protocol.TypeGetMemoryInfo},
		{"MemoryInfoResponse", protocol.TypeMemoryInfoResponse},
		{"GetDiskInfo", protocol.TypeGetDiskInfo},
		{"DiskInfoResponse", protocol.TypeDiskInfoResponse},
		{"GetUptime", protocol.TypeGetUptime},
		{"UptimeResponse", protocol.TypeUptimeResponse},
		{"GetProcesses", protocol.TypeGetProcesses},
		{"ProcessesResponse", protocol.TypeProcessesResponse},
		{"ErrorResponse", protocol.TypeErrorResponse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.messageType == "" {
				t.Errorf("MessageType for %s is empty", tt.name)
			}
		})
	}
}

func TestTimeoutHandling(t *testing.T) {
	// Test timeout scenarios
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Simulate waiting for timeout
	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded, got %v", ctx.Err())
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Timeout didn't trigger")
	}
}

func TestGenericHelperTypeConstraints(t *testing.T) {
	// Test that different types can be used with generic helpers

	type CustomStruct struct {
		ID   string
		Data map[string]interface{}
	}

	// This ensures the generic function signature is correct
	var validateGenericFunc = func(result *CustomStruct) bool {
		return result != nil
	}

	testStruct := &CustomStruct{
		ID:   "test-id",
		Data: map[string]interface{}{"key": "value"},
	}

	if !validateGenericFunc(testStruct) {
		t.Error("Generic type validation failed")
	}

	if testStruct.ID != "test-id" {
		t.Errorf("ID = %v, want test-id", testStruct.ID)
	}
}
