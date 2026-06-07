package net_http_server

import (
	"context"
	"maps"
	"net/http"
	"testing"

	logpkg "github.com/brunojet/go-infra-adapters/v4/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewHeaderControlConfig_DefaultsLoaded(t *testing.T) {
	cfg := newHeaderControlConfig()

	assert.Equal(t, len(defaultInboundDeniedHeaders), len(cfg.inboundDeniedHeaders))
	assert.Empty(t, cfg.inboundAllowedHeaders)
	assert.NotNil(t, cfg.logger) // Logger is always set to Default

	for k, v := range defaultInboundDeniedHeaders {
		assert.True(t, cfg.inboundDeniedHeaders[k], "default inbound denied header %s missing", k)
		assert.Equal(t, v, cfg.inboundDeniedHeaders[k])
	}
}

func TestWithAllowedInboundHeaders_Success(t *testing.T) {
	cfg := newHeaderControlConfig(
		WithAllowedInboundHeaders("Content-Type", "X-Request-ID"),
	)

	assert.True(t, cfg.inboundAllowedHeaders["Content-Type"])
	assert.True(t, cfg.inboundAllowedHeaders["X-Request-Id"])
	assert.Equal(t, 2, len(cfg.inboundAllowedHeaders))
}

func TestWithAllowedInboundHeaders_CaseInsensitive(t *testing.T) {
	cfg := newHeaderControlConfig(
		WithAllowedInboundHeaders("Content-Type", "content-type", "CONTENT-TYPE"),
	)

	assert.True(t, cfg.inboundAllowedHeaders["Content-Type"])
	assert.Equal(t, 1, len(cfg.inboundAllowedHeaders))
}

func TestWithAllowedInboundHeaders_EmptyPanic(t *testing.T) {
	assert.Panics(t, func() {
		newHeaderControlConfig(
			WithAllowedInboundHeaders(),
		)
	})
}

func TestWithAllowedInboundHeaders_EmptyHeaderNamePanic(t *testing.T) {
	assert.Panics(t, func() {
		newHeaderControlConfig(
			WithAllowedInboundHeaders("Content-Type", "", "X-Custom"),
		)
	})
}

func TestWithAllowedInboundHeaders_ConflictPanic(t *testing.T) {
	assert.Panics(t, func() {
		newHeaderControlConfig(
			WithAllowedInboundHeaders("Authorization"),
		)
	})

	assert.Panics(t, func() {
		newHeaderControlConfig(
			WithAllowedInboundHeaders("Cookie"),
		)
	})

	assert.Panics(t, func() {
		newHeaderControlConfig(
			WithAllowedInboundHeaders("X-Forwarded-For"),
		)
	})
}

func TestWithLogger_NilPanic(t *testing.T) {
	assert.Panics(t, func() {
		newHeaderControlConfig(
			WithLogger(nil),
		)
	})
}

func TestWithLogger_SetCorrectly(t *testing.T) {
	warnCalls := 0
	mockLogger := &testConfigLogger{
		onWarn: func(ctx context.Context, msg string, fields ...logpkg.Field) {
			warnCalls++
		},
	}

	cfg := newHeaderControlConfig(
		WithLogger(mockLogger),
	)

	assert.NotNil(t, cfg.logger)
	cfg.logger.Warn(context.Background(), "test")
	assert.Equal(t, 1, warnCalls)
}

func TestHeaderControlConfig_MultipleOptions(t *testing.T) {
	mockLogger := &testConfigLogger{}

	cfg := newHeaderControlConfig(
		WithAllowedInboundHeaders("Content-Type", "X-Custom-Header"),
		WithLogger(mockLogger),
	)

	assert.Equal(t, 2, len(cfg.inboundAllowedHeaders))
	assert.NotNil(t, cfg.logger)

	assert.True(t, cfg.inboundAllowedHeaders["Content-Type"])
	assert.True(t, cfg.inboundAllowedHeaders["X-Custom-Header"])
}

func TestMapsClone_CreatesIndependentCopy(t *testing.T) {
	// Verify that maps.Clone creates independent copies
	// This is used in newHeaderControlConfig for copying defaults
	original := map[string]bool{
		"key1": true,
		"key2": false,
	}

	cloned := maps.Clone(original)

	assert.Equal(t, original, cloned)

	cloned["key1"] = false
	cloned["key3"] = true

	assert.NotEqual(t, original, cloned)
	assert.True(t, original["key1"])
	assert.False(t, cloned["key1"])
	assert.NotContains(t, original, "key3")
	assert.Contains(t, cloned, "key3")
}

// testConfigLogger is a mock logger for config tests
type testConfigLogger struct {
	onDebug func(ctx context.Context, msg string, fields ...logpkg.Field)
	onInfo  func(ctx context.Context, msg string, fields ...logpkg.Field)
	onWarn  func(ctx context.Context, msg string, fields ...logpkg.Field)
	onError func(ctx context.Context, msg string, err error, fields ...logpkg.Field)
}

func (t *testConfigLogger) Debug(ctx context.Context, msg string, fields ...logpkg.Field) {
	if t.onDebug != nil {
		t.onDebug(ctx, msg, fields...)
	}
}

func (t *testConfigLogger) Info(ctx context.Context, msg string, fields ...logpkg.Field) {
	if t.onInfo != nil {
		t.onInfo(ctx, msg, fields...)
	}
}

func (t *testConfigLogger) Warn(ctx context.Context, msg string, fields ...logpkg.Field) {
	if t.onWarn != nil {
		t.onWarn(ctx, msg, fields...)
	}
}

func (t *testConfigLogger) Error(ctx context.Context, msg string, err error, fields ...logpkg.Field) {
	if t.onError != nil {
		t.onError(ctx, msg, err, fields...)
	}
}

func TestNewHeaderControlMiddleware_IntegrationWithConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mockLogger := &testConfigLogger{}
	middleware := NewHeaderControlMiddleware(
		handler,
		WithAllowedInboundHeaders("Content-Type"),
		WithLogger(mockLogger),
	)

	assert.NotNil(t, middleware)

	mw := middleware.(*headerControlMiddleware)
	assert.Equal(t, 1, len(mw.inboundAllowedHeaders))
	assert.NotNil(t, mw.logger)
}
