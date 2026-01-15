package interfaces

import (
	"github.com/sirupsen/logrus"
)

// LogrusAdapter adapts logrus.Logger to interfaces.Logger
type LogrusAdapter struct {
	Entry *logrus.Entry
}

// NewLogrusAdapter creates a new logrus adapter
func NewLogrusAdapter(logger *logrus.Logger) Logger {
	return &LogrusAdapter{Entry: logrus.NewEntry(logger)}
}

// Debug logs a debug message
func (l *LogrusAdapter) Debug(args ...interface{}) {
	l.Entry.Debug(args...)
}

// Debugf logs a debug message with formatting
func (l *LogrusAdapter) Debugf(format string, args ...interface{}) {
	l.Entry.Debugf(format, args...)
}

// Info logs an info message
func (l *LogrusAdapter) Info(args ...interface{}) {
	l.Entry.Info(args...)
}

// Infof logs an info message with formatting
func (l *LogrusAdapter) Infof(format string, args ...interface{}) {
	l.Entry.Infof(format, args...)
}

// Warn logs a warning message
func (l *LogrusAdapter) Warn(args ...interface{}) {
	l.Entry.Warn(args...)
}

// Warnf logs a warning message with formatting
func (l *LogrusAdapter) Warnf(format string, args ...interface{}) {
	l.Entry.Warnf(format, args...)
}

// Error logs an error message
func (l *LogrusAdapter) Error(args ...interface{}) {
	l.Entry.Error(args...)
}

// Errorf logs an error message with formatting
func (l *LogrusAdapter) Errorf(format string, args ...interface{}) {
	l.Entry.Errorf(format, args...)
}

// Fatal logs a fatal message
func (l *LogrusAdapter) Fatal(args ...interface{}) {
	l.Entry.Fatal(args...)
}

// Fatalf logs a fatal message with formatting
func (l *LogrusAdapter) Fatalf(format string, args ...interface{}) {
	l.Entry.Fatalf(format, args...)
}

// WithError returns a new Logger with the error field
func (l *LogrusAdapter) WithError(err error) Logger {
	return &LogrusAdapter{Entry: l.Entry.WithError(err)}
}

// WithField returns a new Logger with the field
func (l *LogrusAdapter) WithField(key string, value interface{}) Logger {
	return &LogrusAdapter{Entry: l.Entry.WithField(key, value)}
}

// WithFields returns a new Logger with the fields
func (l *LogrusAdapter) WithFields(fields map[string]interface{}) Logger {
	return &LogrusAdapter{Entry: l.Entry.WithFields(fields)}
}
