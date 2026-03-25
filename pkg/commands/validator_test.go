package commands

import (
	"fmt"
	"strings"
	"testing"

	"github.com/godofphonk/ServerEye/pkg/protocol"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestValidator() *PayloadValidator {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	return NewPayloadValidator(logger)
}

// ---------------------------------------------------------------------------
// ValidationErrors
// ---------------------------------------------------------------------------

func TestValidationErrors_Error(t *testing.T) {
	errs := ValidationErrors{
		{Field: "name", Message: "too long"},
		{Field: "limit", Message: "below minimum"},
	}
	msg := errs.Error()
	assert.Contains(t, msg, "name")
	assert.Contains(t, msg, "limit")
	assert.True(t, strings.HasPrefix(msg, "validation failed:"))
}

func TestValidationError_Error(t *testing.T) {
	e := ValidationError{Field: "foo", Message: "bar"}
	assert.Equal(t, `field "foo": bar`, e.Error())
}

// ---------------------------------------------------------------------------
// PayloadValidator – commands without parameters
// ---------------------------------------------------------------------------

func TestValidate_NoParamsCommands_AcceptEmptyPayload(t *testing.T) {
	v := newTestValidator()
	noParamCommands := []protocol.MessageType{
		protocol.TypeGetCPUTemp,
		protocol.TypeGetSystemInfo,
		protocol.TypeGetMemoryInfo,
		protocol.TypeGetUptime,
		protocol.TypePing,
	}
	for _, cmd := range noParamCommands {
		t.Run(string(cmd), func(t *testing.T) {
			err := v.Validate(cmd, map[string]interface{}{})
			assert.NoError(t, err)
		})
	}
}

func TestValidate_NoParamsCommands_RejectUnknownField(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.TypeGetCPUTemp, map[string]interface{}{
		"unexpected": "value",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}

// ---------------------------------------------------------------------------
// String field validation
// ---------------------------------------------------------------------------

func TestValidate_StringField_TooLong(t *testing.T) {
	v := newTestValidator()
	longFilter := strings.Repeat("x", 257) // max is 256 for TypeGetContainers.filter
	err := v.Validate(protocol.TypeGetContainers, map[string]interface{}{
		"filter": longFilter,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestValidate_StringField_AcceptedWithinLimit(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.TypeGetContainers, map[string]interface{}{
		"filter": "running",
	})
	assert.NoError(t, err)
}

func TestValidate_StringField_WrongType(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.TypeGetContainers, map[string]interface{}{
		"filter": 12345,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected string")
}

// ---------------------------------------------------------------------------
// Integer field validation
// ---------------------------------------------------------------------------

func TestValidate_IntField_BelowMin(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.TypeGetProcesses, map[string]interface{}{
		"limit": float64(0), // min is 1
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "below minimum")
}

func TestValidate_IntField_AboveMax(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.TypeGetProcesses, map[string]interface{}{
		"limit": float64(9999), // max is 1000
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestValidate_IntField_Valid(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.TypeGetProcesses, map[string]interface{}{
		"limit": float64(50),
	})
	assert.NoError(t, err)
}

func TestValidate_IntField_FloatRejected(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.TypeGetProcesses, map[string]interface{}{
		"limit": float64(10.5),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "floating-point")
}

func TestValidate_IntField_WrongType(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.TypeGetProcesses, map[string]interface{}{
		"limit": "ten",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected integer")
}

func TestValidate_IntField_NegativeOverflow(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.TypeGetProcesses, map[string]interface{}{
		"limit": float64(-2147483648),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "below minimum")
}

// ---------------------------------------------------------------------------
// Required field validation
// ---------------------------------------------------------------------------

func TestValidate_RequiredField_Missing(t *testing.T) {
	v := newTestValidator()
	// TypeCreateContainer requires "image"
	err := v.Validate(protocol.TypeCreateContainer, map[string]interface{}{
		"name": "mycontainer",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required field is missing")
}

func TestValidate_RequiredField_Present(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.TypeCreateContainer, map[string]interface{}{
		"image": "nginx:latest",
	})
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Unknown command type – should be accepted
// ---------------------------------------------------------------------------

func TestValidate_UnknownCommandType_Accepted(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.MessageType("unknown_custom_command"), map[string]interface{}{
		"anything": "goes",
	})
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Large payload (DoS protection)
// ---------------------------------------------------------------------------

func TestValidate_TooManyFields_Rejected(t *testing.T) {
	v := newTestValidator()
	params := make(map[string]interface{}, maxPayloadFields+1)
	for i := 1; i <= maxPayloadFields+1; i++ {
		params[fmt.Sprintf("field%d", i)] = "v"
	}
	err := v.Validate(protocol.TypeGetCPUTemp, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "payload has")
}

// ---------------------------------------------------------------------------
// Container action commands
// ---------------------------------------------------------------------------

func TestValidate_ContainerAction_Valid(t *testing.T) {
	v := newTestValidator()
	for _, cmd := range []protocol.MessageType{
		protocol.TypeStartContainer,
		protocol.TypeStopContainer,
		protocol.TypeRestartContainer,
		protocol.TypeRemoveContainer,
	} {
		t.Run(string(cmd), func(t *testing.T) {
			err := v.Validate(cmd, map[string]interface{}{
				"container_id": "abc123",
			})
			assert.NoError(t, err)
		})
	}
}

func TestValidate_ContainerAction_NameTooLong(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.TypeStartContainer, map[string]interface{}{
		"container_name": strings.Repeat("x", 300), // max 256
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

// ---------------------------------------------------------------------------
// UpdateAgent
// ---------------------------------------------------------------------------

func TestValidate_UpdateAgent_Valid(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.TypeUpdateAgent, map[string]interface{}{
		"version": "1.2.3",
	})
	assert.NoError(t, err)
}

func TestValidate_UpdateAgent_EmptyVersion(t *testing.T) {
	v := newTestValidator()
	err := v.Validate(protocol.TypeUpdateAgent, map[string]interface{}{
		"version": "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "below minimum")
}

// ---------------------------------------------------------------------------
// toFloat64 helper
// ---------------------------------------------------------------------------

func TestToFloat64(t *testing.T) {
	cases := []struct {
		in  interface{}
		out float64
		ok  bool
	}{
		{float64(3.14), 3.14, true},
		{float32(1.5), float64(float32(1.5)), true},
		{int(7), 7.0, true},
		{int32(8), 8.0, true},
		{int64(9), 9.0, true},
		{uint(10), 10.0, true},
		{uint32(11), 11.0, true},
		{uint64(12), 12.0, true},
		{"not a number", 0, false},
		{nil, 0, false},
	}
	for _, c := range cases {
		got, ok := toFloat64(c.in)
		assert.Equal(t, c.ok, ok)
		if c.ok {
			assert.InDelta(t, c.out, got, 1e-6)
		}
	}
}
