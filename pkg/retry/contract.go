// Package retry provides retry strategy contracts and constructors.
package retry

import (
	"time"

	"github.com/brunojet/go-infra-adapters/v3/internal/retry"
)

// Strategy defines how to retry failed operations.
// Implementations must be thread-safe.
type Strategy interface {
	// IsRetryable returns true if the error should be retried.
	IsRetryable(err error) bool
	// MaxAttempts returns the maximum number of retry attempts.
	MaxAttempts() int
	// BackoffFor returns the duration to wait before the given attempt.
	BackoffFor(attempt int) time.Duration
}

// NewStandard creates a Standard retry strategy with 3 attempts.
// Suitable for AWS API calls with transient failures (5xx, 429, 409).
//
// Attempts: 3
// Backoff: exponential (500ms, 1s, 2s)
// Retryable errors: 5xx, 429, 409, timeouts, connection errors
func NewStandard() Strategy {
	return retry.NewStandard()
}

// NewNoRetry creates a NoRetry strategy that never retries.
// Useful for testing or operations that should fail fast.
func NewNoRetry() Strategy {
	return &retry.NoRetry{}
}
