package logger

import (
	"context"

	logpkg "github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

// NoOpLogger is a no-operation logger that discards all messages
// Used as default when no logger is provided
type NoOpLogger struct{}

// Debug discards debug messages
func (n *NoOpLogger) Debug(ctx context.Context, msg string, fields ...logpkg.Field) {}

// Info discards info messages
func (n *NoOpLogger) Info(ctx context.Context, msg string, fields ...logpkg.Field) {}

// Warn discards warning messages
func (n *NoOpLogger) Warn(ctx context.Context, msg string, fields ...logpkg.Field) {}

// Error discards error messages
func (n *NoOpLogger) Error(ctx context.Context, msg string, err error, fields ...logpkg.Field) {}

var (
	// defaultNoOpLogger is the singleton no-op logger instance
	defaultNoOpLogger logpkg.Logger = &NoOpLogger{}
)

// Default returns the default no-op logger (never nil)
// Use this when no logger is provided by the consumer
func Default() logpkg.Logger {
	return defaultNoOpLogger
}
