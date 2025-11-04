package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	logger := NewStructuredLogger(nil)
	bot := &Bot{
		logger: logger,
	}

	tests := []struct {
		name string
		data interface{}
		want map[string]interface{}
	}{
		{
			name: "simple string map",
			data: map[string]string{
				"status":  "ok",
				"message": "success",
			},
			want: map[string]interface{}{
				"status":  "ok",
				"message": "success",
			},
		},
		{
			name: "nested structure",
			data: map[string]interface{}{
				"success": true,
				"data": map[string]string{
					"key": "value",
				},
			},
			want: map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"key": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			bot.writeJSON(w, tt.data)

			// Check status code
			if w.Code != http.StatusOK {
				t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
			}

			// Check Content-Type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %v, want application/json", contentType)
			}

			// Check JSON response
			var got map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			// Compare basic fields
			if tt.name == "simple string map" {
				if got["status"] != "ok" {
					t.Errorf("status = %v, want ok", got["status"])
				}
				if got["message"] != "success" {
					t.Errorf("message = %v, want success", got["message"])
				}
			}
		})
	}
}

func TestWriteJSONSuccess(t *testing.T) {
	logger := NewStructuredLogger(nil)
	bot := &Bot{
		logger: logger,
	}

	w := httptest.NewRecorder()
	testData := map[string]string{
		"message": "operation successful",
	}

	bot.writeJSONSuccess(w, testData)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
	}

	// Check Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %v, want application/json", contentType)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Check success field
	success, ok := response["success"].(bool)
	if !ok || !success {
		t.Errorf("success = %v, want true", response["success"])
	}

	// Check data field exists
	if _, ok := response["data"]; !ok {
		t.Error("data field not found in response")
	}
}
