package commands

import (
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/godofphonk/ServerEye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// maxPayloadFields is the maximum number of keys allowed in a single payload
// to prevent DoS attacks that rely on submitting extremely large objects.
const maxPayloadFields = 64

// PayloadValidator validates command payloads against the schemas defined in
// pkg/protocol/schemas.go and logs every validation failure for auditing.
type PayloadValidator struct {
	logger *logrus.Logger
}

// NewPayloadValidator returns a new PayloadValidator.
func NewPayloadValidator(logger *logrus.Logger) *PayloadValidator {
	return &PayloadValidator{logger: logger}
}

// Validate checks params against the schema registered for msgType.
//
// If no schema exists for msgType the payload is accepted as-is (forward
// compatibility: unknown commands are filtered elsewhere).
//
// All detected errors are collected and returned together so the caller
// receives a complete picture of what is wrong in one shot.
func (v *PayloadValidator) Validate(msgType protocol.MessageType, params map[string]interface{}) error {
	schema, ok := protocol.Schemas[msgType]
	if !ok {
		// No schema registered – accept any payload.
		return nil
	}

	var errs ValidationErrors

	// Guard against payloads that are too large.
	if len(params) > maxPayloadFields {
		errs = append(errs, ValidationError{
			Field:   "<root>",
			Message: fmt.Sprintf("payload has %d fields, maximum allowed is %d", len(params), maxPayloadFields),
		})
		v.logFailure(msgType, errs)
		return errs
	}

	// Reject unknown fields when the schema does not allow extras.
	if !schema.AllowExtra {
		for key := range params {
			if _, defined := schema.Fields[key]; !defined {
				errs = append(errs, ValidationError{
					Field:   key,
					Message: "unknown field",
				})
			}
		}
	}

	// Validate each defined field.
	for name, fs := range schema.Fields {
		raw, present := params[name]

		if !present {
			if fs.Required {
				errs = append(errs, ValidationError{Field: name, Message: "required field is missing"})
			}
			continue
		}

		if fieldErrs := v.validateField(name, raw, fs); len(fieldErrs) > 0 {
			errs = append(errs, fieldErrs...)
		}
	}

	if len(errs) > 0 {
		v.logFailure(msgType, errs)
		return errs
	}
	return nil
}

// validateField validates a single field value against its FieldSchema.
func (v *PayloadValidator) validateField(name string, value interface{}, fs protocol.FieldSchema) ValidationErrors {
	var errs ValidationErrors

	switch fs.Kind {
	case protocol.FieldKindString:
		s, ok := value.(string)
		if !ok {
			errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("expected string, got %T", value)})
			return errs
		}
		length := utf8.RuneCountInString(s)
		if fs.MinLen > 0 && length < fs.MinLen {
			errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("string length %d is below minimum %d", length, fs.MinLen)})
		}
		if fs.MaxLen > 0 && length > fs.MaxLen {
			errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("string length %d exceeds maximum %d", length, fs.MaxLen)})
		}
		if len(fs.AllowedValues) > 0 {
			allowed := false
			for _, av := range fs.AllowedValues {
				if s == av {
					allowed = true
					break
				}
			}
			if !allowed {
				errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("value %q is not in the allowed set", s)})
			}
		}

	case protocol.FieldKindInt:
		f, ok := toFloat64(value)
		if !ok {
			errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("expected integer, got %T", value)})
			return errs
		}
		// Reject non-integers expressed as floating-point numbers.
		if math.Trunc(f) != f {
			errs = append(errs, ValidationError{Field: name, Message: "expected integer value, got floating-point"})
			return errs
		}
		if fs.HasRange {
			if f < fs.Min {
				errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("value %.0f is below minimum %.0f", f, fs.Min)})
			}
			if f > fs.Max {
				errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("value %.0f exceeds maximum %.0f", f, fs.Max)})
			}
		}

	case protocol.FieldKindFloat:
		f, ok := toFloat64(value)
		if !ok {
			errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("expected number, got %T", value)})
			return errs
		}
		if fs.HasRange {
			if f < fs.Min {
				errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("value %g is below minimum %g", f, fs.Min)})
			}
			if f > fs.Max {
				errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("value %g exceeds maximum %g", f, fs.Max)})
			}
		}

	case protocol.FieldKindBool:
		if _, ok := value.(bool); !ok {
			errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("expected bool, got %T", value)})
		}

	case protocol.FieldKindArray:
		arr, ok := value.([]interface{})
		if !ok {
			errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("expected array, got %T", value)})
			return errs
		}
		if fs.MaxSize > 0 && len(arr) > fs.MaxSize {
			errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("array length %d exceeds maximum %d", len(arr), fs.MaxSize)})
		}

	case protocol.FieldKindObject:
		if _, ok := value.(map[string]interface{}); !ok {
			errs = append(errs, ValidationError{Field: name, Message: fmt.Sprintf("expected object, got %T", value)})
		}
	}

	return errs
}

// logFailure emits a structured warning for audit purposes.
func (v *PayloadValidator) logFailure(msgType protocol.MessageType, errs ValidationErrors) {
	v.logger.WithFields(logrus.Fields{
		"command":          string(msgType),
		"validation_errors": errs.Error(),
	}).Warn("payload validation failed")
}

// toFloat64 converts a JSON-decoded numeric value to float64.
// JSON numbers are always decoded as float64 by encoding/json, but callers
// may also pass int, int64, float32, etc.
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	}
	return 0, false
}
