package net_http_client

import (
	"maps"

	internallgr "github.com/brunojet/go-infra-adapters/v4/internal/logger"
	"github.com/brunojet/go-infra-adapters/v4/internal/middleware/net_http_base"
	logpkg "github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

// Client-side default denied headers for REQUEST (before sending to upstream)
// and RESPONSE (after receiving from upstream)
var (
	defaultClientRequestDeniedHeaders  = buildClientRequestDeniedHeaders()
	defaultClientResponseDeniedHeaders = buildClientResponseDeniedHeaders()
)

func buildClientRequestDeniedHeaders() map[string]bool {
	// Start with common denied headers (credentials, hop-by-hop, proxy)
	result := net_http_base.BuildDeniedHeaders(net_http_base.DeniedHeadersRequestOnly)
	return result
}

func buildClientResponseDeniedHeaders() map[string]bool {
	// Start with common denied headers (credentials, hop-by-hop, proxy)
	// Add response-specific denied headers (Set-Cookie from upstream)
	return net_http_base.BuildDeniedHeaders(net_http_base.DeniedHeadersResponseOnly)
}

type headerControlConfig struct {
	requestDeniedHeaders  map[string]bool
	responseDeniedHeaders map[string]bool
	logger                logpkg.Logger
}

type HeaderControlOption func(cfg *headerControlConfig)

func newHeaderControlConfig(opts ...HeaderControlOption) *headerControlConfig {
	cfg := &headerControlConfig{
		requestDeniedHeaders:  maps.Clone(defaultClientRequestDeniedHeaders),
		responseDeniedHeaders: maps.Clone(defaultClientResponseDeniedHeaders),
		logger:                internallgr.Default(),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// WithLogger sets the logger for security events.
//
// Panics if log is nil.
func WithLogger(log logpkg.Logger) HeaderControlOption {
	return func(cfg *headerControlConfig) {
		if log == nil {
			panic("header control: logger cannot be nil")
		}
		cfg.logger = log
	}
}
