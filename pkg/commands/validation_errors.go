package commands

import (
	"fmt"
	"strings"
)

// ValidationError represents a single field validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("field %q: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of ValidationError values returned when
// one or more payload fields fail validation.
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	msgs := make([]string, len(ve))
	for i, e := range ve {
		msgs[i] = e.Error()
	}
	return "validation failed: " + strings.Join(msgs, "; ")
}
