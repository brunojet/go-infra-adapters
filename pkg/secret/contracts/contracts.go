// Package contracts defines the public secret contracts used by callers to
// obtain and interact with provider-specific secret adapters.
package contracts

import "context"

// SecretAdapter represents an adapter instance bound to a single secret name.
// This keeps the surface small: callers create a per-secret adapter via
// secret.NewSecret[T](factory, name) and then call GetCurrent/GetVersion without
// passing the name repeatedly.
type SecretAdapter[T any] interface {
	// Name returns the secret name this adapter is bound to.
	Name() string

	// GetCurrent returns the current/active version of the secret.
	GetCurrent(ctx context.Context) (*T, error)

	// GetVersion returns a specific version by provider-specific id.
	GetVersion(ctx context.Context, version string) (*T, error)

	// SetVersion writes a new version and moves AWSPENDING to it.
	// version is used as ClientRequestToken; pass "" to let AWS generate one.
	// Returns the VersionId assigned by the provider.
	SetVersion(ctx context.Context, payload *T, version string) (string, error)

	// PromoteVersion promotes an existing version to current without modifying it.
	PromoteVersion(ctx context.Context, version string) error

	// DiscardVersion removes a specific version by provider-specific id.
	DiscardVersion(ctx context.Context, version string) error

	// HealthCheck verifies that the secret exists and credentials are valid.
	// Uses DescribeSecret which is lightweight and safe for initialization checks.
	HealthCheck(ctx context.Context) error
}
