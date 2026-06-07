package net_http_client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/brunojet/go-infra-adapters/v4/internal/middleware/net_http_base"
	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

// headerControlRoundTripper wraps a downstream RoundTripper and sanitizes
// both request headers (before sending) and response headers (after receiving).
// Uses separate denied lists for each direction; no whitelist enforcement at client level.
type headerControlRoundTripper struct {
	next                  http.RoundTripper
	requestDeniedHeaders  map[string]bool
	responseDeniedHeaders map[string]bool
	logger                logger.Logger
}

// NewHeaderControlMiddleware returns a ready-to-use http.RoundTripper that wraps
// the provided base RoundTripper with client-side header control (sanitization).
// If base is nil, http.DefaultTransport will be used.
func NewHeaderControlMiddleware(base http.RoundTripper, opts ...HeaderControlOption) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}

	cfg := newHeaderControlConfig(opts...)

	return &headerControlRoundTripper{
		next:                  base,
		requestDeniedHeaders:  cfg.requestDeniedHeaders,
		responseDeniedHeaders: cfg.responseDeniedHeaders,
		logger:                cfg.logger,
	}
}

// RoundTrip sanitizes request headers before sending and response headers after receiving.
// This ensures sensitive headers from both client and upstream are filtered.
func (h *headerControlRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if h.next == nil {
		return nil, fmt.Errorf("header control middleware: next RoundTripper is nil")
	}

	// Sanitize request headers before sending to upstream
	h.sanitizeRequestHeaders(req)

	// Delegate to next middleware/transport
	resp, err := h.next.RoundTrip(req)

	// Sanitize response headers from upstream before returning to caller
	// Protects against upstream leaking sensitive headers (Set-Cookie, Authorization, etc)
	if resp != nil {
		h.sanitizeResponseHeaders(resp)
	}

	return resp, err
}

// requestHeaderTarget implements net_http_base.HeaderSanitizationTarget for http.Request
type requestHeaderTarget struct {
	req *http.Request
}

func (t *requestHeaderTarget) Headers() http.Header {
	return t.req.Header
}

func (t *requestHeaderTarget) SanitizationContext() context.Context {
	return t.req.Context()
}

// sanitizeRequestHeaders removes denied headers from the request before sending to upstream
func (h *headerControlRoundTripper) sanitizeRequestHeaders(req *http.Request) {
	net_http_base.SanitizeHeaders(
		&requestHeaderTarget{req: req},
		h.logger,
		h.requestDeniedHeaders,
		nil, // No whitelist for client requests (dev controls what to send)
		net_http_base.OutboundRequest,
	)
}

// responseHeaderTarget implements net_http_base.HeaderSanitizationTarget for http.Response
type responseHeaderTarget struct {
	resp *http.Response
}

func (t *responseHeaderTarget) Headers() http.Header {
	return t.resp.Header
}

func (t *responseHeaderTarget) SanitizationContext() context.Context {
	// Response doesn't have a direct context; use background
	// In future, could pass context from original request if needed
	return context.Background()
}

// sanitizeResponseHeaders removes denied headers from the response after receiving from upstream
// This protects against upstream services leaking sensitive headers (Set-Cookie, Authorization, etc)
func (h *headerControlRoundTripper) sanitizeResponseHeaders(resp *http.Response) {
	net_http_base.SanitizeHeaders(
		&responseHeaderTarget{resp: resp},
		h.logger,
		h.responseDeniedHeaders,
		nil, // No whitelist for upstream responses (simple security baseline)
		net_http_base.InboundResponse,
	)
}
