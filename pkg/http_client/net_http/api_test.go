package net_http

import (
	"net/http"
	"testing"
	"time"
)

func TestApiWrappers_ReturnOptions(t *testing.T) {
	if WithBaseURL("https://x") == nil {
		t.Fatalf("WithBaseURL returned nil")
	}
	if WithHeader("K", "V") == nil {
		t.Fatalf("WithHeader returned nil")
	}
	if WithRoundTripper(http.DefaultTransport) == nil {
		t.Fatalf("WithRoundTripper returned nil")
	}
	if WithTimeout(100*time.Millisecond) == nil {
		t.Fatalf("WithTimeout returned nil")
	}
	if WithMaxIdleConns(50) == nil {
		t.Fatalf("WithMaxIdleConns returned nil")
	}
	if WithMaxIdleConnsPerHost(10) == nil {
		t.Fatalf("WithMaxIdleConnsPerHost returned nil")
	}
	if WithIdleConnTimeout(90*time.Second) == nil {
		t.Fatalf("WithIdleConnTimeout returned nil")
	}
	if WithDialContext(30*time.Second, 30*time.Second) == nil {
		t.Fatalf("WithDialContext returned nil")
	}
	if WithMiddleware(func(rt http.RoundTripper) http.RoundTripper { return rt }) == nil {
		t.Fatalf("WithMiddleware returned nil")
	}
}

func TestExists_pkg_http_client_net_http_api(t *testing.T) {}

func TestNewNetHttpClient_Wrapper(t *testing.T) {
	c, err := NewNetHttpClient(WithBaseURL("http://example"), WithHeader("K", "V"))
	if err != nil || c == nil {
		t.Fatalf("NewNetHttpClient failed: %v", err)
	}
}
