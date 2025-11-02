package bot

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewStructuredLogger(t *testing.T) {
	logger := logrus.New()
	structLogger := NewStructuredLogger(logger)

	if structLogger == nil {
		t.Fatal("NewStructuredLogger() returned nil")
	}

	if structLogger.logger != logger {
		t.Error("Logger not set correctly")
	}
}

func TestStructuredLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	structLogger := NewStructuredLogger(logger)
	structLogger.Debug("test debug message", StringField("key", "value"))

	output := buf.String()
	if !strings.Contains(output, "test debug message") {
		t.Errorf("Debug output missing message: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("Debug output missing field: %s", output)
	}
}

func TestStructuredLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	structLogger := NewStructuredLogger(logger)
	structLogger.Info("test info message", IntField("count", 42))

	output := buf.String()
	if !strings.Contains(output, "test info message") {
		t.Errorf("Info output missing message: %s", output)
	}
	if !strings.Contains(output, "count=42") {
		t.Errorf("Info output missing field: %s", output)
	}
}

func TestStructuredLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.WarnLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	structLogger := NewStructuredLogger(logger)
	structLogger.Warn("test warning", BoolField("critical", true))

	output := buf.String()
	if !strings.Contains(output, "test warning") {
		t.Errorf("Warn output missing message: %s", output)
	}
	if !strings.Contains(output, "critical=true") {
		t.Errorf("Warn output missing field: %s", output)
	}
}

func TestStructuredLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.ErrorLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	structLogger := NewStructuredLogger(logger)
	testErr := errors.New("test error")
	structLogger.Error("error occurred", testErr, StringField("context", "test"))

	output := buf.String()
	if !strings.Contains(output, "error occurred") {
		t.Errorf("Error output missing message: %s", output)
	}
	if !strings.Contains(output, "test error") {
		t.Errorf("Error output missing error: %s", output)
	}
	if !strings.Contains(output, "context=test") {
		t.Errorf("Error output missing field: %s", output)
	}
}

func TestStructuredLogger_Error_NilError(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.ErrorLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	structLogger := NewStructuredLogger(logger)
	structLogger.Error("error message", nil)

	output := buf.String()
	if !strings.Contains(output, "error message") {
		t.Errorf("Error output missing message: %s", output)
	}
}

func TestFieldsToLogrus(t *testing.T) {
	logger := logrus.New()
	structLogger := NewStructuredLogger(logger)

	fields := []Field{
		StringField("string", "value"),
		IntField("int", 123),
		Int64Field("int64", 456),
		FloatField("float", 3.14),
		BoolField("bool", true),
	}

	logrusFields := structLogger.fieldsToLogrus(fields)

	if logrusFields["string"] != "value" {
		t.Errorf("string field = %v, want 'value'", logrusFields["string"])
	}
	if logrusFields["int"] != 123 {
		t.Errorf("int field = %v, want 123", logrusFields["int"])
	}
	if logrusFields["int64"] != int64(456) {
		t.Errorf("int64 field = %v, want 456", logrusFields["int64"])
	}
	if logrusFields["float"] != 3.14 {
		t.Errorf("float field = %v, want 3.14", logrusFields["float"])
	}
	if logrusFields["bool"] != true {
		t.Errorf("bool field = %v, want true", logrusFields["bool"])
	}
}

func TestFieldsToLogrus_Empty(t *testing.T) {
	logger := logrus.New()
	structLogger := NewStructuredLogger(logger)

	logrusFields := structLogger.fieldsToLogrus([]Field{})

	if len(logrusFields) != 0 {
		t.Errorf("Empty fields should produce empty logrus.Fields, got %d items", len(logrusFields))
	}
}

func TestStringField(t *testing.T) {
	field := StringField("test", "value")
	if field.Key != "test" {
		t.Errorf("Key = %v, want 'test'", field.Key)
	}
	if field.Value != "value" {
		t.Errorf("Value = %v, want 'value'", field.Value)
	}
}

func TestIntField(t *testing.T) {
	field := IntField("count", 42)
	if field.Key != "count" {
		t.Errorf("Key = %v, want 'count'", field.Key)
	}
	if field.Value != 42 {
		t.Errorf("Value = %v, want 42", field.Value)
	}
}

func TestInt64Field(t *testing.T) {
	field := Int64Field("bignum", 9999999999)
	if field.Key != "bignum" {
		t.Errorf("Key = %v, want 'bignum'", field.Key)
	}
	if field.Value != int64(9999999999) {
		t.Errorf("Value = %v, want 9999999999", field.Value)
	}
}

func TestFloatField(t *testing.T) {
	field := FloatField("pi", 3.14159)
	if field.Key != "pi" {
		t.Errorf("Key = %v, want 'pi'", field.Key)
	}
	if field.Value != 3.14159 {
		t.Errorf("Value = %v, want 3.14159", field.Value)
	}
}

func TestBoolField(t *testing.T) {
	field := BoolField("enabled", true)
	if field.Key != "enabled" {
		t.Errorf("Key = %v, want 'enabled'", field.Key)
	}
	if field.Value != true {
		t.Errorf("Value = %v, want true", field.Value)
	}
}

func TestErrorField(t *testing.T) {
	testErr := errors.New("something went wrong")
	field := ErrorField(testErr)

	if field.Key != "error" {
		t.Errorf("Key = %v, want 'error'", field.Key)
	}
	if field.Value != "something went wrong" {
		t.Errorf("Value = %v, want 'something went wrong'", field.Value)
	}
}

func TestStructuredLogger_MultipleFields(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	structLogger := NewStructuredLogger(logger)
	structLogger.Info("complex log",
		StringField("user", "john"),
		IntField("age", 30),
		BoolField("active", true),
		FloatField("score", 95.5),
	)

	output := buf.String()
	if !strings.Contains(output, "user=john") {
		t.Error("Missing user field")
	}
	if !strings.Contains(output, "age=30") {
		t.Error("Missing age field")
	}
	if !strings.Contains(output, "active=true") {
		t.Error("Missing active field")
	}
	if !strings.Contains(output, "score=95.5") {
		t.Error("Missing score field")
	}
}

func TestStructuredLogger_NoFields(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	structLogger := NewStructuredLogger(logger)
	structLogger.Info("simple message")

	output := buf.String()
	if !strings.Contains(output, "simple message") {
		t.Errorf("Output missing message: %s", output)
	}
}
