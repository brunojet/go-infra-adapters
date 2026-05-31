// Package retry provides retry strategies for AWS SDK clients.
package retry

import (
	"time"
)

// Strategy defines how to retry failed operations.
type Strategy interface {
	// IsRetryable returns true if the error should be retried.
	IsRetryable(err error) bool
	// MaxAttempts returns the maximum number of retry attempts.
	MaxAttempts() int
	// BackoffFor returns the duration to wait before the given attempt.
	BackoffFor(attempt int) time.Duration
}

// Standard implements Strategy with sensible defaults.
// 3 attempts with exponential backoff (500ms, 1s, 2s).
type Standard struct {
	maxAttempts int
	baseBackoff time.Duration
}

// NewStandard creates a Standard retry strategy with 3 attempts.
func NewStandard() *Standard {
	return &Standard{
		maxAttempts: 3,
		baseBackoff: 500 * time.Millisecond,
	}
}

// IsRetryable returns true for transient AWS errors (5xx, 429, connection errors).
func (s *Standard) IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	// Check for common transient AWS errors
	errStr := err.Error()
	// 5xx server errors
	if contains(errStr, "StatusCode: 5") {
		return true
	}
	// 429 throttling
	if contains(errStr, "StatusCode: 429") {
		return true
	}
	// 409 Conflict (for DeletePublicKey when key still in use)
	if contains(errStr, "StatusCode: 409") {
		return true
	}
	// Connection timeouts
	if contains(errStr, "timeout") || contains(errStr, "connection refused") {
		return true
	}
	return false
}

// MaxAttempts returns 3.
func (s *Standard) MaxAttempts() int {
	return s.maxAttempts
}

// BackoffFor returns exponential backoff: 500ms, 1s, 2s.
func (s *Standard) BackoffFor(attempt int) time.Duration {
	if attempt < 1 {
		return 0
	}
	// 2^(attempt-1) * baseBackoff
	// attempt 1: 500ms
	// attempt 2: 1s
	// attempt 3: 2s
	multiplier := 1 << uint(attempt-1) // 2^(attempt-1)
	return time.Duration(multiplier) * s.baseBackoff
}

// NoRetry implements Strategy with no retries.
type NoRetry struct{}

// IsRetryable always returns false.
func (n *NoRetry) IsRetryable(err error) bool {
	return false
}

// MaxAttempts returns 1 (no retries).
func (n *NoRetry) MaxAttempts() int {
	return 1
}

// BackoffFor returns 0.
func (n *NoRetry) BackoffFor(attempt int) time.Duration {
	return 0
}

// contains checks if a string contains a substring (case-insensitive helper).
func contains(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
