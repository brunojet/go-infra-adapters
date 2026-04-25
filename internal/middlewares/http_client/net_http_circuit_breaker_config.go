package middlewares

import (
	"fmt"
	"log"
	"time"
)

// circuitBreakerConfig configures the circuit breaker used by the
// Breaker middleware.
type circuitBreakerConfig struct {
	MaxFailures      int           // consecutive failures before opening
	ResetTimeout     time.Duration // how long to wait before transitioning to half-open
	HalfOpenRequests int           // probe requests allowed in half-open state
}

// BreakerOption configures a CircuitBreakerConfig.
// BreakerOption configures a CircuitBreakerConfig and may return an error
// when the provided value is invalid.
type BreakerOption func(cfg *circuitBreakerConfig) error

// newCircuitBreakerConfig builds a CircuitBreakerConfig applying provided
// functional options.
func newCircuitBreakerConfig(opts ...BreakerOption) circuitBreakerConfig {
	cfg := circuitBreakerConfig{}
	for _, o := range opts {
		if o == nil {
			continue
		}
		if err := o(&cfg); err != nil {
			log.Panicf("invalid circuit breaker configuration: %v", err)
		}
	}
	return cfg
}

// WithCircuitBreakerMaxFailures sets the number of consecutive failures
// required to open the circuit.
func WithCircuitBreakerMaxFailures(n int) BreakerOption {
	return func(c *circuitBreakerConfig) error {
		if n <= 0 {
			return fmt.Errorf("max failures must be > 0")
		}
		c.MaxFailures = n
		return nil
	}
}

// WithCircuitBreakerResetTimeout sets the reset timeout for the breaker.
func WithCircuitBreakerResetTimeout(d time.Duration) BreakerOption {
	return func(c *circuitBreakerConfig) error {
		if d <= 0 {
			return fmt.Errorf("reset timeout must be > 0")
		}
		c.ResetTimeout = d
		return nil
	}
}

// WithCircuitBreakerHalfOpenRequests sets how many probe requests are
// allowed while the breaker is half-open.
func WithCircuitBreakerHalfOpenRequests(n int) BreakerOption {
	return func(c *circuitBreakerConfig) error {
		if n < 0 {
			return fmt.Errorf("half-open requests cannot be negative")
		}
		c.HalfOpenRequests = n
		return nil
	}
}
