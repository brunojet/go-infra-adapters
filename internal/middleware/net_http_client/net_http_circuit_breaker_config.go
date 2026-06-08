package net_http_client

import (
	"time"
)

// circuitBreakerConfig configures the circuit breaker used by the
// Breaker middleware.
type circuitBreakerConfig struct {
	MaxFailures       int               // consecutive failures before opening
	ResetTimeout      time.Duration     // how long to wait before transitioning to half-open
	HalfOpenRequests  int               // probe requests allowed in half-open state
	FailureClassifier FailureClassifier // error classification for circuit breaker increments
}

// BreakerOption configures a CircuitBreakerConfig.
// Options now panic on invalid input instead of returning errors.
type BreakerOption func(cfg *circuitBreakerConfig)

// newCircuitBreakerConfig builds a CircuitBreakerConfig applying provided
// functional options.
//
// Defaults (conservative, suitable for production HTTP clients):
// - MaxFailures: 5 consecutive failures before opening
// - ResetTimeout: 30s before transitioning to half-open
// - HalfOpenRequests: 1 probe request allowed in half-open state
// - FailureClassifier: DefaultFailureClassifier (5xx, 429, network errors)
func newCircuitBreakerConfig(opts ...BreakerOption) circuitBreakerConfig {
	cfg := circuitBreakerConfig{
		MaxFailures:      5,
		ResetTimeout:     30 * time.Second,
		HalfOpenRequests: 1,
		// FailureClassifier is nil initially, will default to DefaultFailureClassifier in NewBreakerMiddleware
	}
	for _, o := range opts {
		if o == nil {
			continue
		}
		o(&cfg)
	}
	return cfg
}

// WithCircuitBreakerMaxFailures sets the number of consecutive failures
// required to open the circuit.
func WithCircuitBreakerMaxFailures(n int) BreakerOption {
	return func(c *circuitBreakerConfig) {
		if n <= 0 {
			panic("max failures must be > 0")
		}
		c.MaxFailures = n
	}
}

// WithCircuitBreakerResetTimeout sets the reset timeout for the breaker.
func WithCircuitBreakerResetTimeout(d time.Duration) BreakerOption {
	return func(c *circuitBreakerConfig) {
		if d <= 0 {
			panic("reset timeout must be > 0")
		}
		c.ResetTimeout = d
	}
}

// WithCircuitBreakerHalfOpenRequests sets how many probe requests are
// allowed while the breaker is half-open.
func WithCircuitBreakerHalfOpenRequests(n int) BreakerOption {
	return func(c *circuitBreakerConfig) {
		if n < 0 {
			panic("half-open requests cannot be negative")
		}
		c.HalfOpenRequests = n
	}
}

// WithFailureClassifier sets a custom error classifier for the circuit breaker.
func WithFailureClassifier(classifier FailureClassifier) BreakerOption {
	return func(c *circuitBreakerConfig) {
		if classifier != nil {
			c.FailureClassifier = classifier
		}
	}
}
