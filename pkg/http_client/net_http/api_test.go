package net_http

import (
	"net/http"
	"testing"
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
	if WithTimeout(10, 20) == nil {
		t.Fatalf("WithTimeout returned nil")
	}
}

func TestExists_pkg_http_client_net_http_api(t *testing.T) {}

func TestNewNetHttpClient_Wrapper(t *testing.T) {
	c, err := NewNetHttpClient(WithBaseURL("http://example"), WithHeader("K", "V"))
	if err != nil || c == nil {
		t.Fatalf("NewNetHttpClient failed: %v", err)
	}
}
