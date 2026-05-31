package aws

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/brunojet/go-infra-adapters/v3/pkg/retry"
)

func TestSDKRetryer(t *testing.T) {
	logger := slog.Default()
	strategy := retry.NewStandard()
	retryer := NewSDKRetryer(strategy, logger)

	t.Run("IsErrorRetryable_5xx", func(t *testing.T) {
		err := errors.New("StatusCode: 500")
		if !retryer.IsErrorRetryable(err) {
			t.Fatal("expected 5xx error to be retryable")
		}
	})

	t.Run("IsErrorRetryable_429", func(t *testing.T) {
		err := errors.New("StatusCode: 429")
		if !retryer.IsErrorRetryable(err) {
			t.Fatal("expected 429 error to be retryable")
		}
	})

	t.Run("IsErrorRetryable_409", func(t *testing.T) {
		err := errors.New("StatusCode: 409")
		if !retryer.IsErrorRetryable(err) {
			t.Fatal("expected 409 error to be retryable")
		}
	})

	t.Run("IsErrorRetryable_ConnectionRefused", func(t *testing.T) {
		err := errors.New("connection refused")
		if !retryer.IsErrorRetryable(err) {
			t.Fatal("expected connection refused error to be retryable")
		}
	})

	t.Run("IsErrorRetryable_NonRetryable", func(t *testing.T) {
		err := errors.New("validation error")
		if retryer.IsErrorRetryable(err) {
			t.Fatal("expected validation error to not be retryable")
		}
	})

	t.Run("IsErrorRetryable_Nil", func(t *testing.T) {
		if retryer.IsErrorRetryable(nil) {
			t.Fatal("expected nil error to not be retryable")
		}
	})

	t.Run("MaxAttempts", func(t *testing.T) {
		attempts := retryer.MaxAttempts()
		if attempts != 3 {
			t.Fatalf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("RetryDelay", func(t *testing.T) {
		delay, err := retryer.RetryDelay(1, errors.New("test"))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if delay != 500*time.Millisecond {
			t.Fatalf("expected 500ms, got %v", delay)
		}
	})

	t.Run("AttemptTimeout", func(t *testing.T) {
		ctx := context.Background()
		timeout, ctxOut, err := retryer.AttemptTimeout(ctx, 1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if timeout != 30*time.Second {
			t.Fatalf("expected 30s timeout, got %v", timeout)
		}
		if ctxOut == nil {
			t.Fatal("expected non-nil context")
		}
	})

	t.Run("GetInitialToken", func(t *testing.T) {
		token := retryer.GetInitialToken()
		if token == nil {
			t.Fatal("expected non-nil token function")
		}
		// Token function should accept an error and return nil
		if token(errors.New("test")) != nil {
			t.Fatal("expected token to return nil")
		}
	})

	t.Run("FreshToken", func(t *testing.T) {
		token := retryer.FreshToken(errors.New("test"))
		if token == nil {
			t.Fatal("expected non-nil token function")
		}
		if token(errors.New("test")) != nil {
			t.Fatal("expected token to return nil")
		}
	})

	t.Run("GetRetryToken", func(t *testing.T) {
		token, err := retryer.GetRetryToken(context.Background(), errors.New("test"))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if token == nil {
			t.Fatal("expected non-nil token function")
		}
		if token(errors.New("test")) != nil {
			t.Fatal("expected token to return nil")
		}
	})

	t.Run("LogRollback", func(t *testing.T) {
		// LogRollback is a no-op, just verify it doesn't panic
		retryer.LogRollback(context.Background(), nil)
	})
}

func TestSDKRetryer_WithNoRetryStrategy(t *testing.T) {
	logger := slog.Default()
	strategy := retry.NewNoRetry()
	retryer := NewSDKRetryer(strategy, logger)

	t.Run("MaxAttempts_NoRetry", func(t *testing.T) {
		attempts := retryer.MaxAttempts()
		if attempts != 1 {
			t.Fatalf("expected 1 attempt for NoRetry, got %d", attempts)
		}
	})

	t.Run("IsErrorRetryable_NoRetry", func(t *testing.T) {
		err := errors.New("StatusCode: 500")
		if retryer.IsErrorRetryable(err) {
			t.Fatal("expected NoRetry to not retry any error")
		}
	})

	t.Run("RetryDelay_NoRetry", func(t *testing.T) {
		delay, err := retryer.RetryDelay(1, errors.New("test"))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if delay != 0 {
			t.Fatalf("expected 0 delay for NoRetry, got %v", delay)
		}
	})
}
