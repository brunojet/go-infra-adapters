package net_http_server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHeaderControlMiddleware_NilHandler(t *testing.T) {
	assert.Panics(t, func() {
		NewHeaderControlMiddleware(nil)
	}, "should panic on nil handler")
}

func TestHeaderControlMiddleware_WithAllowedInboundHeaders_ConflictPanic(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	assert.Panics(t, func() {
		NewHeaderControlMiddleware(
			nextHandler,
			WithAllowedInboundHeaders("Authorization"),
		)
	}, "should panic when trying to whitelist hardcoded denied header")

	assert.Panics(t, func() {
		NewHeaderControlMiddleware(
			nextHandler,
			WithAllowedInboundHeaders("Cookie"),
		)
	}, "should panic when trying to whitelist hardcoded denied header")
}

func TestHeaderControlMiddleware_IntegrationFlow(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get("Cookie"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Internal-Debug", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	middleware := NewHeaderControlMiddleware(
		nextHandler,
		WithAllowedInboundHeaders("Content-Type", "X-Request-ID"),
	)

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Cookie", "session=abc")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-456")
	req.Header.Set("User-Agent", "test-client")

	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, "true", rec.Header().Get("X-Internal-Debug"))

	body, _ := io.ReadAll(rec.Body)
	assert.Equal(t, `{"status":"ok"}`, string(body))
}
