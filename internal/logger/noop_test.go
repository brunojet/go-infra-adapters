package logger

import (
	"context"
	"testing"

	logpkg "github.com/brunojet/go-infra-adapters/v4/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestNoOpLogger_Debug(t *testing.T) {
	logger := &NoOpLogger{}
	// Should not panic
	logger.Debug(context.Background(), "test message", logpkg.String("key", "value"))
}

func TestNoOpLogger_Info(t *testing.T) {
	logger := &NoOpLogger{}
	// Should not panic
	logger.Info(context.Background(), "test message", logpkg.Int("count", 42))
}

func TestNoOpLogger_Warn(t *testing.T) {
	logger := &NoOpLogger{}
	// Should not panic
	logger.Warn(context.Background(), "test message", logpkg.Bool("active", true))
}

func TestNoOpLogger_Error(t *testing.T) {
	logger := &NoOpLogger{}
	// Should not panic
	err := context.Canceled
	logger.Error(context.Background(), "test message", err, logpkg.Error("cause", err))
}

func TestDefault_ReturnsLogger(t *testing.T) {
	logger := Default()
	assert.NotNil(t, logger)
}

func TestDefault_ReturnsSameInstance(t *testing.T) {
	// Default should return the same singleton instance
	logger1 := Default()
	logger2 := Default()
	assert.Equal(t, logger1, logger2)
}

func TestDefault_Implements_Logger(t *testing.T) {
	// Verify Default returns something that implements Logger interface
	var _ = Default()
}

func TestNoOpLogger_ImplementsLogger(t *testing.T) {
	// Verify NoOpLogger implements Logger interface
	var noop *NoOpLogger
	var _ logpkg.Logger = noop
}

func TestNoOpLogger_AllMethodsNoop(t *testing.T) {
	logger := &NoOpLogger{}
	ctx := context.Background()

	// All methods should complete without error
	logger.Debug(ctx, "debug")
	logger.Info(ctx, "info")
	logger.Warn(ctx, "warn")
	logger.Error(ctx, "error", context.DeadlineExceeded)

	// With fields
	fields := []logpkg.Field{
		logpkg.String("key", "value"),
		logpkg.Int("num", 42),
	}
	logger.Debug(ctx, "debug", fields...)
	logger.Info(ctx, "info", fields...)
	logger.Warn(ctx, "warn", fields...)
	logger.Error(ctx, "error", context.Canceled, fields...)
}
