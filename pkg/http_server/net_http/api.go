// Package net_http provides a public facade for HTTP server configuration using net/http.
package net_http

import (
	"net/http"
	"time"

	impl "github.com/brunojet/go-infra-adapters/v4/internal/http_server/net_http"
	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

// MiddlewareFunc is a function that wraps an http.Handler with middleware.
type MiddlewareFunc = impl.MiddlewareFunc

// ServerOption configures a Server.
type ServerOption = impl.ServerOption

// Server represents an HTTP server instance.
type Server = impl.Server

// NewNetHttpServer creates a new HTTP server with the provided options.
func NewNetHttpServer(opts ...ServerOption) Server {
	return impl.NewNetHttpServer(opts...)
}

// WithHandler sets the request handler for the server.
func WithHandler(h http.Handler) ServerOption {
	return impl.WithHandler(h)
}

// WithAddr sets the server bind address (host:port).
func WithAddr(addr string) ServerOption {
	return impl.WithAddr(addr)
}

// WithReadTimeout sets the request read timeout.
func WithReadTimeout(d time.Duration) ServerOption {
	return impl.WithReadTimeout(d)
}

// WithWriteTimeout sets the response write timeout.
func WithWriteTimeout(d time.Duration) ServerOption {
	return impl.WithWriteTimeout(d)
}

// WithIdleTimeout sets the idle connection timeout.
func WithIdleTimeout(d time.Duration) ServerOption {
	return impl.WithIdleTimeout(d)
}

// WithObservabilityMiddleware sets the observability middleware (OTEL, tracing).
func WithObservabilityMiddleware(fn MiddlewareFunc) ServerOption {
	return impl.WithObservabilityMiddleware(fn)
}

// WithCustomMiddleware adds a custom app-specific middleware to the server.
func WithCustomMiddleware(fn MiddlewareFunc) ServerOption {
	return impl.WithCustomMiddleware(fn)
}

// WithMiddleware adds middleware to the server.
//
// Deprecated: Use WithCustomMiddleware instead.
func WithMiddleware(fn MiddlewareFunc) ServerOption {
	return impl.WithMiddleware(fn)
}

// WithAllowedInboundHeaders sets headers allowed in requests.
func WithAllowedInboundHeaders(headers ...string) ServerOption {
	return impl.WithAllowedInboundHeaders(headers...)
}

// WithLogger sets the logger for security events.
func WithLogger(log logger.Logger) ServerOption {
	return impl.WithLogger(log)
}

// WithCrossOriginProtection enables CSRF/CORS protection with trusted origins.
func WithCrossOriginProtection(origins ...string) ServerOption {
	return impl.WithCrossOriginProtection(origins...)
}

// WithCrossOriginBypassPatterns adds regex patterns that bypass CORS validation.
func WithCrossOriginBypassPatterns(patterns ...string) ServerOption {
	return impl.WithCrossOriginBypassPatterns(patterns...)
}

// WithCrossOriginDenyHandler sets a custom handler for rejected cross-origin requests.
func WithCrossOriginDenyHandler(h http.Handler) ServerOption {
	return impl.WithCrossOriginDenyHandler(h)
}
