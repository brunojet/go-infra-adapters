package net_http

import (
	"fmt"
	"net"
	"net/http"
	"time"

	internallgr "github.com/brunojet/go-infra-adapters/v4/internal/logger"
	middlmw "github.com/brunojet/go-infra-adapters/v4/internal/middleware/net_http_server"
	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

const (
	defaultAddr            = ":8080"
	defaultReadTimeout     = 15 * time.Second
	defaultWriteTimeout    = 15 * time.Second
	defaultIdleTimeout     = 60 * time.Second
	minReadTimeout         = 1 * time.Second
	minWriteTimeout        = 1 * time.Second
	minIdleTimeout         = 1 * time.Second
	errAddrInvalid         = "http server: invalid address: "
	errReadTimeoutMin      = "http server: read timeout must be >= "
	errWriteTimeoutMin     = "http server: write timeout must be >= "
	errIdleTimeoutMin      = "http server: idle timeout must be >= "
	errHandlerNil          = "http server: handler cannot be nil"
	errMiddlewareNil       = "http server: middleware cannot be nil"
	errInboundHeadersEmpty = "http server: inbound headers cannot be empty"
	errHeaderEmpty         = "http server: header name cannot be empty"
)

type serverConfig struct {
	handler               http.Handler
	addr                  string
	readTimeout           time.Duration
	writeTimeout          time.Duration
	idleTimeout           time.Duration
	observabilityMw       MiddlewareFunc              // Observability (OTEL, logging, tracing)
	customMiddlewares     []MiddlewareFunc            // Custom app-specific middlewares
	corsOptions           []middlmw.CrossOriginOption // CORS/CSRF options
	inboundAllowedHeaders map[string]bool
	logger                logger.Logger
}

type ServerOption func(cfg *serverConfig)

func newServerConfig(opts ...ServerOption) *serverConfig {
	cfg := &serverConfig{
		handler:               http.DefaultServeMux,
		addr:                  defaultAddr,
		readTimeout:           defaultReadTimeout,
		writeTimeout:          defaultWriteTimeout,
		idleTimeout:           defaultIdleTimeout,
		observabilityMw:       nil, // Optional
		customMiddlewares:     make([]MiddlewareFunc, 0),
		corsOptions:           make([]middlmw.CrossOriginOption, 0),
		inboundAllowedHeaders: make(map[string]bool),
		logger:                internallgr.Default(), // Never nil - always has a default
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

func WithHandler(h http.Handler) ServerOption {
	return func(cfg *serverConfig) {
		if h == nil {
			panic(errHandlerNil)
		}
		cfg.handler = h
	}
}

func WithAddr(addr string) ServerOption {
	return func(cfg *serverConfig) {
		// net.SplitHostPort validates: format (host:port), port is number, port range 1-65535
		// No DNS resolution happens here (hostname resolved at runtime)
		_, _, err := net.SplitHostPort(addr)
		if err != nil {
			panic(errAddrInvalid + err.Error())
		}

		cfg.addr = addr
	}
}

func WithReadTimeout(d time.Duration) ServerOption {
	return func(cfg *serverConfig) {
		if d < minReadTimeout {
			panic(fmt.Sprintf("%s%s (got %s)", errReadTimeoutMin, minReadTimeout, d))
		}
		cfg.readTimeout = d
	}
}

func WithWriteTimeout(d time.Duration) ServerOption {
	return func(cfg *serverConfig) {
		if d < minWriteTimeout {
			panic(fmt.Sprintf("%s%s (got %s)", errWriteTimeoutMin, minWriteTimeout, d))
		}
		cfg.writeTimeout = d
	}
}

func WithIdleTimeout(d time.Duration) ServerOption {
	return func(cfg *serverConfig) {
		if d < minIdleTimeout {
			panic(fmt.Sprintf("%s%s (got %s)", errIdleTimeoutMin, minIdleTimeout, d))
		}
		cfg.idleTimeout = d
	}
}

// WithObservabilityMiddleware sets the observability middleware (OTEL, logging, tracing).
// This middleware executes FIRST (outermost) and sees ALL requests, including those rejected by CORS.
// It's optional and designed for distributed tracing and metrics collection.
// Example: OTEL middleware, custom logging, request/response tracing.
func WithObservabilityMiddleware(fn MiddlewareFunc) ServerOption {
	return func(cfg *serverConfig) {
		if fn == nil {
			panic("http server: observability middleware cannot be nil")
		}
		cfg.observabilityMw = fn
	}
}

// WithCustomMiddleware adds a custom app-specific middleware to the chain.
// These middlewares execute AFTER CORS validation (only valid requests reach here).
// Use for auth, rate-limiting, custom logging, etc.
// Middlewares are applied in order and can be called multiple times (composable).
func WithCustomMiddleware(fn MiddlewareFunc) ServerOption {
	return func(cfg *serverConfig) {
		if fn == nil {
			panic(errMiddlewareNil)
		}
		cfg.customMiddlewares = append(cfg.customMiddlewares, fn)
	}
}

// Deprecated: Use WithCustomMiddleware instead. Kept for backwards compatibility.
func WithMiddleware(fn MiddlewareFunc) ServerOption {
	return WithCustomMiddleware(fn)
}

// WithAllowedInboundHeaders sets headers allowed in requests
// These are added to the hardcoded allowed list from header control middleware
// Duplicates are ignored (set semantics)
// Headers are canonicalized to standard HTTP header format (e.g., "content-type" → "Content-Type")
func WithAllowedInboundHeaders(headers ...string) ServerOption {
	return func(cfg *serverConfig) {
		if len(headers) == 0 {
			panic(errInboundHeadersEmpty)
		}
		for i, h := range headers {
			if h == "" {
				panic(fmt.Sprintf("%s at index %d", errHeaderEmpty, i))
			}
			canonical := http.CanonicalHeaderKey(h)
			cfg.inboundAllowedHeaders[canonical] = true
		}
	}
}

// WithLogger sets the logger for middleware events (header removals, security events)
// Logger is called for headers that are removed during sanitization
func WithLogger(log logger.Logger) ServerOption {
	if log == nil {
		panic("http server: logger cannot be nil")
	}
	return func(cfg *serverConfig) {
		cfg.logger = log
	}
}

// WithCrossOriginProtection enables CSRF protection by rejecting non-safe cross-origin requests.
// Origins are validated to have http:// or https:// schema.
// Safe methods (GET, HEAD, OPTIONS) are always allowed.
// Can be called multiple times to append more options (composable).
func WithCrossOriginProtection(origins ...string) ServerOption {
	return func(cfg *serverConfig) {
		cfg.corsOptions = append(cfg.corsOptions, middlmw.WithTrustedOrigins(origins...))
	}
}

// WithCrossOriginBypassPatterns adds regex patterns that bypass CSRF protection.
// Requests matching the patterns will NOT be checked for cross-origin.
// Use carefully - only for paths that truly need cross-origin requests.
// Examples: webhooks, public APIs, asset uploads
func WithCrossOriginBypassPatterns(patterns ...string) ServerOption {
	return func(cfg *serverConfig) {
		cfg.corsOptions = append(cfg.corsOptions, middlmw.WithInsecureBypassPatterns(patterns...))
	}
}

// WithCrossOriginDenyHandler sets custom handler for rejected cross-origin requests.
// If not set, uses default problem details response (RFC 7807).
func WithCrossOriginDenyHandler(h http.Handler) ServerOption {
	return func(cfg *serverConfig) {
		cfg.corsOptions = append(cfg.corsOptions, middlmw.WithDenyHandler(h))
	}
}

// buildFinalHandler composes all middlewares and returns the final handler.
//
// Wrapping order (code):      headerControl → customMWs → cors → observability
// Execution order (request):  observability → cors → customMWs → headerControl → handler
//
// Note: Last wrapped = First executed! Middlewares wrap each other like onions.
//
// Execution flow:
//  1. ObservabilityMiddleware (outermost) - sees ALL requests (OTEL, tracing, logging)
//     ↓ Always passes through (unless explicitly rejects)
//  2. CORS validation - rejects cross-origin if not allowed (fail-fast security)
//     ↓ If rejected: returns 403 + STOPS chain
//     ↓ If allowed: continues
//  3. CustomMiddlewares - app-specific logic (auth, rate-limit, etc)
//     ↓ Only executed if CORS passed
//  4. HeaderControl - sanitizes sensitive headers (Authorization, Cookie, etc)
//     ↓ Always active, removes dangerous headers before handler
//  5. Handler - app business logic
func (cfg *serverConfig) buildFinalHandler() http.Handler {
	// Start with handler (innermost)
	finalHandler := cfg.handler

	// Wrap 1: HeaderControl (innermost - executes last)
	// Always active to protect against header-based attacks
	mwOpts := []middlmw.HeaderControlOption{}
	if len(cfg.inboundAllowedHeaders) > 0 {
		mwOpts = append(mwOpts, middlmw.WithAllowedInboundHeaders(mapKeysToSlice(cfg.inboundAllowedHeaders)...))
	}
	mwOpts = append(mwOpts, middlmw.WithLogger(cfg.logger))
	finalHandler = middlmw.NewHeaderControlMiddleware(finalHandler, mwOpts...)

	// Wrap 2: CustomMiddlewares (executes 2nd)
	// Only reached if CORS passed; intended for app-specific logic
	for _, mw := range cfg.customMiddlewares {
		finalHandler = mw(finalHandler)
	}

	// Wrap 3: CORS (executes 3rd)
	// Security gate: rejects cross-origin requests early (fail-fast)
	if len(cfg.corsOptions) > 0 {
		corsMw := middlmw.NewCrossOriginMiddleware(cfg.corsOptions...)
		finalHandler = corsMw(finalHandler)
	}

	// Wrap 4: ObservabilityMiddleware (outermost - executes first)
	// Optional: OTEL, distributed tracing, metrics
	// Sees ALL requests (even those rejected by CORS)
	if cfg.observabilityMw != nil {
		finalHandler = cfg.observabilityMw(finalHandler)
	}

	return finalHandler
}

// mapKeysToSlice converts a map[string]bool to a string slice (helper for headers)
func mapKeysToSlice(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
