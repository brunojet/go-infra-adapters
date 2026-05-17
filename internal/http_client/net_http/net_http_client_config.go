package net_http

import (
	"net/http"
	"net/url"
)

const (
	DefaultConnectTimeoutMs  = 15 * 1000
	DefaultResponseTimeoutMs = 30 * 1000
)

type httpClientConfig struct {
	baseURL           string
	roundTripper      http.RoundTripper
	headers           http.Header
	connectTimeoutMs  int
	responseTimeoutMs int
}

// defaultHttpClientConfig returns a sensible default configuration.
func defaultHttpClientConfig() httpClientConfig {
	return httpClientConfig{
		baseURL:           "",
		roundTripper:      http.DefaultTransport,
		headers:           make(http.Header),
		connectTimeoutMs:  DefaultConnectTimeoutMs,
		responseTimeoutMs: DefaultResponseTimeoutMs,
	}
}

// HttpClientOption configures a HttpClientConfig. Options panic on invalid input.
type HttpClientOption func(cfg *httpClientConfig)

// newHttpClientConfig builds a HttpClientConfig applying provided options.
func newHttpClientConfig(opts ...HttpClientOption) httpClientConfig {
	cfg := defaultHttpClientConfig()
	for _, o := range opts {
		if o == nil {
			continue
		}
		o(&cfg)
	}
	return cfg
}

// NOTE: Circuit breakers are provided as standard http.RoundTripper
// middleware. Use `WithRoundTripper(...)` and supply the BreakerRoundTripper
// (or any other RoundTripper) to enable circuit-breaking behavior.

// NOTE: We intentionally avoid a single-blob `WithConfig` helper because the
// functional options are small and composable (WithBaseURL/WithHeaders/etc.).

// WithBaseURL sets the base URL used by the client.
func WithBaseURL(base string) HttpClientOption {
	return func(c *httpClientConfig) {
		if _, err := url.Parse(base); err != nil {
			panic("invalid base URL")
		}
		c.baseURL = base
	}
}

// WithTimeout sets the client timeout in milliseconds.
func WithTimeout(connectionTimeoutMs, responseTimeoutMs int) HttpClientOption {
	return func(c *httpClientConfig) {
		if connectionTimeoutMs <= 0 {
			panic("connection timeout must be > 0")
		}
		if responseTimeoutMs <= 0 {
			panic("response timeout must be > 0")
		}
		c.connectTimeoutMs = connectionTimeoutMs
		c.responseTimeoutMs = responseTimeoutMs
	}
}

// WithHeader sets a single header key/value.
func WithHeader(key, value string) HttpClientOption {
	return func(c *httpClientConfig) {
		if key == "" || value == "" {
			panic("header key and value cannot be empty")
		}
		c.headers.Set(key, value)
	}
}

// WithRoundTripper sets the custom http.RoundTripper used by the client.
// Caller is responsible for composing middleware into a single RoundTripper
// (e.g. breaker := middlewares.NewBreakerMiddleware(base); WithRoundTripper(breaker)).
func WithRoundTripper(rt http.RoundTripper) HttpClientOption {
	return func(c *httpClientConfig) {
		if rt == nil {
			panic("roundTripper cannot be nil")
		}
		c.roundTripper = rt
	}
}
