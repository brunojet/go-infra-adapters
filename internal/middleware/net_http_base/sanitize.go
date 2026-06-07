package net_http_base

import (
	"context"
	"net/http"

	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

// Log event messages and field keys for header sanitization
const (
	logMsgHeaderRemoved = "header removed"
	logReasonDenied     = "in hardcoded deny list"
	logReasonNotAllowed = "not in allowed list"
	logKeyEvent         = "event"
	logKeyHeader        = "header"
	logKeyReason        = "reason"
)

// HeaderSanitizationTarget represents any collection of headers to sanitize (request or response)
type HeaderSanitizationTarget interface {
	Headers() http.Header
	SanitizationContext() context.Context
}

// SanitizeHeaders removes denied headers and optionally enforces whitelist for InboundRequest.
// Behavior by event type:
//   - InboundRequest: removes denied headers AND non-whitelisted headers (server controls input)
//   - InboundResponse: removes only denied headers (simple security baseline)
//   - OutboundRequest: removes only denied headers (dev controls what to send)
//
// Performance optimization: Iterates over smaller set first (deniedHeaders is typically 15-20 items).
// http.Header keys are already canonical form when iterating, no need to normalize.
func SanitizeHeaders(
	target HeaderSanitizationTarget,
	log logger.Logger,
	deniedHeaders, allowedHeaders map[string]bool,
	eventType EventType,
) {
	ctx := target.SanitizationContext()
	headers := target.Headers()

	// Pass 1: Remove denied headers by iterating over the smaller set (15-20 items vs 50+ headers)
	// http.Header keys are already in canonical form, no CanonicalHeaderKey() needed
	for denied := range deniedHeaders {
		if _, exists := headers[denied]; exists {
			headers.Del(denied)
			log.Debug(ctx, logMsgHeaderRemoved,
				logger.String(logKeyEvent, eventType.String()),
				logger.String(logKeyHeader, denied),
				logger.String(logKeyReason, logReasonDenied),
			)
		}
	}

	if eventType != InboundRequest {
		return
	}

	// Pass 2: If whitelist provided, remove any headers not in allowed list
	removeAllHeaders := len(allowedHeaders) == 0
	for headerName := range headers {
		if removeAllHeaders || !allowedHeaders[headerName] {
			headers.Del(headerName)
			log.Debug(ctx, logMsgHeaderRemoved,
				logger.String(logKeyEvent, eventType.String()),
				logger.String(logKeyHeader, headerName),
				logger.String(logKeyReason, logReasonNotAllowed),
			)
		}
	}
}
