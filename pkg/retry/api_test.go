package retry

import (
	"errors"
	"testing"
	"time"
)

func TestNewStandard_Interface(t *testing.T) {
	strategy := NewStandard()
	if strategy == nil {
		t.Fatal("expected non-nil strategy from NewStandard()")
	}

	// Verify it implements Strategy interface
	if strategy.MaxAttempts() != 2 {
		t.Fatalf("expected 2 attempts, got %d", strategy.MaxAttempts())
	}

	backoff := strategy.BackoffFor(1)
	if backoff != 500*time.Millisecond {
		t.Fatalf("expected 500ms backoff, got %v", backoff)
	}

	if !strategy.IsRetryable(errors.New("StatusCode: 500")) {
		t.Fatal("expected 5xx error to be retryable")
	}
}

func TestNewNoRetry_Interface(t *testing.T) {
	strategy := NewNoRetry()
	if strategy == nil {
		t.Fatal("expected non-nil strategy from NewNoRetry()")
	}

	// Verify it implements Strategy interface
	if strategy.MaxAttempts() != 1 {
		t.Fatalf("expected 1 attempt for NoRetry, got %d", strategy.MaxAttempts())
	}

	backoff := strategy.BackoffFor(1)
	if backoff != 0 {
		t.Fatalf("expected 0 backoff for NoRetry, got %v", backoff)
	}

	if strategy.IsRetryable(errors.New("any error")) {
		t.Fatal("expected NoRetry to not retry any error")
	}
}
