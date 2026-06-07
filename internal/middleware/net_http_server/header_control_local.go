package net_http_server

import (
	"context"
	"net/http"

	"github.com/brunojet/go-infra-adapters/v4/internal/middleware/net_http_base"
)

// inboundHeaderTarget wraps an inbound request for sanitization
type inboundHeaderTarget struct {
	req *http.Request
}

func (t *inboundHeaderTarget) Headers() http.Header {
	return t.req.Header
}

func (t *inboundHeaderTarget) SanitizationContext() context.Context {
	return t.req.Context()
}

// sanitizeInboundHeaders removes denied/non-whitelisted headers from the request
func (m *headerControlMiddleware) sanitizeInboundHeaders(r *http.Request) {
	net_http_base.SanitizeHeaders(
		&inboundHeaderTarget{req: r},
		m.logger,
		m.inboundDeniedHeaders,
		m.inboundAllowedHeaders,
		net_http_base.InboundRequest,
	)
}
