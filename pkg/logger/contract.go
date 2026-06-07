// Package logger provides structured logging abstractions for use across the library.
package logger

import (
	"context"
)

// Logger defines the interface for structured logging across the library
type Logger interface {
	// Debug logs a debug message with optional fields
	Debug(ctx context.Context, msg string, fields ...Field)

	// Info logs an informational message with optional fields
	Info(ctx context.Context, msg string, fields ...Field)

	// Warn logs a warning message with optional fields
	Warn(ctx context.Context, msg string, fields ...Field)

	// Error logs an error message with the error and optional fields
	Error(ctx context.Context, msg string, err error, fields ...Field)
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// Helper functions for common field types

// String creates a field with a string value
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates a field with an integer value
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a field with a boolean value
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Error creates a field with an error value
func Error(key string, err error) Field {
	return Field{Key: key, Value: err}
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}
