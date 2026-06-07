// Package net_http provides a thin public facade over the internal
// net/http-based HTTP client implementation, exposing options and a
// lightweight constructor used across the repository.
package net_http

import (
	"net/http"
	"time"

	"github.com/brunojet/go-infra-adapters/v4/internal/http_client/net_http"
	"github.com/brunojet/go-infra-adapters/v4/pkg/http_client/contracts"
)

// HttpClient is the public alias for the internal HTTP client contract.
type HttpClient = contracts.HttpClient

// HttpClientOption configures the public HttpClient facade and maps
// directly to the internal option type.
type HttpClientOption = net_http.HttpClientOption

// NewNetHttpClient creates a new instance of the net/http-based HttpClient
// adapter with the provided options.
// Complexity: O(1). Memory: ~2-3 KB for client state.
func NewNetHttpClient(opts ...HttpClientOption) (HttpClient, error) {
	return net_http.NewNetHttpClient(opts...)
}

// WithBaseURL returns an option to set the base URL for the HttpClient.
// Complexity: O(n) where n = len(url). Memory: ~16 bytes + string.
func WithBaseURL(url string) net_http.HttpClientOption { return net_http.WithBaseURL(url) }

// WithHeader returns an option to set a default header for the HttpClient.
// Complexity: O(n) where n = len(key) + len(value). Memory: ~16 bytes + strings.
func WithHeader(key, value string) net_http.HttpClientOption { return net_http.WithHeader(key, value) }

// WithTimeout returns an option to set the overall request timeout
// for the HttpClient.
// Complexity: O(1). Memory: ~8 bytes.
func WithTimeout(timeout time.Duration) HttpClientOption {
	return net_http.WithTimeout(timeout)
}

// WithRoundTripper returns an option to set a custom http.RoundTripper for the
// HttpClient.
// Complexity: O(1). Memory: ~8-16 bytes (ref).
func WithRoundTripper(rt http.RoundTripper) HttpClientOption { return net_http.WithRoundTripper(rt) }

// WithObservabilityMiddleware returns an option to add observability middleware (OTEL, tracing).
// This middleware executes FIRST (outermost) and wraps everything else.
// It sees ALL requests before any other processing.
// Optional - use for distributed tracing and metrics collection.
// Complexity: O(1). Memory: ~8 bytes (ref).
func WithObservabilityMiddleware(mw net_http.MiddlewareFunc) HttpClientOption {
	return net_http.WithObservabilityMiddleware(mw)
}

// WithCustomMiddleware returns an option to add a custom app-specific middleware to the HttpClient.
// These middlewares execute AFTER observability and AFTER header control.
// Middlewares are applied in order and can be called multiple times (composable).
// Example: circuit breaker, retry, custom logging, etc.
// Complexity: O(1). Memory: ~8 bytes (ref).
func WithCustomMiddleware(mw net_http.MiddlewareFunc) HttpClientOption {
	return net_http.WithCustomMiddleware(mw)
}

// WithMiddleware adds middleware to the HTTP client.
//
// Deprecated: Use WithCustomMiddleware instead.
func WithMiddleware(mw net_http.MiddlewareFunc) HttpClientOption {
	return net_http.WithMiddleware(mw)
}

// WithMaxIdleConns returns an option to set the maximum number of idle connections.
// Complexity: O(1). Memory: ~4 bytes.
func WithMaxIdleConns(maxIdle int) HttpClientOption {
	return net_http.WithMaxIdleConns(maxIdle)
}

// WithMaxIdleConnsPerHost returns an option to set the maximum number of idle connections per host.
// Default is 2. Complexity: O(1). Memory: ~4 bytes.
func WithMaxIdleConnsPerHost(maxIdlePerHost int) HttpClientOption {
	return net_http.WithMaxIdleConnsPerHost(maxIdlePerHost)
}

// WithIdleConnTimeout returns an option to set the idle connection timeout.
// Complexity: O(1). Memory: ~8 bytes.
func WithIdleConnTimeout(timeout time.Duration) HttpClientOption {
	return net_http.WithIdleConnTimeout(timeout)
}

// WithDialContext returns an option to set both dial timeout and keep-alive period.
// Both parameters are applied atomically to prevent state inconsistency.
// Complexity: O(1). Memory: ~16 bytes.
func WithDialContext(timeout, keepAlive time.Duration) HttpClientOption {
	return net_http.WithDialContext(timeout, keepAlive)
}
