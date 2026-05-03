package net_http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// helper type to build inline RoundTripper values
type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDefaultHttpClientConfig(t *testing.T) {
	cfg := defaultHttpClientConfig()
	assert.Equal(t, "", cfg.baseURL)
	assert.Equal(t, DefaultConnectTimeoutMs, cfg.connectTimeoutMs)
	assert.Equal(t, DefaultResponseTimeoutMs, cfg.responseTimeoutMs)
	if cfg.headers == nil {
		t.Fatalf("expected headers map to be initialized")
	}
	assert.Equal(t, http.DefaultTransport, cfg.roundTripper)
}

func TestNewHttpClientConfig_WithOptions(t *testing.T) {
	myRT := rtFunc(func(r *http.Request) (*http.Response, error) { return nil, nil })
	cfg := newHttpClientConfig(WithBaseURL("https://api.example"), WithTimeout(123, 456), WithHeader("X-Test", "ok"), WithRoundTripper(myRT))

	assert.Equal(t, "https://api.example", cfg.baseURL)
	assert.Equal(t, 123, cfg.connectTimeoutMs)
	assert.Equal(t, 456, cfg.responseTimeoutMs)
	assert.Equal(t, "ok", cfg.headers.Get("X-Test"))
	assert.NotNil(t, cfg.roundTripper)
}
