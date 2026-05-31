package aws

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/smithy-go/logging"

	"github.com/brunojet/go-infra-adapters/v3/pkg/retry"
)

// SDKRetryer adapts retry.Strategy to AWS SDK v2's retry.Retryer interface.
// Implements aws.Retryer using the configured Strategy for max attempts and backoff.
type SDKRetryer struct {
	strategy retry.Strategy
	logger   *slog.Logger
}

// NewSDKRetryer creates an AWS SDK Retryer from a retry Strategy.
func NewSDKRetryer(strategy retry.Strategy, logger *slog.Logger) *SDKRetryer {
	return &SDKRetryer{
		strategy: strategy,
		logger:   logger,
	}
}

// IsErrorRetryable checks if the error is retryable using the strategy.
func (sr *SDKRetryer) IsErrorRetryable(err error) bool {
	retryable := sr.strategy.IsRetryable(err)
	if !retryable && err != nil {
		sr.logger.Debug("error not retryable, will not retry",
			"error", err.Error())
	}
	return retryable
}

// MaxAttempts returns the maximum number of retry attempts from the strategy.
func (sr *SDKRetryer) MaxAttempts() int {
	return sr.strategy.MaxAttempts()
}

// AttemptTimeout returns a timeout duration for an attempt.
// The AWS SDK is responsible for managing the context lifecycle.
func (sr *SDKRetryer) AttemptTimeout(ctx context.Context, attempt int) (time.Duration, context.Context, error) {
	// Return a reasonable timeout per attempt (e.g., 30s)
	timeout := 30 * time.Second
	// Return original context; AWS SDK manages the timeout
	return timeout, ctx, nil
}

// RetryDelay returns the backoff duration for a retry using the strategy.
func (sr *SDKRetryer) RetryDelay(attempt int, err error) (time.Duration, error) {
	backoff := sr.strategy.BackoffFor(attempt)
	sr.logger.Debug("applying retry backoff",
		"attempt", attempt,
		"backoffDuration", backoff.String(),
		"maxAttempts", sr.strategy.MaxAttempts(),
		"error", err.Error())
	return backoff, nil
}

// GetInitialToken is required by aws.Retryer but not used in our strategy.
func (sr *SDKRetryer) GetInitialToken() func(error) error {
	return func(err error) error { return nil }
}

// FreshToken is required by aws.Retryer but not used in our strategy.
func (sr *SDKRetryer) FreshToken(lastTokenErr error) func(error) error {
	return func(err error) error { return nil }
}

// LogRollback is required by aws.Retryer but not used in our strategy.
func (sr *SDKRetryer) LogRollback(ctx context.Context, fn logging.LoggerFunc) {
	// No-op: strategy doesn't use this
}

// GetRetryToken is required by aws.Retryer.
func (sr *SDKRetryer) GetRetryToken(ctx context.Context, err error) (func(error) error, error) {
	return func(e error) error { return nil }, nil
}
