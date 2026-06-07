package net_http_client

import (
	"context"
	"testing"

	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewHeaderControlConfig_DefaultDeniedHeaders(t *testing.T) {
	cfg := newHeaderControlConfig()

	// Verify core denied headers are present
	expectedDenied := []string{
		"Authorization",
		"Cookie",
		"Proxy-Authorization",
		"Host",
		"User-Agent",
		"Connection",
		"Keep-Alive",
		"Transfer-Encoding",
		"X-Forwarded-For",
	}

	for _, header := range expectedDenied {
		assert.True(t, cfg.requestDeniedHeaders[header], "expected %s to be in request denied headers", header)
	}
}

func TestWithLogger_Valid(t *testing.T) {
	mockLogger := &mockClientLogger{}

	cfg := newHeaderControlConfig(
		WithLogger(mockLogger),
	)

	assert.Equal(t, mockLogger, cfg.logger)
}

func TestWithLogger_NilPanic(t *testing.T) {
	assert.Panics(t, func() {
		newHeaderControlConfig(WithLogger(nil))
	})
}

type mockClientLogger struct {
}

func (m *mockClientLogger) Debug(ctx context.Context, msg string, fields ...logger.Field) {
}

func (m *mockClientLogger) Info(ctx context.Context, msg string, fields ...logger.Field) {
}

func (m *mockClientLogger) Warn(ctx context.Context, msg string, fields ...logger.Field) {
}

func (m *mockClientLogger) Error(ctx context.Context, msg string, err error, fields ...logger.Field) {
}
