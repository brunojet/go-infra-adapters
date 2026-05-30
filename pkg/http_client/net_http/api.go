// Package net_http provides a thin public facade over the internal
// net/http-based HTTP client implementation, exposing options and a
// lightweight constructor used across the repository.
package net_http

import (
	"net/http"

	"github.com/brunojet/go-infra-adapters/internal/http_client/net_http"
	"github.com/brunojet/go-infra-adapters/pkg/http_client/contracts"
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

// WithTimeout returns an option to set the connection and response timeouts
// for the HttpClient.
// Complexity: O(1). Memory: ~32 bytes.
func WithTimeout(connectTimeoutMs, responseTimeoutMs int) HttpClientOption {
	return net_http.WithTimeout(connectTimeoutMs, responseTimeoutMs)
}

// WithRoundTripper returns an option to set a custom http.RoundTripper for the
// HttpClient.
// Complexity: O(1). Memory: ~8-16 bytes (ref).
func WithRoundTripper(rt http.RoundTripper) HttpClientOption { return net_http.WithRoundTripper(rt) }
