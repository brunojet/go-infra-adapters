package net_http_server

import (
	"fmt"
	"net/http"
)

// CrossOriginConfig holds CSRF protection configuration
type CrossOriginConfig struct {
	trustedOrigins         map[string]bool // set semantics
	insecureBypassPatterns []string        // regex patterns that bypass CSRF check
	denyHandler            http.Handler    // custom handler for rejected requests
}

// CrossOriginOption configures CSRF protection. Options panic on invalid input.
type CrossOriginOption func(cfg *CrossOriginConfig)

// NewCrossOriginConfig creates a new CSRF protection configuration
// Uses DefaultCrossOriginDenyHandler for rejections if not overridden.
func NewCrossOriginConfig(opts ...CrossOriginOption) *CrossOriginConfig {
	cfg := &CrossOriginConfig{
		trustedOrigins:         make(map[string]bool),
		insecureBypassPatterns: make([]string, 0),
		denyHandler:            DefaultCrossOriginDenyHandler(),
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(cfg)
	}

	return cfg
}

// WithTrustedOrigins adds trusted origins for CSRF protection.
// Origins are validated by http.CrossOriginProtection.AddTrustedOrigin.
// Duplicates are ignored (set semantics).
func WithTrustedOrigins(origins ...string) CrossOriginOption {
	return func(cfg *CrossOriginConfig) {
		if len(origins) == 0 {
			panic("cross origin: at least one origin must be provided")
		}

		for _, origin := range origins {
			// Store origin - validation happens when AddTrustedOrigin is called
			cfg.trustedOrigins[origin] = true
		}
	}
}

// WithInsecureBypassPatterns adds regex patterns that bypass CSRF protection.
// Requests matching the patterns will NOT be checked for cross-origin.
// Use carefully - only for paths that truly need cross-origin requests.
// Examples: webhooks, public APIs, asset uploads
func WithInsecureBypassPatterns(patterns ...string) CrossOriginOption {
	return func(cfg *CrossOriginConfig) {
		if len(patterns) == 0 {
			panic("cross origin: at least one bypass pattern must be provided")
		}
		for i, pattern := range patterns {
			if pattern == "" {
				panic(fmt.Sprintf("cross origin: bypass pattern cannot be empty at index %d", i))
			}
			cfg.insecureBypassPatterns = append(cfg.insecureBypassPatterns, pattern)
		}
	}
}

// WithDenyHandler sets custom handler for rejected cross-origin requests.
// If not set, uses default problem details response (RFC 7807).
// Handler receives the rejected request and should send appropriate response.
func WithDenyHandler(h http.Handler) CrossOriginOption {
	return func(cfg *CrossOriginConfig) {
		if h == nil {
			panic("cross origin: deny handler cannot be nil")
		}
		cfg.denyHandler = h
	}
}

// BuildCrossOriginProtection creates and configures http.CrossOriginProtection
// with all origins, bypass patterns, and deny handler from this config.
// Uses DefaultCrossOriginDenyHandler (RFC 7807 problem details) if no custom handler provided.
func (c *CrossOriginConfig) BuildCrossOriginProtection() *http.CrossOriginProtection {
	cop := http.NewCrossOriginProtection()

	// Add trusted origins
	for origin := range c.trustedOrigins {
		if err := cop.AddTrustedOrigin(origin); err != nil {
			panic("cross origin: failed to add trusted origin: " + err.Error())
		}
	}

	// Add insecure bypass patterns
	for _, pattern := range c.insecureBypassPatterns {
		cop.AddInsecureBypassPattern(pattern)
	}

	// Set deny handler (always has a default, overridable via WithDenyHandler)
	cop.SetDenyHandler(c.denyHandler)

	return cop
}
