package net_http

import (
	"net/http"

	"github.com/brunojet/go-infra-adapters/internal/http_clients/net_http"
	"github.com/brunojet/go-infra-adapters/pkg/http_clients/contracts"
)

type (
	HttpClient       = contracts.HttpClient
	HttpClientOption = net_http.HttpClientOption
)

// NewNetHttpClient creates a new instance of the net/http-based HttpClient adapter with the provided options.
func NewNetHttpClient(opts ...HttpClientOption) (HttpClient, error) {
	return net_http.NewNetHttpClient(opts...)
}

// WithBaseURL returns an option to set the base URL for the HttpClient.
func WithBaseURL(url string) net_http.HttpClientOption {
	return net_http.WithBaseURL(url)
}

// WithHeader returns an option to set a default header for the HttpClient.
func WithHeader(key, value string) net_http.HttpClientOption {
	return net_http.WithHeader(key, value)
}

// WithTimeout returns an option to set the connection and response timeouts for the HttpClient.
func WithTimeout(connectTimeoutMs, responseTimeoutMs int) HttpClientOption {
	return net_http.WithTimeout(connectTimeoutMs, responseTimeoutMs)
}

// WithRoundTripper returns an option to set a custom http.RoundTripper for the HttpClient.
func WithRoundTripper(rt http.RoundTripper) HttpClientOption {
	return net_http.WithRoundTripper(rt)
}
