// Package net_http_client provides OpenTelemetry middleware wrappers for HTTP clients.
//go:build otel
// +build otel

package net_http_client

import (
	"net/http"

	impl "github.com/brunojet/go-infra-adapters/v4/internal/middleware/net_http_client"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewOTELRoundTripper creates client-side observability middleware with OpenTelemetry.
// Simple wrapper around the official otelhttp package - no reinvention, just simplification.
//
// Usage with HTTP client:
//
//	import (
//		"github.com/brunojet/go-infra-adapters/v4/pkg/http_client/net_http"
//		mw "github.com/brunojet/go-infra-adapters/v4/pkg/middleware/net_http_client"
//	)
//
//	// Create observability middleware (one-liner)
//	obsMw := mw.NewOTELRoundTripper(http.DefaultTransport)
//
//	// Option 1: Use directly with http.Client
//	client := &http.Client{
//		Transport: obsMw,
//	}
//
//	// Option 2: Use with our HTTP client adapter (recommended)
//	client, _ := net_http.NewNetHttpClient(
//		net_http.WithObservabilityMiddleware(obsMw),
//	)
//
// What it captures:
//   - Distributed traces (request lifecycle with W3C Trace Context propagation)
//   - Metrics: latency, request count, status codes
//   - Automatic trace context injection in headers
//   - Custom attributes support
//
// Powered by: go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
// (official OpenTelemetry HTTP client instrumentation)
//
// Note: Trace context is automatically propagated to upstream services,
// enabling distributed tracing across service boundaries.
func NewOTELRoundTripper(rt http.RoundTripper, opts ...otelhttp.ClientOption) http.RoundTripper {
	return impl.NewOTELRoundTripper(rt, opts...)
}
