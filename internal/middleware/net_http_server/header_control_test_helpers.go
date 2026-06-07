package net_http_server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// serverHeaderTestCase represents a test scenario for server header sanitization
type serverHeaderTestCase struct {
	method          string
	path            string
	headers         map[string]string // headers to add to request
	allowedHeaders  []string          // whitelist
	expectedPresent map[string]bool   // headers that should be in the request
	expectedAbsent  map[string]bool   // headers that should NOT be in the request
}

// runServerHeaderTest executes a header control test for server middleware
func runServerHeaderTest(t *testing.T, tc *serverHeaderTestCase) {
	nextCalled := false
	var capturedReq *http.Request

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		capturedReq = r
		w.WriteHeader(http.StatusOK)
	})

	var opts []HeaderControlOption
	if len(tc.allowedHeaders) > 0 {
		opts = append(opts, WithAllowedInboundHeaders(tc.allowedHeaders...))
	}

	middleware := NewHeaderControlMiddleware(nextHandler, opts...)

	req := httptest.NewRequestWithContext(context.Background(), tc.method, tc.path, http.NoBody)
	for key, value := range tc.headers {
		req.Header.Set(key, value)
	}

	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("next handler was not called")
	}

	if capturedReq == nil {
		t.Error("captured request is nil")
		return
	}

	// Check expected present headers
	for header := range tc.expectedPresent {
		if capturedReq.Header.Get(header) == "" {
			t.Errorf("expected header %s to be present, but was empty", header)
		}
	}

	// Check expected absent headers
	for header := range tc.expectedAbsent {
		if capturedReq.Header.Get(header) != "" {
			t.Errorf("expected header %s to be absent, but was: %s", header, capturedReq.Header.Get(header))
		}
	}
}
