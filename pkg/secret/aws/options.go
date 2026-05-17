// Package aws provides a thin facade over the internal AWS secret adapter
// options so consumers can construct clients without importing internal
// packages directly.
package aws

import (
	"github.com/brunojet/go-infra-adapters/internal/secret/aws"
	"github.com/brunojet/go-infra-adapters/pkg/secret/contracts"
)

// WithRegion configures the AWS region for the secret client.
func WithRegion(region string) aws.Option { return aws.WithRegion(region) }

// WithEndpoint configures a custom endpoint for the secret client (useful
// for local testing/mocks).
func WithEndpoint(endpoint string) aws.Option { return aws.WithEndpoint(endpoint) }

// NewSecretClient constructs a SecretClientAPI using the provided options.
func NewSecretClient(opts ...aws.Option) (contracts.SecretClientAPI, error) {
	return aws.NewSecretClient(opts...)
}
