package net_http_client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// clientHeaderTestCase represents a test scenario for client header sanitization
type clientHeaderTestCase struct {
	method          string
	url             string
	headers         map[string]string // headers to add to request
	expectedPresent map[string]bool   // headers that should be in the request
	expectedAbsent  map[string]bool   // headers that should NOT be in the request
}

// runClientHeaderTest executes a header control test for client RoundTripper middleware
func runClientHeaderTest(t *testing.T, tc *clientHeaderTestCase) {
	capturedReq := &capturedRequestRT{}

	middleware := NewHeaderControlMiddleware(capturedReq)

	req := httptest.NewRequestWithContext(context.Background(), tc.method, tc.url, http.NoBody)
	for key, value := range tc.headers {
		req.Header.Set(key, value)
	}

	resp, err := middleware.RoundTrip(req)
	if err != nil {
		t.Errorf("RoundTrip failed: %v", err)
		return
	}

	if resp == nil {
		t.Error("response is nil")
		return
	}

	if capturedReq.req == nil {
		t.Error("captured request is nil")
		return
	}

	// Check expected present headers
	for header := range tc.expectedPresent {
		if capturedReq.req.Header.Get(header) == "" {
			t.Errorf("expected header %s to be present, but was empty", header)
		}
	}

	// Check expected absent headers
	for header := range tc.expectedAbsent {
		if capturedReq.req.Header.Get(header) != "" {
			t.Errorf("expected header %s to be absent, but was: %s", header, capturedReq.req.Header.Get(header))
		}
	}

	// Close response body
	if resp.Body != nil {
		_ = resp.Body.Close()
	}
}

// capturedRequestRT is a test RoundTripper that captures the request
type capturedRequestRT struct {
	req *http.Request
}

func (c *capturedRequestRT) RoundTrip(req *http.Request) (*http.Response, error) {
	c.req = req
	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       http.NoBody,
	}, nil
}
