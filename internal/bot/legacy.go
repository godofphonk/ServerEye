package bot

import (
	"github.com/sirupsen/logrus"
)

// LegacyLoggerWrapper provides backward compatibility for old logging methods
type LegacyLoggerWrapper struct {
	logger *logrus.Logger
}

// NewLegacyLoggerWrapper creates a wrapper for legacy logging
func NewLegacyLoggerWrapper(logger *logrus.Logger) *LegacyLoggerWrapper {
	return &LegacyLoggerWrapper{logger: logger}
}

// WithField returns a logrus entry with a field
func (l *LegacyLoggerWrapper) WithField(key string, value interface{}) *logrus.Entry {
	return l.logger.WithField(key, value)
}

// WithFields returns a logrus entry with fields
func (l *LegacyLoggerWrapper) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.logger.WithFields(fields)
}

// WithError returns a logrus entry with an error
func (l *LegacyLoggerWrapper) WithError(err error) *logrus.Entry {
	return l.logger.WithError(err)
}

// Info logs an info message
func (l *LegacyLoggerWrapper) Info(msg string) {
	l.logger.Info(msg)
}

// Error logs an error message
func (l *LegacyLoggerWrapper) Error(msg string) {
	l.logger.Error(msg)
}

// Debug logs a debug message
func (l *LegacyLoggerWrapper) Debug(msg string) {
	l.logger.Debug(msg)
}

// Warn logs a warning message
func (l *LegacyLoggerWrapper) Warn(msg string) {
	l.logger.Warn(msg)
}

// Fatal logs a fatal message and exits
func (l *LegacyLoggerWrapper) Fatal(msg string) {
	l.logger.Fatal(msg)
}

// Temporary method to add logger field to Bot for backward compatibility
func (b *Bot) getLegacyLogger() *LegacyLoggerWrapper {
	// This is a hack for backward compatibility
	// In production, we should migrate all logging to the new interface
	if structuredLogger, ok := b.logger.(*StructuredLogger); ok {
		return NewLegacyLoggerWrapper(structuredLogger.logger)
	}
	
	// Fallback - create a new logger
	return NewLegacyLoggerWrapper(logrus.New())
}
