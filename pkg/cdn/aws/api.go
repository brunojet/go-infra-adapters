// Package aws provides a thin facade over the internal AWS CDN adapter
// so consumers can configure distributions without importing internal packages.
package aws

import (
	"log/slog"

	cdn "github.com/brunojet/go-infra-adapters/v3/internal/cdn/aws"
	"github.com/brunojet/go-infra-adapters/v3/pkg/cdn/contracts"
	"github.com/brunojet/go-infra-adapters/v3/pkg/retry"
)

// Option configures a CdnAPI instance. Use With* functions to create options.
type Option = cdn.CdnOption

// ClientAPI is the interface for a custom AWS CloudFront client,
// useful for testing or local mocks.
type ClientAPI = cdn.CloudFrontClient

// WithClient configures a custom CloudFront client implementation.
func WithClient(client ClientAPI) Option { return cdn.WithClient(client) }

// WithMaxKeys configures the maximum number of public keys kept in a key group.
// Complexity: O(1). Memory: ~16 bytes.
func WithMaxKeys(n int) Option {
	return cdn.WithMaxKeys(n)
}

// WithConcurrency configures the maximum number of concurrent CloudFront API calls.
// Complexity: O(1). Memory: ~16 bytes.
func WithConcurrency(n int) Option {
	return cdn.WithConcurrency(n)
}

// WithLogger sets the structured logger for CloudFront distribution operations.
func WithLogger(logger *slog.Logger) Option {
	return cdn.WithLogger(logger)
}

// WithRetryStrategy sets the retry strategy for CloudFront API calls.
// Complexity: O(1). Memory: ~8 bytes.
func WithRetryStrategy(strategy retry.Strategy) Option {
	return cdn.WithRetryStrategy(strategy)
}

// NewCdn constructs a CdnAPI backed by AWS CloudFront.
// Complexity: O(1) for construction. Memory: ~2-5 KB for client + state.
func NewCdn(opts ...Option) contracts.CdnAdapter {
	return cdn.NewCdn(opts...)
}
