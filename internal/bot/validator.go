package bot

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// ErrInvalidServerKey indicates an invalid server key format
	ErrInvalidServerKey = errors.New("invalid server key format")
	// ErrInvalidServerName indicates an invalid server name
	ErrInvalidServerName = errors.New("invalid server name")
	// ErrInvalidContainerID indicates an invalid container ID
	ErrInvalidContainerID = errors.New("invalid container ID")
	// ErrServerNameTooLong indicates server name exceeds maximum length
	ErrServerNameTooLong = errors.New("server name too long")
	// ErrContainerIDTooShort indicates container ID is too short
	ErrContainerIDTooShort = errors.New("container ID too short")
	// ErrContainerIDTooLong indicates container ID is too long
	ErrContainerIDTooLong = errors.New("container ID too long")
	// ErrProtectedContainer indicates attempt to manage a protected container
	ErrProtectedContainer = errors.New("container is protected and cannot be managed")
)

// InputValidator implements the Validator interface
type InputValidator struct {
	serverKeyRegex    *regexp.Regexp
	containerIDRegex  *regexp.Regexp
	protectedContainers []string
}

// NewInputValidator creates a new input validator
func NewInputValidator() *InputValidator {
	return &InputValidator{
		serverKeyRegex:   regexp.MustCompile(`^srv_[a-f0-9]{32}$`),
		containerIDRegex: regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`),
		protectedContainers: []string{
			"servereye-bot",
			"redis",
			"postgres",
			"postgresql",
			"database",
			"db",
		},
	}
}

// ValidateServerKey validates server key format
func (v *InputValidator) ValidateServerKey(key string) error {
	if key == "" {
		return ErrInvalidServerKey
	}
	
	if !strings.HasPrefix(key, "srv_") {
		return ErrInvalidServerKey
	}
	
	if !v.serverKeyRegex.MatchString(key) {
		return ErrInvalidServerKey
	}
	
	return nil
}

// ValidateServerName validates server name
func (v *InputValidator) ValidateServerName(name string) error {
	if name == "" {
		return ErrInvalidServerName
	}
	
	if len(name) > 50 {
		return ErrServerNameTooLong
	}
	
	// Check for potentially dangerous characters
	if strings.ContainsAny(name, "<>\"'&;") {
		return ErrInvalidServerName
	}
	
	return nil
}

// ValidateContainerID validates container ID format
func (v *InputValidator) ValidateContainerID(id string) error {
	if len(id) < 3 {
		return ErrContainerIDTooShort
	}
	
	if len(id) > 64 {
		return ErrContainerIDTooLong
	}
	
	if !v.containerIDRegex.MatchString(id) {
		return ErrInvalidContainerID
	}
	
	// Check if container is protected
	idLower := strings.ToLower(id)
	for _, protected := range v.protectedContainers {
		if strings.Contains(idLower, protected) {
			return ErrProtectedContainer
		}
	}
	
	return nil
}

// SanitizeInput removes potentially dangerous characters from user input
func (v *InputValidator) SanitizeInput(input string) string {
	// Remove null bytes and control characters
	input = strings.ReplaceAll(input, "\x00", "")
	input = regexp.MustCompile(`[\x00-\x1f\x7f]`).ReplaceAllString(input, "")
	
	// Trim whitespace
	input = strings.TrimSpace(input)
	
	return input
}
