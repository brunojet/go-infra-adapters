// Package net_http_server provides OpenTelemetry middleware wrappers for HTTP servers.
//go:build otel
// +build otel

package net_http_server

import (
	"net/http"

	impl "github.com/brunojet/go-infra-adapters/v4/internal/middleware/net_http_server"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewOTELMiddleware creates server-side observability middleware with OpenTelemetry.
// Simple wrapper around the official otelhttp package - no reinvention, just simplification.
//
// Usage with HTTP server:
//
//	import (
//		"github.com/brunojet/go-infra-adapters/v4/pkg/http_server/net_http"
//		mw "github.com/brunojet/go-infra-adapters/v4/pkg/middleware/net_http_server"
//	)
//
//	// Create observability middleware (one-liner)
//	obsMw := mw.NewOTELMiddleware("my-service")
//
//	// Use in server
//	server := net_http.NewNetHttpServer(
//		net_http.WithObservabilityMiddleware(obsMw),
//		net_http.WithHandler(handler),
//	)
//
// What it captures:
//   - Distributed traces (request lifecycle with W3C Trace Context)
//   - Metrics: latency, request/response size, status codes
//   - Request ID propagation
//   - Custom attributes support
//
// Powered by: go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
// (official OpenTelemetry HTTP instrumentation)
func NewOTELMiddleware(serviceName string, opts ...otelhttp.Option) func(http.Handler) http.Handler {
	return impl.NewOTELMiddleware(serviceName, opts...)
}
