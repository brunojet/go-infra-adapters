package net_http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMergeConfigHeaders_ClientWins(t *testing.T) {
	c := &netHttpClient{headers: make(http.Header)}
	c.headers.Set("X-Client", "client")
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("X-Client", "request")
	c.mergeConfigHeaders(req)
	if got := req.Header.Get("X-Client"); got != "client" {
		t.Fatalf("expected client header to win, got %q", got)
	}
}

func TestBuildRequest_ResolveRelative(t *testing.T) {
	c := &netHttpClient{baseURL: "http://example.com/base/"}
	req, _ := http.NewRequest("GET", "/path", nil)
	r, err := c.buildRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	if r.URL.String() != "http://example.com/path" {
		t.Fatalf("unexpected URL: %s", r.URL.String())
	}
}

func TestBuildRequest_ParseBaseError(t *testing.T) {
	c := &netHttpClient{baseURL: "http://bad%z"}
	req, _ := http.NewRequest("GET", "/path", nil)
	r, err := c.buildRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("buildRequest error: %v", err)
	}
	if r.URL.String() != "/path" {
		t.Fatalf("expected relative URL, got %s", r.URL.String())
	}
}

func TestDo_UsesRoundTripperAndResolvesURL(t *testing.T) {
	c := &netHttpClient{client: &http.Client{Transport: rtFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Request: r}, nil
	})}, baseURL: "http://example.com"}

	req, _ := http.NewRequest("GET", "/path", bytes.NewReader([]byte("")))
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if resp.Request == nil || resp.Request.URL.String() != "http://example.com/path" {
		t.Fatalf("unexpected resolved URL: %+v", resp.Request)
	}
	// ensure body is closed to satisfy linters
	if resp.Body != nil {
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("resp body close failed: %v", err)
		}
	}
}

// Ensure the constructor returns a working client with defaults applied.
func TestNewNetHttpClient_Defaults(t *testing.T) {
	c, err := NewNetHttpClient()
	require.NoError(t, err)
	nhc, ok := c.(*netHttpClient)
	require.True(t, ok)
	require.NotNil(t, nhc.client)
	require.Equal(t, http.DefaultTransport, nhc.client.Transport)
	require.Equal(t, time.Duration(DefaultResponseTimeoutMs)*time.Millisecond, nhc.client.Timeout)
	require.Equal(t, "", nhc.baseURL)
	require.NotNil(t, nhc.headers)
}

// A RoundTripper that records the incoming request and returns a simple response.
type captureRT struct {
	last *http.Request
}

func (c *captureRT) RoundTrip(r *http.Request) (*http.Response, error) {
	c.last = r
	return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
}

func TestNewNetHttpClient_WithOptions_UsesRoundTripperAndHeadersAndTimeout(t *testing.T) {
	rt := &captureRT{}
	// small timeouts (ms) are fine for unit tests
	c, err := NewNetHttpClient(WithBaseURL("http://example.com"), WithHeader("X-Client", "clientval"), WithRoundTripper(rt), WithTimeout(5, 10))
	require.NoError(t, err)
	nhc, ok := c.(*netHttpClient)
	require.True(t, ok)

	// perform a request that will be handled by captureRT
	req, _ := http.NewRequest("GET", "/path", nil)
	resp, err := nhc.Do(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, 204, resp.StatusCode)

	// the captured request should have been resolved against base URL
	require.NotNil(t, rt.last)
	require.Equal(t, "http://example.com/path", rt.last.URL.String())

	// client header should be present (client headers win according to mergeConfigHeaders)
	require.Equal(t, "clientval", rt.last.Header.Get("X-Client"))

	// timeout should reflect response timeout set via WithTimeout
	require.Equal(t, time.Duration(10)*time.Millisecond, nhc.client.Timeout)

	// ensure body is closed to satisfy linters
	if resp.Body != nil {
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("resp body close failed: %v", err)
		}
	}
}
