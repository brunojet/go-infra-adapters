//go:build otel
// +build otel

package net_http_client

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewOTELRoundTripper creates client-side observability middleware using OpenTelemetry.
// Wraps the official otelhttp package for RoundTripper instrumentation.
//
// Captures:
// - Distributed traces with automatic trace context propagation (W3C)
// - Request/response metrics (duration, status codes, etc)
// - Automatic span creation per HTTP request
// - Request/response filtering and custom attributes support
//
// Usage:
//
//	rt := NewOTELRoundTripper(http.DefaultTransport)
//	client := &http.Client{
//		Transport: rt,
//	}
//
//	// Or with our HTTP client adapter:
//	client, _ := net_http.NewNetHttpClient(
//		net_http.WithRoundTripper(rt),
//	)
//
// Under the hood: Uses go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
// which is the official, battle-tested OpenTelemetry instrumentation for HTTP clients.
// Automatically propagates trace context to upstream services.
func NewOTELRoundTripper(rt http.RoundTripper, opts ...otelhttp.ClientOption) http.RoundTripper {
	// otelhttp.NewClient wraps a Transport with full tracing/metrics
	// Default options: automatic trace propagation, span creation per request
	return otelhttp.NewTransport(rt, opts...)
}
