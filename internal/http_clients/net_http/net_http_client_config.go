package adapters

import (
	"errors"
	"log"
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

// HttpClientOption configures a HttpClientConfig when constructing a client.
type HttpClientOption func(cfg *httpClientConfig) error

// newHttpClientConfig builds a HttpClientConfig applying provided options.
func newHttpClientConfig(opts ...HttpClientOption) httpClientConfig {
	cfg := defaultHttpClientConfig()
	for _, o := range opts {
		if o == nil {
			continue
		}
		if err := o(&cfg); err != nil {
			log.Panicf("invalid http client configuration: %v", err)
		}
	}
	return cfg
}

// NOTE: Circuit breakers are provided as standard http.RoundTripper
// middleware. Use `WithRoundTripper(...)` and supply the BreakerRoundTripper
// (or any other RoundTripper) to enable circuit-breaking behaviour.

// NOTE: We intentionally avoid a single-blob `WithConfig` helper because the
// functional options are small and composable (WithBaseURL/WithHeaders/etc.).

// WithBaseURL sets the base URL used by the client.
func WithBaseURL(base string) HttpClientOption {
	return func(c *httpClientConfig) error {
		_, err := url.Parse(base)
		if err != nil {
			return err
		}
		c.baseURL = base
		return nil
	}
}

// WithTimeout sets the client timeout in milliseconds.
func WithTimeout(connectionTimeoutMs, responseTimeoutMs int) HttpClientOption {
	return func(c *httpClientConfig) error {
		if connectionTimeoutMs <= 0 {
			return errors.New("connection timeout must be > 0")
		}
		if responseTimeoutMs <= 0 {
			return errors.New("response timeout must be > 0")
		}
		c.connectTimeoutMs = connectionTimeoutMs
		c.responseTimeoutMs = responseTimeoutMs
		return nil
	}
}

// WithHeader sets a single header key/value.
func WithHeader(key, value string) HttpClientOption {
	return func(c *httpClientConfig) error {
		if key == "" || value == "" {
			return errors.New("header key and value cannot be empty")
		}
		c.headers.Set(key, value)
		return nil
	}
}

// WithRoundTripper sets the custom http.RoundTripper used by the client.
// Caller is responsible for composing middleware into a single RoundTripper
// (e.g. breaker := middlewares.NewBreakerMiddleware(base); WithRoundTripper(breaker)).
func WithRoundTripper(rt http.RoundTripper) HttpClientOption {
	return func(c *httpClientConfig) error {
		if rt == nil {
			return errors.New("roundTripper cannot be nil")
		}
		c.roundTripper = rt
		return nil
	}
}
