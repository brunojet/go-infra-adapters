package net_http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHttpClientOptions_Validation(t *testing.T) {
	cfg := defaultHttpClientConfig()

	// WithBaseURL valid
	WithBaseURL("http://example.com")(&cfg)
	if cfg.baseURL != "http://example.com" {
		t.Fatalf("expected baseURL set")
	}
	// invalid base should panic
	require.Panics(t, func() { WithBaseURL("http://%zz")(&cfg) })

	// WithTimeout validations
	WithTimeout(100, 200)(&cfg)
	if cfg.connectTimeoutMs != 100 || cfg.responseTimeoutMs != 200 {
		t.Fatalf("expected timeouts set")
	}
	require.Panics(t, func() { WithTimeout(0, 100)(&cfg) })
	require.Panics(t, func() { WithTimeout(100, 0)(&cfg) })

	// WithHeader validations
	WithHeader("k", "v")(&cfg)
	if cfg.headers.Get("k") != "v" {
		t.Fatalf("expected header set")
	}
	require.Panics(t, func() { WithHeader("", "v")(&cfg) })
	require.Panics(t, func() { WithHeader("k", "")(&cfg) })

	// WithRoundTripper nil
	rt := rtFunc(func(r *http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200}, nil })
	WithRoundTripper(rt)(&cfg)
	require.NotNil(t, cfg.roundTripper)
	require.Panics(t, func() { WithRoundTripper(nil)(&cfg) })
}

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNewHttpClientConfig_PanicOnInvalidOptions(t *testing.T) {
	require.Panics(t, func() { newHttpClientConfig(WithBaseURL("http://%zz")) })
	require.Panics(t, func() { newHttpClientConfig(WithTimeout(0, 100)) })
	require.Panics(t, func() { newHttpClientConfig(WithTimeout(100, 0)) })
	require.Panics(t, func() { newHttpClientConfig(WithHeader("", "v")) })
	require.Panics(t, func() { newHttpClientConfig(WithHeader("k", "")) })
	require.Panics(t, func() { newHttpClientConfig(WithRoundTripper(nil)) })
}

func TestNewHttpClientConfig_NilOptionSkipped(t *testing.T) {
	cfg := newHttpClientConfig(nil, WithBaseURL("http://example.com"), nil)
	if cfg.baseURL != "http://example.com" {
		t.Fatalf("expected baseURL set despite nil options, got %q", cfg.baseURL)
	}
}

func TestNewHttpClientConfig_ValidOptions(t *testing.T) {
	rt := rtFunc(func(r *http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Request: r}, nil })
	cfg := newHttpClientConfig(WithBaseURL("http://example.com"), WithTimeout(100, 200), WithHeader("X-K", "v"), WithRoundTripper(rt))
	require.Equal(t, "http://example.com", cfg.baseURL)
	require.Equal(t, 100, cfg.connectTimeoutMs)
	require.Equal(t, 200, cfg.responseTimeoutMs)
	require.Equal(t, "v", cfg.headers.Get("X-K"))
	require.NotNil(t, cfg.roundTripper)
	_, ok := cfg.roundTripper.(rtFunc)
	require.True(t, ok, "expected roundTripper to be of rtFunc type")
}
