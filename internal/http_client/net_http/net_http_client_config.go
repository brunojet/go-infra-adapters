package net_http

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"

	internallgr "github.com/brunojet/go-infra-adapters/v4/internal/logger"
	middlmw "github.com/brunojet/go-infra-adapters/v4/internal/middleware/net_http_client"
	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

// MiddlewareFunc is a function that wraps an http.RoundTripper with middleware
type MiddlewareFunc func(http.RoundTripper) http.RoundTripper

type httpClientConfig struct {
	baseURL           string
	roundTripper      http.RoundTripper
	observabilityMw   MiddlewareFunc   // Observability (OTEL, tracing)
	customMiddlewares []MiddlewareFunc // Custom app-specific middlewares
	timeout           time.Duration
	headers           http.Header
	logger            logger.Logger
}

// defaultHttpClientConfig returns a sensible default configuration.
// Uses http.DefaultTransport from stdlib (already configured with sensible defaults).
// Default timeout is 0 (no timeout, matching stdlib behavior).
func defaultHttpClientConfig() httpClientConfig {
	return httpClientConfig{
		baseURL:           "",
		roundTripper:      http.DefaultTransport,
		observabilityMw:   nil, // Optional
		customMiddlewares: make([]MiddlewareFunc, 0),
		timeout:           0, // 0 means no timeout (stdlib default)
		headers:           make(http.Header),
		logger:            internallgr.Default(),
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

// NOTE: Header control middleware is automatically applied and acts as the
// second middleware layer (after Observability). It removes dangerous headers (Authorization, Cookie,
// X-Forwarded-*, etc.) ensuring they never reach upstream services.
// Custom middlewares (circuit breaker, etc.) are applied after header control and work with
// sanitized headers, so they can safely add headers without risk of sanitization.

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

// WithTimeout sets the client timeout (overall request timeout).
// timeout is applied to the http.Client, affecting all requests.
func WithTimeout(timeout time.Duration) HttpClientOption {
	return func(c *httpClientConfig) {
		if timeout <= 0 {
			panic("timeout must be > 0")
		}
		c.timeout = timeout
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
// The provided RoundTripper acts as the base for the middleware chain.
// It will be wrapped by HeaderControl, then by CustomMiddleware (if any), then by Observability.
//
// Example:
//
//	breaker := middleware.NewBreakerMiddleware(http.DefaultTransport, ...)
//	client := NewNetHttpClient(
//		WithRoundTripper(breaker),
//		WithCustomMiddleware(otherMW),
//	)
//	// Execution order: Observability → HeaderControl → CustomMiddleware → breaker
func WithRoundTripper(rt http.RoundTripper) HttpClientOption {
	return func(c *httpClientConfig) {
		if rt == nil {
			panic("roundTripper cannot be nil")
		}
		c.roundTripper = rt
	}
}

// WithLogger sets the logger for security events (header sanitization).
func WithLogger(log logger.Logger) HttpClientOption {
	return func(c *httpClientConfig) {
		if log == nil {
			panic("http client: logger cannot be nil")
		}
		c.logger = log
	}
}

// WithObservabilityMiddleware sets the observability middleware (OTEL, tracing).
// This middleware executes FIRST (outermost) and wraps everything else.
// It sees ALL requests before any other processing.
// Optional - use for distributed tracing and metrics collection.
func WithObservabilityMiddleware(mw MiddlewareFunc) HttpClientOption {
	return func(c *httpClientConfig) {
		if mw == nil {
			panic("http client: observability middleware cannot be nil")
		}
		c.observabilityMw = mw
	}
}

// WithCustomMiddleware adds a custom app-specific middleware to the client RoundTripper chain.
// These middlewares execute AFTER observability and AFTER header control.
// Example: circuit breaker, retry, custom logging, etc.
// Middlewares are applied in order and can be called multiple times (composable).
func WithCustomMiddleware(mw MiddlewareFunc) HttpClientOption {
	return func(c *httpClientConfig) {
		if mw == nil {
			panic("http client: middleware cannot be nil")
		}
		c.customMiddlewares = append(c.customMiddlewares, mw)
	}
}

// Deprecated: Use WithCustomMiddleware instead. Kept for backwards compatibility.
func WithMiddleware(mw MiddlewareFunc) HttpClientOption {
	return WithCustomMiddleware(mw)
}

// WithMaxIdleConns sets the maximum number of idle connections.
// Modifies the Transport directly if it's an *http.Transport.
func WithMaxIdleConns(maxIdle int) HttpClientOption {
	return func(c *httpClientConfig) {
		if maxIdle <= 0 {
			panic("http client: max idle conns must be > 0")
		}
		if transport, ok := c.roundTripper.(*http.Transport); ok {
			transport.MaxIdleConns = maxIdle
		}
	}
}

// WithMaxIdleConnsPerHost sets the maximum number of idle connections per host.
// Default is 2. Modifies the Transport directly if it's an *http.Transport.
func WithMaxIdleConnsPerHost(maxIdlePerHost int) HttpClientOption {
	return func(c *httpClientConfig) {
		if maxIdlePerHost <= 0 {
			panic("http client: max idle conns per host must be > 0")
		}
		if transport, ok := c.roundTripper.(*http.Transport); ok {
			transport.MaxIdleConnsPerHost = maxIdlePerHost
		}
	}
}

// WithIdleConnTimeout sets the timeout for idle connections.
// Modifies the Transport directly if it's an *http.Transport.
func WithIdleConnTimeout(timeout time.Duration) HttpClientOption {
	return func(c *httpClientConfig) {
		if timeout <= 0 {
			panic("http client: idle conn timeout must be > 0")
		}
		if transport, ok := c.roundTripper.(*http.Transport); ok {
			transport.IdleConnTimeout = timeout
		}
	}
}

// defaultTransportDialContext creates a DialContext with timeout and keep-alive settings.
// Both parameters are applied together to avoid state loss when setting one without the other.
func defaultTransportDialContext(timeout, keepAlive time.Duration) func(context.Context, string, string) (net.Conn, error) {
	return (&net.Dialer{
		Timeout:   timeout,
		KeepAlive: keepAlive,
	}).DialContext
}

// WithDialContext sets both the dial timeout and keep-alive period for establishing connections.
// Modifies the Transport's DialContext directly if it's an *http.Transport.
// Both parameters are applied atomically to prevent state inconsistency.
func WithDialContext(timeout, keepAlive time.Duration) HttpClientOption {
	return func(c *httpClientConfig) {
		if timeout <= 0 {
			panic("http client: dial timeout must be > 0")
		}
		if keepAlive <= 0 {
			panic("http client: dial keep-alive must be > 0")
		}
		if transport, ok := c.roundTripper.(*http.Transport); ok {
			transport.DialContext = defaultTransportDialContext(timeout, keepAlive)
		}
	}
}

// buildFinalRoundTripper composes RoundTripper middlewares.
//
// Wrapping order (code):      baseRT → customMWs → headerControl → observability
// Execution order (request):  observability → headerControl → customMWs → baseRT
//
// Note: Last wrapped = First executed! Middlewares wrap each other like onions.
//
// Execution flow:
//  1. ObservabilityMiddleware (outermost) - OTEL tracing, metrics (optional)
//     ↓ Sees original request, logs/traces everything
//  2. HeaderControl - sanitizes sensitive headers (Authorization, Cookie, etc)
//     ↓ Always active, removes dangerous headers BEFORE custom middleware
//     ↓ Security baseline: ensures custom MW doesn't accidentally leak credentials
//  3. CustomMiddlewares - circuit breaker, retry, custom logic (optional)
//     ↓ Works with sanitized headers (safe to implement custom logic)
//     ↓ Retry/circuit-breaker don't need Authorization to function
//  4. BaseRoundTripper - sends request to server
func (cfg *httpClientConfig) buildFinalRoundTripper() http.RoundTripper {
	// Wrap 1: BaseRoundTripper (innermost - executes last)
	finalRT := cfg.roundTripper

	// Wrap 2: CustomMiddlewares (execute 2nd)
	// Circuit breaker, retry, custom logic - work with sanitized headers
	// These are developer-controlled, but protected by HeaderControl below
	for _, mw := range cfg.customMiddlewares {
		finalRT = mw(finalRT)
	}

	// Wrap 3: HeaderControl (executes 3rd - BEFORE custom middlewares)
	// Sanitizes headers before custom logic runs
	// Ensures even buggy custom middleware can't leak Authorization/Cookie
	finalRT = middlmw.NewHeaderControlMiddleware(
		finalRT,
		middlmw.WithLogger(cfg.logger),
	)

	// Wrap 4: ObservabilityMiddleware (outermost - executes first)
	// Optional: OTEL tracing, metrics
	// Sees all requests in their original form before any processing
	if cfg.observabilityMw != nil {
		finalRT = cfg.observabilityMw(finalRT)
	}

	return finalRT
}
