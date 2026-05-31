// Package aws provides a thin facade over the internal AWS secret adapter
// so consumers can construct clients without importing internal packages directly.
package aws

import (
	"log/slog"

	internalaws "github.com/brunojet/go-infra-adapters/v3/internal/secret/aws"
	"github.com/brunojet/go-infra-adapters/v3/pkg/retry"
	"github.com/brunojet/go-infra-adapters/v3/pkg/secret/contracts"
)

// Option configures a secret client. Use With* functions to create options.
type Option = internalaws.SecretsOption

// ClientAPI is the interface for a custom AWS Secrets Manager client,
// useful for testing or local mocks.
type ClientAPI = internalaws.SecretsManagerClient

// SecretAPI is the shared Secrets Manager client. Obtain via NewSecretAPI.
type SecretAPI = internalaws.SecretAPI

// WithClient configures a custom Secrets Manager client implementation.
func WithClient(client ClientAPI) Option { return internalaws.WithClient(client) }

// WithRegion configures the AWS region.
// Complexity: O(n) where n = len(region). Memory: ~16 bytes + string.
func WithRegion(region string) Option { return internalaws.WithRegion(region) }

// WithEndpoint configures a custom endpoint (useful for local testing).
// Complexity: O(n) where n = len(endpoint). Memory: ~16 bytes + string.
func WithEndpoint(endpoint string) Option { return internalaws.WithEndpoint(endpoint) }

// WithLogger configures a structured logger. A nil logger falls back to slog.Default().
func WithLogger(logger *slog.Logger) Option { return internalaws.WithLogger(logger) }

// WithRetryStrategy sets the retry strategy for Secrets Manager API calls.
func WithRetryStrategy(strategy retry.Strategy) Option {
	return internalaws.WithRetryStrategy(strategy)
}

// NewSecretAPI constructs a SecretAPI backed by AWS Secrets Manager.
// Returns an error if no client is injected and the SDK cannot initialize.
// Complexity: O(1). Memory: ~1-2 KB for client + internal state.
func NewSecretAPI(opts ...Option) (*SecretAPI, error) {
	return internalaws.NewSecretAPI(opts...)
}

// NewSecrets creates a type-safe SecretAdapter[T] for the named secret,
// reusing the client held by api. T must be JSON-serialisable.
// The returned adapter supports HealthCheck (via DescribeSecret) for initialization validation.
// Complexity: O(n) where n = len(name). Memory: ~200 bytes for adapter + string reference.
func NewSecrets[T any](api *SecretAPI, name string) contracts.SecretAdapter[T] {
	return internalaws.NewSecrets[T](api, name)
}
