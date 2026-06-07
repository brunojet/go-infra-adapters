package net_http_client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeaderControlRoundTripper_SanitizesRequestHeaders(t *testing.T) {
	runClientHeaderTest(t, &clientHeaderTestCase{
		method: "GET",
		url:    "https://example.com/api",
		headers: map[string]string{
			"Authorization": "Bearer token123",
			"Content-Type":  "application/json",
			"X-Custom":      "value",
		},
		expectedAbsent: map[string]bool{
			"Authorization": true,
		},
		expectedPresent: map[string]bool{
			"Content-Type": true,
			"X-Custom":     true,
		},
	})
}

func TestHeaderControlRoundTripper_RemovesDeniedHeaders(t *testing.T) {
	runClientHeaderTest(t, &clientHeaderTestCase{
		method: "GET",
		url:    "https://example.com/api",
		headers: map[string]string{
			"Authorization":   "Bearer token",
			"Cookie":          "session=123",
			"X-Forwarded-For": "192.168.1.1",
			"X-Custom-Header": "value",
		},
		expectedAbsent: map[string]bool{
			"Authorization":   true,
			"Cookie":          true,
			"X-Forwarded-For": true,
		},
		expectedPresent: map[string]bool{
			"X-Custom-Header": true,
		},
	})
}

func TestHeaderControlRoundTripper_CaseInsensitive(t *testing.T) {
	runClientHeaderTest(t, &clientHeaderTestCase{
		method: "GET",
		url:    "https://example.com/api",
		headers: map[string]string{
			"authorization": "Bearer token",
			"CONTENT-TYPE":  "application/json",
		},
		expectedAbsent: map[string]bool{
			"Authorization": true,
		},
		expectedPresent: map[string]bool{
			"Content-Type": true,
		},
	})
}

func TestHeaderControlRoundTripper_NextNilError(t *testing.T) {
	// Create with nil base (should use DefaultTransport)
	// But explicitly set next to nil for testing
	h := &headerControlRoundTripper{
		next:                  nil,
		requestDeniedHeaders:  defaultClientRequestDeniedHeaders,
		responseDeniedHeaders: defaultClientResponseDeniedHeaders,
		logger:                nil,
	}

	req := httptest.NewRequest("GET", "https://example.com/api", http.NoBody)

	resp, err := h.RoundTrip(req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestHeaderControlRoundTripper_DefaultTransport(t *testing.T) {
	// NewHeaderControlMiddleware with nil should use DefaultTransport
	middleware := NewHeaderControlMiddleware(nil)

	assert.NotNil(t, middleware)
	rt, ok := middleware.(*headerControlRoundTripper)
	assert.True(t, ok)
	assert.Equal(t, http.DefaultTransport, rt.next)
}

func TestHeaderControlRoundTripper_SanitizesResponseHeaders(t *testing.T) {
	// Mock transport that returns response with sensitive headers from upstream
	// Simulating a proxy scenario where upstream leaks credentials
	mockTransport := &mockRoundTripperWithResponse{
		returnHeaders: map[string]string{
			"Authorization":   "Bearer upstream-token",
			"Set-Cookie":      "session=abc123",
			"X-Forwarded-For": "10.0.0.1",
			"Content-Type":    "application/json",
			"X-Custom-Header": "safe-value",
		},
	}

	middleware := NewHeaderControlMiddleware(mockTransport)

	req := httptest.NewRequest("GET", "https://example.com/api", http.NoBody)
	resp, err := middleware.RoundTrip(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Denied headers from upstream should be removed (protect downstream client)
	assert.Empty(t, resp.Header.Get("Authorization"), "Authorization should be removed from response")
	assert.Empty(t, resp.Header.Get("Set-Cookie"), "Set-Cookie should be removed from response")
	assert.Empty(t, resp.Header.Get("X-Forwarded-For"), "X-Forwarded-For should be removed from response")

	// Content-Type and X-Custom-Header should remain (no whitelist, only denied removal)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"), "Content-Type should pass through")
	assert.Equal(t, "safe-value", resp.Header.Get("X-Custom-Header"), "X-Custom-Header should pass through")

	if resp.Body != nil {
		_ = resp.Body.Close()
	}
}

type mockRoundTripperWithResponse struct {
	returnHeaders map[string]string
}

func (m *mockRoundTripperWithResponse) RoundTrip(req *http.Request) (*http.Response, error) {
	// Return response with sensitive headers (simulating upstream)
	header := make(http.Header)
	for k, v := range m.returnHeaders {
		header.Set(k, v)
	}

	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     header,
		Body:       http.NoBody,
	}, nil
}
