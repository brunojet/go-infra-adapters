package net_http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewNetHttpServer_PublicAPI(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewNetHttpServer(
		WithHandler(handler),
		WithAddr(":8081"),
		WithReadTimeout(20*time.Second),
		WithWriteTimeout(20*time.Second),
		WithIdleTimeout(120*time.Second),
	)

	assert.NotNil(t, server)
	assert.Implements(t, (*Server)(nil), server)
}

func TestPublicAPI_WithAllowedHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	server := NewNetHttpServer(
		WithHandler(handler),
		WithAllowedInboundHeaders("Content-Type"),
	)

	assert.NotNil(t, server)
}

func TestPublicAPI_WithMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	// Deprecated: WithMiddleware, but still works for backwards compatibility
	server := NewNetHttpServer(
		WithHandler(handler),
		WithMiddleware(middleware),
	)

	assert.NotNil(t, server)
}

func TestPublicAPI_WithObservabilityMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	observabilityMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Tracing/metrics before CORS
			next.ServeHTTP(w, r)
		})
	}

	server := NewNetHttpServer(
		WithHandler(handler),
		WithObservabilityMiddleware(observabilityMw),
	)

	assert.NotNil(t, server)
}

func TestPublicAPI_WithCustomMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	customMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// App-specific logic (auth, rate-limit, etc)
			next.ServeHTTP(w, r)
		})
	}

	server := NewNetHttpServer(
		WithHandler(handler),
		WithCustomMiddleware(customMw),
	)

	assert.NotNil(t, server)
}

func TestPublicAPI_ServerInterface(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewNetHttpServer(WithHandler(handler))

	// Verify Server interface is implemented
	var _ = server

	// Verify methods exist (interface contract)
	assert.NotNil(t, server.ListenAndServe)
	assert.NotNil(t, server.ListenAndServeTLS)
	assert.NotNil(t, server.Shutdown)
}

func TestPublicAPI_WithLogger(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mockLogger := &serverTestLogger{}

	server := NewNetHttpServer(
		WithHandler(handler),
		WithLogger(mockLogger),
	)

	assert.NotNil(t, server)
}

type serverTestLogger struct {
}

func (t *serverTestLogger) Debug(ctx context.Context, msg string, fields ...logger.Field) {
}

func (t *serverTestLogger) Info(ctx context.Context, msg string, fields ...logger.Field) {
}

func (t *serverTestLogger) Warn(ctx context.Context, msg string, fields ...logger.Field) {
}

func (t *serverTestLogger) Error(ctx context.Context, msg string, err error, fields ...logger.Field) {
}

func TestPublicAPI_WithCrossOriginProtection(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewNetHttpServer(
		WithHandler(handler),
		WithCrossOriginProtection("https://example.com", "https://trusted.com"),
	)

	assert.NotNil(t, server)
}

func TestPublicAPI_WithCrossOriginBypassPatterns(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewNetHttpServer(
		WithHandler(handler),
		WithCrossOriginProtection("https://example.com"),
		WithCrossOriginBypassPatterns("^/webhooks/.*", "^/public/.*"),
	)

	assert.NotNil(t, server)
}

func TestPublicAPI_WithCrossOriginDenyHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	customDenyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	server := NewNetHttpServer(
		WithHandler(handler),
		WithCrossOriginProtection("https://example.com"),
		WithCrossOriginDenyHandler(customDenyHandler),
	)

	assert.NotNil(t, server)
}

func TestPublicAPI_WithCrossOriginProtection_Composable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewNetHttpServer(
		WithHandler(handler),
		WithCrossOriginProtection("https://example.com"),
		WithCrossOriginProtection("https://trusted.com"),
		WithCrossOriginBypassPatterns("^/public/.*"),
	)

	assert.NotNil(t, server)
}
