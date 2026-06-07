package net_http_server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCrossOriginMiddleware_NoConfig(t *testing.T) {
	mw := NewCrossOriginMiddleware()
	assert.NotNil(t, mw)
	// mw should be a func(http.Handler) http.Handler
}

func TestNewCrossOriginMiddleware_WithOrigins(t *testing.T) {
	mw := NewCrossOriginMiddleware(
		WithTrustedOrigins("https://example.com"),
	)
	assert.NotNil(t, mw)
}

func TestNewCrossOriginMiddleware_WrapsHandler(t *testing.T) {
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mw := NewCrossOriginMiddleware(
		WithTrustedOrigins("https://example.com"),
	)
	wrappedHandler := mw(innerHandler)
	assert.NotNil(t, wrappedHandler)
}

func TestCrossOriginMiddleware_SameOriginAllowed(t *testing.T) {
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := NewCrossOriginMiddleware(
		WithTrustedOrigins("https://example.com"),
	)
	wrappedHandler := mw(innerHandler)

	// Same-origin request (no Origin header or same host)
	req := httptest.NewRequest("POST", "https://example.com/api", nil)
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)

	// Should reach inner handler
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCrossOriginMiddleware_SafeMethodAllowed(t *testing.T) {
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := NewCrossOriginMiddleware(
		WithTrustedOrigins("https://example.com"),
	)
	wrappedHandler := mw(innerHandler)

	// GET is a safe method - always allowed even cross-origin
	req := httptest.NewRequest("GET", "https://example.com/api", nil)
	req.Header.Set("Origin", "https://other.com")
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCrossOriginMiddleware_TrustedOriginAllowed(t *testing.T) {
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := NewCrossOriginMiddleware(
		WithTrustedOrigins("https://trusted.com"),
	)
	wrappedHandler := mw(innerHandler)

	// Request from trusted origin
	req := httptest.NewRequest("POST", "https://example.com/api", nil)
	req.Header.Set("Origin", "https://trusted.com")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)

	// Should be allowed
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCrossOriginMiddleware_WithCustomDenyHandler(t *testing.T) {
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	denyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("CSRF Rejected"))
	})

	mw := NewCrossOriginMiddleware(
		WithTrustedOrigins("https://example.com"),
		WithDenyHandler(denyHandler),
	)
	wrappedHandler := mw(innerHandler)

	// Request from untrusted origin (will be denied)
	req := httptest.NewRequest("POST", "https://example.com/api", nil)
	req.Header.Set("Origin", "https://untrusted.com")
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)

	// Custom deny handler should be called
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "CSRF Rejected")
}

func TestCrossOriginMiddleware_MultipleOrigins(t *testing.T) {
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := NewCrossOriginMiddleware(
		WithTrustedOrigins("https://first.com", "https://second.com", "https://third.com"),
	)
	wrappedHandler := mw(innerHandler)

	// Test all trusted origins
	for _, origin := range []string{"https://first.com", "https://second.com", "https://third.com"} {
		req := httptest.NewRequest("POST", "https://example.com/api", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, "origin %s should be allowed", origin)
	}
}

func TestCrossOriginMiddleware_Composition(t *testing.T) {
	// Verify middleware can be composed with other handlers
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Processed", "true")
		w.WriteHeader(http.StatusOK)
	})

	mw := NewCrossOriginMiddleware(
		WithTrustedOrigins("https://example.com"),
	)
	wrappedHandler := mw(innerHandler)

	req := httptest.NewRequest("GET", "https://example.com/api", nil)
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "true", rec.Header().Get("X-Processed"))
}

func TestCrossOriginMiddleware_UntrustedOriginRejected(t *testing.T) {
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := NewCrossOriginMiddleware(
		WithTrustedOrigins("https://example.com"),
	)
	wrappedHandler := mw(innerHandler)

	// Request from untrusted origin
	req := httptest.NewRequest("POST", "https://example.com/api", nil)
	req.Header.Set("Origin", "https://untrusted.com")
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)

	// Should be rejected (403 Forbidden)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCrossOriginMiddleware_BypassPatternConfiguration(t *testing.T) {
	// Verify bypass patterns are properly configured
	mw := NewCrossOriginMiddleware(
		WithTrustedOrigins("https://trusted.com"),
		WithInsecureBypassPatterns("^/webhooks/.*", "^/health"),
	)
	assert.NotNil(t, mw)
	// Middleware wraps handler correctly
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrappedHandler := mw(innerHandler)
	assert.NotNil(t, wrappedHandler)
}
