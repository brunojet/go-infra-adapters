//go:build otel
// +build otel

package net_http_server

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewOTELMiddleware creates server-side observability middleware using OpenTelemetry.
// Wraps the official otelhttp package with sensible defaults.
//
// Captures:
// - Distributed traces (W3C Trace Context propagation)
// - Request/response metrics (duration, size, status codes)
// - Automatic span creation per request
// - Request/response filtering and custom attributes support
//
// Usage:
//
//	obsMw := NewOTELMiddleware("my-service")
//	server := NewNetHttpServer(
//		WithObservabilityMiddleware(obsMw),
//		WithHandler(handler),
//	)
//
// Under the hood: Uses go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
// which is the official, battle-tested OpenTelemetry instrumentation.
func NewOTELMiddleware(serviceName string, opts ...otelhttp.Option) func(http.Handler) http.Handler {
	// Use default global TracerProvider and MeterProvider if not configured
	return func(next http.Handler) http.Handler {
		// otelhttp.NewHandler wraps the handler with full tracing/metrics
		// Default options: captures all requests, automatic span creation
		return otelhttp.NewHandler(next, serviceName, opts...)
	}
}
