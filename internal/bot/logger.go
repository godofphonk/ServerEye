package bot

import (
	"github.com/sirupsen/logrus"
)

// StructuredLogger implements the Logger interface with structured logging
type StructuredLogger struct {
	logger *logrus.Logger
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(logger *logrus.Logger) *StructuredLogger {
	return &StructuredLogger{
		logger: logger,
	}
}

// Debug logs a debug message with structured fields
func (l *StructuredLogger) Debug(msg string, fields ...Field) {
	entry := l.logger.WithFields(l.fieldsToLogrus(fields))
	entry.Debug(msg)
}

// Info logs an info message with structured fields
func (l *StructuredLogger) Info(msg string, fields ...Field) {
	entry := l.logger.WithFields(l.fieldsToLogrus(fields))
	entry.Info(msg)
}

// Warn logs a warning message with structured fields
func (l *StructuredLogger) Warn(msg string, fields ...Field) {
	entry := l.logger.WithFields(l.fieldsToLogrus(fields))
	entry.Warn(msg)
}

// Error logs an error message with structured fields
func (l *StructuredLogger) Error(msg string, err error, fields ...Field) {
	logrusFields := l.fieldsToLogrus(fields)
	if err != nil {
		logrusFields["error"] = err.Error()
	}
	entry := l.logger.WithFields(logrusFields)
	entry.Error(msg)
}

// Fatal logs a fatal message with structured fields and exits
func (l *StructuredLogger) Fatal(msg string, err error, fields ...Field) {
	logrusFields := l.fieldsToLogrus(fields)
	if err != nil {
		logrusFields["error"] = err.Error()
	}
	entry := l.logger.WithFields(logrusFields)
	entry.Fatal(msg)
}

// fieldsToLogrus converts our Field slice to logrus.Fields
func (l *StructuredLogger) fieldsToLogrus(fields []Field) logrus.Fields {
	logrusFields := make(logrus.Fields, len(fields))
	for _, field := range fields {
		logrusFields[field.Key] = field.Value
	}
	return logrusFields
}

// Helper functions to create fields
func StringField(key, value string) Field {
	return Field{Key: key, Value: value}
}

func IntField(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Int64Field(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

func FloatField(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

func BoolField(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func ErrorField(err error) Field {
	return Field{Key: "error", Value: err.Error()}
}
