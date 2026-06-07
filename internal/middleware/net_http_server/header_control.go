package net_http_server

import (
	"net/http"

	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

type headerControlMiddleware struct {
	next                  http.Handler
	inboundDeniedHeaders  map[string]bool
	inboundAllowedHeaders map[string]bool
	logger                logger.Logger
}

func NewHeaderControlMiddleware(next http.Handler, opts ...HeaderControlOption) http.Handler {
	if next == nil {
		panic("header control middleware: next handler cannot be nil")
	}

	cfg := newHeaderControlConfig(opts...)

	return &headerControlMiddleware{
		next:                  next,
		inboundDeniedHeaders:  cfg.inboundDeniedHeaders,
		inboundAllowedHeaders: cfg.inboundAllowedHeaders,
		logger:                cfg.logger,
	}
}

func (m *headerControlMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Sanitize inbound headers from client
	m.sanitizeInboundHeaders(r)

	// Delegate to next handler without wrapping response
	// (server controls its own headers - no outbound sanitization needed)
	m.next.ServeHTTP(w, r)
}
