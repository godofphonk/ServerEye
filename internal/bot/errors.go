package bot

import (
	"errors"
	"fmt"
)

// Common error types
var (
	// ErrUserNotFound indicates user was not found in database
	ErrUserNotFound = errors.New("user not found")
	// ErrServerNotFound indicates server was not found
	ErrServerNotFound = errors.New("server not found")
	// ErrServerAlreadyExists indicates server already exists
	ErrServerAlreadyExists = errors.New("server already exists")
	// ErrUnauthorized indicates user is not authorized for operation
	ErrUnauthorized = errors.New("unauthorized")
	// ErrInvalidCommand indicates invalid command format
	ErrInvalidCommand = errors.New("invalid command")
	// ErrAgentTimeout indicates agent response timeout
	ErrAgentTimeout = errors.New("agent response timeout")
	// ErrAgentUnavailable indicates agent is unavailable
	ErrAgentUnavailable = errors.New("agent unavailable")
	// ErrDatabaseConnection indicates database connection error
	ErrDatabaseConnection = errors.New("database connection error")
	// ErrRedisConnection indicates Redis connection error
	ErrRedisConnection = errors.New("redis connection error")
)

// BotError represents a structured error with context
type BotError struct {
	Code    string
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error implements the error interface
func (e *BotError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *BotError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target
func (e *BotError) Is(target error) bool {
	if target == nil {
		return false
	}
	
	if botErr, ok := target.(*BotError); ok {
		return e.Code == botErr.Code
	}
	
	return errors.Is(e.Cause, target)
}

// NewBotError creates a new BotError
func NewBotError(code, message string, cause error) *BotError {
	return &BotError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context to the error
func (e *BotError) WithContext(key string, value interface{}) *BotError {
	e.Context[key] = value
	return e
}

// Error codes
const (
	ErrCodeValidation     = "VALIDATION_ERROR"
	ErrCodeDatabase       = "DATABASE_ERROR"
	ErrCodeRedis          = "REDIS_ERROR"
	ErrCodeAgent          = "AGENT_ERROR"
	ErrCodeTelegram       = "TELEGRAM_ERROR"
	ErrCodeUnauthorized   = "UNAUTHORIZED_ERROR"
	ErrCodeNotFound       = "NOT_FOUND_ERROR"
	ErrCodeTimeout        = "TIMEOUT_ERROR"
	ErrCodeInternal       = "INTERNAL_ERROR"
)

// Helper functions to create common errors
func NewValidationError(message string, cause error) *BotError {
	return NewBotError(ErrCodeValidation, message, cause)
}

func NewDatabaseError(message string, cause error) *BotError {
	return NewBotError(ErrCodeDatabase, message, cause)
}

func NewRedisError(message string, cause error) *BotError {
	return NewBotError(ErrCodeRedis, message, cause)
}

func NewAgentError(message string, cause error) *BotError {
	return NewBotError(ErrCodeAgent, message, cause)
}

func NewTelegramError(message string, cause error) *BotError {
	return NewBotError(ErrCodeTelegram, message, cause)
}

func NewTimeoutError(message string, cause error) *BotError {
	return NewBotError(ErrCodeTimeout, message, cause)
}

func NewInternalError(message string, cause error) *BotError {
	return NewBotError(ErrCodeInternal, message, cause)
}

// ErrorToUserMessage converts technical errors to user-friendly messages
func ErrorToUserMessage(err error) string {
	var botErr *BotError
	if errors.As(err, &botErr) {
		switch botErr.Code {
		case ErrCodeValidation:
			return fmt.Sprintf("Invalid input: %s", botErr.Message)
		case ErrCodeDatabase:
			return "Database error occurred. Please try again later."
		case ErrCodeRedis:
			return "Connection error occurred. Please try again later."
		case ErrCodeAgent:
			return "Server communication error. Please check if your server is online."
		case ErrCodeTelegram:
			return "Telegram API error occurred. Please try again."
		case ErrCodeTimeout:
			return "Operation timed out. Please try again."
		case ErrCodeNotFound:
			return "Resource not found."
		case ErrCodeUnauthorized:
			return "You are not authorized to perform this operation."
		default:
			return "An unexpected error occurred. Please try again later."
		}
	}
	
	// Handle standard errors
	switch {
	case errors.Is(err, ErrInvalidServerKey):
		return "Invalid server key format. Server key must start with 'srv_'"
	case errors.Is(err, ErrServerNameTooLong):
		return "Server name too long (max 50 characters)"
	case errors.Is(err, ErrProtectedContainer):
		return "This container is protected and cannot be managed"
	case errors.Is(err, ErrAgentTimeout):
		return "Server response timeout. Please check if your server is online."
	default:
		return "An unexpected error occurred. Please try again later."
	}
}
