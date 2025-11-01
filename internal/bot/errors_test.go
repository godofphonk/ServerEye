package bot

import (
	"errors"
	"testing"
)

func TestBotError_Error(t *testing.T) {
	tests := []struct {
		name    string
		botErr  *BotError
		wantMsg string
	}{
		{
			name: "error with cause",
			botErr: &BotError{
				Code:    "TEST_ERROR",
				Message: "test message",
				Cause:   errors.New("underlying error"),
			},
			wantMsg: "TEST_ERROR: test message (caused by: underlying error)",
		},
		{
			name: "error without cause",
			botErr: &BotError{
				Code:    "TEST_ERROR",
				Message: "test message",
				Cause:   nil,
			},
			wantMsg: "TEST_ERROR: test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.botErr.Error(); got != tt.wantMsg {
				t.Errorf("BotError.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

func TestBotError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	botErr := &BotError{
		Code:    "TEST_ERROR",
		Message: "test",
		Cause:   cause,
	}

	if unwrapped := botErr.Unwrap(); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestBotError_Is(t *testing.T) {
	baseErr := errors.New("base error")
	botErr := &BotError{
		Code:    "TEST_ERROR",
		Message: "test",
		Cause:   baseErr,
	}

	tests := []struct {
		name   string
		target error
		want   bool
	}{
		{
			name:   "same code",
			target: &BotError{Code: "TEST_ERROR"},
			want:   true,
		},
		{
			name:   "different code",
			target: &BotError{Code: "OTHER_ERROR"},
			want:   false,
		},
		{
			name:   "unwrapped error",
			target: baseErr,
			want:   true,
		},
		{
			name:   "nil error",
			target: nil,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := botErr.Is(tt.target); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewBotError(t *testing.T) {
	cause := errors.New("cause")
	err := NewBotError("TEST_CODE", "test message", cause)

	if err.Code != "TEST_CODE" {
		t.Errorf("Code = %v, want TEST_CODE", err.Code)
	}
	if err.Message != "test message" {
		t.Errorf("Message = %v, want test message", err.Message)
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
	if err.Context == nil {
		t.Error("Context should be initialized")
	}
}

func TestBotError_WithContext(t *testing.T) {
	err := NewBotError("TEST", "test", nil)
	err.WithContext("user_id", 123)
	err.WithContext("server", "test-server")

	if val, ok := err.Context["user_id"]; !ok || val != 123 {
		t.Errorf("Context[user_id] = %v, want 123", val)
	}
	if val, ok := err.Context["server"]; !ok || val != "test-server" {
		t.Errorf("Context[server] = %v, want test-server", val)
	}
}

func TestErrorConstructors(t *testing.T) {
	cause := errors.New("test cause")

	tests := []struct {
		name     string
		errFunc  func(string, error) *BotError
		wantCode string
	}{
		{"ValidationError", NewValidationError, ErrCodeValidation},
		{"DatabaseError", NewDatabaseError, ErrCodeDatabase},
		{"RedisError", NewRedisError, ErrCodeRedis},
		{"AgentError", NewAgentError, ErrCodeAgent},
		{"TelegramError", NewTelegramError, ErrCodeTelegram},
		{"TimeoutError", NewTimeoutError, ErrCodeTimeout},
		{"InternalError", NewInternalError, ErrCodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errFunc("test message", cause)
			if err.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", err.Code, tt.wantCode)
			}
			if err.Message != "test message" {
				t.Errorf("Message = %v, want test message", err.Message)
			}
			if err.Cause != cause {
				t.Errorf("Cause = %v, want %v", err.Cause, cause)
			}
		})
	}
}

func TestErrorToUserMessage(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{
			name:    "validation error",
			err:     NewValidationError("invalid input", nil),
			wantMsg: "Invalid input: invalid input",
		},
		{
			name:    "database error",
			err:     NewDatabaseError("connection failed", nil),
			wantMsg: "Database error occurred. Please try again later.",
		},
		{
			name:    "redis error",
			err:     NewRedisError("connection failed", nil),
			wantMsg: "Connection error occurred. Please try again later.",
		},
		{
			name:    "agent error",
			err:     NewAgentError("agent offline", nil),
			wantMsg: "Server communication error. Please check if your server is online.",
		},
		{
			name:    "telegram error",
			err:     NewTelegramError("api error", nil),
			wantMsg: "Telegram API error occurred. Please try again.",
		},
		{
			name:    "timeout error",
			err:     NewTimeoutError("operation timeout", nil),
			wantMsg: "Operation timed out. Please try again.",
		},
		{
			name:    "not found error",
			err:     NewBotError(ErrCodeNotFound, "resource not found", nil),
			wantMsg: "Resource not found.",
		},
		{
			name:    "unauthorized error",
			err:     NewBotError(ErrCodeUnauthorized, "not authorized", nil),
			wantMsg: "You are not authorized to perform this operation.",
		},
		{
			name:    "invalid server key",
			err:     ErrInvalidServerKey,
			wantMsg: "Invalid server key format. Server key must start with 'srv_'",
		},
		{
			name:    "server name too long",
			err:     ErrServerNameTooLong,
			wantMsg: "Server name too long (max 50 characters)",
		},
		{
			name:    "protected container",
			err:     ErrProtectedContainer,
			wantMsg: "This container is protected and cannot be managed",
		},
		{
			name:    "agent timeout",
			err:     ErrAgentTimeout,
			wantMsg: "Server response timeout. Please check if your server is online.",
		},
		{
			name:    "unknown error",
			err:     errors.New("some random error"),
			wantMsg: "An unexpected error occurred. Please try again later.",
		},
		{
			name:    "unknown bot error code",
			err:     NewBotError("UNKNOWN_CODE", "unknown", nil),
			wantMsg: "An unexpected error occurred. Please try again later.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ErrorToUserMessage(tt.err)
			if got != tt.wantMsg {
				t.Errorf("ErrorToUserMessage() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

func TestCommonErrors(t *testing.T) {
	// Test that common errors are defined
	commonErrors := []error{
		ErrUserNotFound,
		ErrServerNotFound,
		ErrServerAlreadyExists,
		ErrUnauthorized,
		ErrInvalidCommand,
		ErrAgentTimeout,
		ErrAgentUnavailable,
		ErrDatabaseConnection,
		ErrRedisConnection,
	}

	for _, err := range commonErrors {
		if err == nil {
			t.Error("Common error should not be nil")
		}
		if err.Error() == "" {
			t.Error("Common error should have error message")
		}
	}
}

func TestErrorsWithErrors_Is(t *testing.T) {
	// Test that errors.Is works with our errors
	botErr := NewDatabaseError("db error", ErrDatabaseConnection)

	if !errors.Is(botErr, ErrDatabaseConnection) {
		t.Error("errors.Is should work with wrapped BotError")
	}
}

func TestErrorsWithErrors_As(t *testing.T) {
	// Test that errors.As works with our errors
	originalErr := NewValidationError("validation failed", nil)
	var botErr *BotError

	if !errors.As(originalErr, &botErr) {
		t.Error("errors.As should work with BotError")
	}

	if botErr.Code != ErrCodeValidation {
		t.Errorf("Code = %v, want %v", botErr.Code, ErrCodeValidation)
	}
}
