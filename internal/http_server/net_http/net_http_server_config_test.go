package net_http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internallgr "github.com/brunojet/go-infra-adapters/v4/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestWithAddr_EmptyAddressPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	assert.Panics(t, func() {
		NewNetHttpServer(WithHandler(handler), WithAddr(""))
	})
}

func TestWithReadTimeout_ZeroPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	assert.Panics(t, func() {
		NewNetHttpServer(WithHandler(handler), WithReadTimeout(0))
	})
}

func TestWithMiddleware_NilPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	assert.Panics(t, func() {
		NewNetHttpServer(WithHandler(handler), WithMiddleware(nil))
	})
}

func TestWithLogger_NilPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	assert.Panics(t, func() {
		NewNetHttpServer(WithHandler(handler), WithLogger(nil))
	})
}

func TestServerConfig_BuildFinalHandler_WithoutHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := &serverConfig{
		handler:               handler,
		customMiddlewares:     make([]MiddlewareFunc, 0),
		inboundAllowedHeaders: make(map[string]bool),
		logger:                internallgr.Default(),
	}

	finalHandler := cfg.buildFinalHandler()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	finalHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServerConfig_BuildFinalHandler_WithInboundHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := &serverConfig{
		handler:           handler,
		customMiddlewares: make([]MiddlewareFunc, 0),
		inboundAllowedHeaders: map[string]bool{
			"Content-Type": true,
		},
		logger: internallgr.Default(),
	}

	finalHandler := cfg.buildFinalHandler()

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")

	rec := httptest.NewRecorder()
	finalHandler.ServeHTTP(rec, req)

	// Authorization should be removed by header control middleware
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWithWriteTimeout_Valid(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	server := NewNetHttpServer(WithHandler(handler), WithWriteTimeout(20*time.Second))
	assert.NotNil(t, server)
}

func TestWithWriteTimeout_BelowMinimumPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	assert.Panics(t, func() {
		NewNetHttpServer(WithHandler(handler), WithWriteTimeout(500*time.Millisecond))
	})
}

func TestWithIdleTimeout_Valid(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	server := NewNetHttpServer(WithHandler(handler), WithIdleTimeout(120*time.Second))
	assert.NotNil(t, server)
}

func TestWithIdleTimeout_BelowMinimumPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	assert.Panics(t, func() {
		NewNetHttpServer(WithHandler(handler), WithIdleTimeout(500*time.Millisecond))
	})
}

func TestWithAllowedInboundHeaders_Valid(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	server := NewNetHttpServer(
		WithHandler(handler),
		WithAllowedInboundHeaders("Content-Type", "X-Custom-Header"),
	)
	assert.NotNil(t, server)
}

func TestWithAllowedInboundHeaders_EmptyPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	assert.Panics(t, func() {
		NewNetHttpServer(WithHandler(handler), WithAllowedInboundHeaders())
	})
}

func TestWithAllowedInboundHeaders_EmptyNamePanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	assert.Panics(t, func() {
		NewNetHttpServer(WithHandler(handler), WithAllowedInboundHeaders("Content-Type", ""))
	})
}

func TestMapKeysToSlice_Empty(t *testing.T) {
	m := make(map[string]bool)
	result := mapKeysToSlice(m)
	assert.Nil(t, result)
}

func TestMapKeysToSlice_MultipleKeys(t *testing.T) {
	m := map[string]bool{
		"Content-Type": true,
		"X-Custom":     true,
		"Location":     true,
	}
	result := mapKeysToSlice(m)
	assert.Len(t, result, 3)
	assert.Contains(t, result, "Content-Type")
	assert.Contains(t, result, "X-Custom")
	assert.Contains(t, result, "Location")
}

func TestWithCrossOriginProtection_ValidOrigins(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	cfg := newServerConfig(
		WithHandler(handler),
		WithCrossOriginProtection("https://example.com", "https://trusted.com"),
	)

	// corsOptions should have been added (internal implementation detail)
	// Just verify no panic occurred, which means origins were valid
	assert.NotNil(t, cfg)
}

func TestWithCrossOriginProtection_Composable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	cfg := newServerConfig(
		WithHandler(handler),
		WithCrossOriginProtection("https://example.com"),
		WithCrossOriginProtection("https://trusted.com"),
	)

	// corsOptions should have been accumulated (composable)
	// Verify both calls completed without panic
	assert.NotNil(t, cfg)
}
