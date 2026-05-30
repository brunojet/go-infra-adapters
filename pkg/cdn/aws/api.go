// Package aws provides a thin facade over the internal AWS CDN adapter
// so consumers can configure distributions without importing internal packages.
package aws

import (
	"log/slog"

	"github.com/brunojet/go-infra-adapters/internal/cdn/aws"
	"github.com/brunojet/go-infra-adapters/pkg/cdn/contracts"
)

// Option configures a CdnAPI instance. Use With* functions to create options.
type Option = aws.CdnOption

// ClientAPI is the interface for a custom AWS CloudFront client,
// useful for testing or local mocks.
type ClientAPI = aws.CloudFrontClient

// WithClient configures a custom CloudFront client implementation.
func WithClient(client ClientAPI) Option { return aws.WithClient(client) }

// WithMaxKeys configures the maximum number of public keys kept in a key group.
// Complexity: O(1). Memory: ~16 bytes.
func WithMaxKeys(n int) Option {
	return aws.WithMaxKeys(n)
}

// WithConcurrency configures the maximum number of concurrent CloudFront API calls.
// Complexity: O(1). Memory: ~16 bytes.
func WithConcurrency(n int) Option {
	return aws.WithConcurrency(n)
}

// WithLogger sets the structured logger for CloudFront distribution operations.
func WithLogger(logger *slog.Logger) Option {
	return aws.WithLogger(logger)
}

// NewCdn constructs a CdnAPI backed by AWS CloudFront.
// Complexity: O(1) for construction. Memory: ~2-5 KB for client + state.
func NewCdn(opts ...Option) contracts.CdnAdapter {
	return aws.NewCdn(opts...)
}
