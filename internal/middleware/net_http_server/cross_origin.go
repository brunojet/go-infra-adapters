package net_http_server

import (
	"net/http"
)

// NewCrossOriginMiddleware creates CORS/CSRF protection middleware
// using Go's native http.CrossOriginProtection with configuration.
func NewCrossOriginMiddleware(opts ...CrossOriginOption) func(http.Handler) http.Handler {
	cfg := NewCrossOriginConfig(opts...)
	cop := cfg.BuildCrossOriginProtection()

	// Return middleware function that wraps handlers
	return func(next http.Handler) http.Handler {
		return cop.Handler(next)
	}
}
