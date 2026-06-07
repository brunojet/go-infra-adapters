package net_http_server

import (
	"fmt"
	"maps"
	"net/http"

	"github.com/brunojet/go-infra-adapters/v4/internal/logger"
	"github.com/brunojet/go-infra-adapters/v4/internal/middleware/net_http_base"
	logpkg "github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

// Server-side default denied headers for INBOUND (before handler receives them)
// Combines common denied headers with server-specific ones
var (
	defaultInboundDeniedHeaders = buildServerInboundDeniedHeaders()
)

func buildServerInboundDeniedHeaders() map[string]bool {
	// Start with common denied headers (credentials, hop-by-hop, proxy)
	// Add request-specific denied headers (Host, User-Agent, Accept-Encoding, etc)
	return net_http_base.BuildDeniedHeaders(net_http_base.DeniedHeadersRequestOnly)
}

type headerControlConfig struct {
	inboundDeniedHeaders  map[string]bool
	inboundAllowedHeaders map[string]bool
	logger                logpkg.Logger
}

type HeaderControlOption func(cfg *headerControlConfig)

func newHeaderControlConfig(opts ...HeaderControlOption) *headerControlConfig {
	cfg := &headerControlConfig{
		inboundDeniedHeaders:  maps.Clone(defaultInboundDeniedHeaders),
		inboundAllowedHeaders: make(map[string]bool),
		logger:                logger.Default(), // Never nil - always has a default
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

func WithAllowedInboundHeaders(headers ...string) HeaderControlOption {
	return func(cfg *headerControlConfig) {
		if len(headers) == 0 {
			panic("header control: allowed inbound headers cannot be empty")
		}
		for i, h := range headers {
			if h == "" {
				panic("header control: inbound header name cannot be empty at index " + fmt.Sprint(i))
			}
			// Normalize using HTTP canonical form
			canonical := http.CanonicalHeaderKey(h)
			if cfg.inboundDeniedHeaders[canonical] {
				panic("header control: cannot override hardcoded security header '" + canonical + "' in inbound denied list")
			}
			cfg.inboundAllowedHeaders[canonical] = true
		}
	}
}

func WithLogger(log logpkg.Logger) HeaderControlOption {
	return func(cfg *headerControlConfig) {
		if log == nil {
			panic("header control: logger cannot be nil")
		}
		cfg.logger = log
	}
}
