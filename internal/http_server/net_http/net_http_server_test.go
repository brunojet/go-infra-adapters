package net_http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewNetHttpServer_WithHandler_Nil(t *testing.T) {
	assert.Panics(t, func() {
		NewNetHttpServer(WithHandler(nil))
	}, "should panic on nil handler")
}

func TestNewNetHttpServer_DefaultConfig(t *testing.T) {
	// Uses DefaultServeMux when no handler configured
	server := NewNetHttpServer()

	assert.NotNil(t, server)
	httpServer := &server.(*netHttpServer).Server
	assert.Equal(t, ":8080", httpServer.Addr)
	assert.NotNil(t, httpServer.Handler)
}

func TestNewNetHttpServer_WithAddr(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewNetHttpServer(
		WithHandler(handler),
		WithAddr(":9090"),
	)

	assert.Equal(t, ":9090", server.(*netHttpServer).Addr)
}

func TestNewNetHttpServer_WithMiddleware(t *testing.T) {
	middlewareCalled := false
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewNetHttpServer(
		WithHandler(handler),
		WithMiddleware(middleware),
	)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	server.(*netHttpServer).Handler.ServeHTTP(rec, req)

	assert.True(t, middlewareCalled, "middleware should be called")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestNewNetHttpServer_MultipleMiddlewares(t *testing.T) {
	var callOrder []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "mw1")
			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "mw2")
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	server := NewNetHttpServer(
		WithHandler(handler),
		WithMiddleware(mw1),
		WithMiddleware(mw2),
	)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	server.(*netHttpServer).Handler.ServeHTTP(rec, req)

	// Middlewares are applied in order: mw1 first wraps handler, then mw2 wraps mw1
	// Execution order: mw2 → mw1 → handler
	assert.Equal(t, []string{"mw2", "mw1", "handler"}, callOrder)
}

func TestNewNetHttpServer_HeaderControlAppliedLast(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header was removed before reaching handler
		assert.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Internal-Secret", "secret")
		w.WriteHeader(http.StatusOK)
	})

	server := NewNetHttpServer(
		WithHandler(handler),
	)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	server.(*netHttpServer).Handler.ServeHTTP(rec, req)

	// Outbound headers are controlled by handler (server controls its own headers)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, "secret", rec.Header().Get("X-Internal-Secret"))
}

func TestNewNetHttpServer_WithAllowedInboundHeaders(t *testing.T) {
	var receivedHeaders http.Header

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	})

	server := NewNetHttpServer(
		WithHandler(handler),
		WithAllowedInboundHeaders("X-Custom-Header", "Content-Type"),
	)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Custom-Header", "value")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("User-Agent", "test")

	rec := httptest.NewRecorder()
	server.(*netHttpServer).Handler.ServeHTTP(rec, req)

	assert.Equal(t, "value", receivedHeaders.Get("X-Custom-Header"))
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
	assert.Empty(t, receivedHeaders.Get("Authorization"))
	assert.Empty(t, receivedHeaders.Get("User-Agent"))
}

func TestNewNetHttpServer_WithTimeouts(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewNetHttpServer(
		WithHandler(handler),
		WithReadTimeout(30*time.Second),
		WithWriteTimeout(30*time.Second),
		WithIdleTimeout(90*time.Second),
	)

	// Cast to *http.Server to verify timeout fields
	httpServer := &server.(*netHttpServer).Server
	assert.Equal(t, 30*time.Second, httpServer.ReadTimeout)
	assert.Equal(t, 30*time.Second, httpServer.WriteTimeout)
	assert.Equal(t, 90*time.Second, httpServer.IdleTimeout)
}

func TestNewNetHttpServer_WithSecurityLogger(t *testing.T) {
	var debugCount int

	mockLogger := &testLogger{
		onDebug: func(ctx context.Context, msg string, fields ...logger.Field) {
			if msg == "header removed" {
				debugCount++
			}
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewNetHttpServer(
		WithHandler(handler),
		WithLogger(mockLogger),
	)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Cookie", "session=xyz")

	rec := httptest.NewRecorder()
	server.(*netHttpServer).Handler.ServeHTTP(rec, req)

	// Should log removal of Authorization and Cookie headers
	assert.GreaterOrEqual(t, debugCount, 2)
}

func TestNewNetHttpServer_IntegrationFlow(t *testing.T) {
	authCalled := false
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCalled = true
			// Auth middleware runs BEFORE sanitization, sees complete request headers
			assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Internal", "secret")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	server := NewNetHttpServer(
		WithHandler(handler),
		WithAddr(":8080"),
		WithMiddleware(auth),
		WithAllowedInboundHeaders("Content-Type", "X-Request-ID"),
	)

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test")

	rec := httptest.NewRecorder()
	server.(*netHttpServer).Handler.ServeHTTP(rec, req)

	assert.True(t, authCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	// Server controls its own outbound headers (no outbound sanitization)
	assert.Equal(t, "secret", rec.Header().Get("X-Internal"))

	body, _ := io.ReadAll(rec.Body)
	assert.Equal(t, `{"status":"ok"}`, string(body))
}

// testLogger is a simple mock logger for testing
type testLogger struct {
	onDebug func(ctx context.Context, msg string, fields ...logger.Field)
	onWarn  func(ctx context.Context, msg string)
}

func (t *testLogger) Debug(ctx context.Context, msg string, fields ...logger.Field) {
	if t.onDebug != nil {
		t.onDebug(ctx, msg, fields...)
	}
}

func (t *testLogger) Info(ctx context.Context, msg string, fields ...logger.Field) {
}

func (t *testLogger) Warn(ctx context.Context, msg string, fields ...logger.Field) {
	if t.onWarn != nil {
		t.onWarn(ctx, msg)
	}
}

func (t *testLogger) Error(ctx context.Context, msg string, err error, fields ...logger.Field) {
}
