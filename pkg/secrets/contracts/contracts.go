package contracts

import "context"

// SecretValue contains the secret payload. Keep small to reduce GC pressure.
type SecretValue struct {
	Data     []byte
	Metadata map[string]string
}

type SecretClientAPI any

// SecretAdapter represents an adapter instance bound to a single secret name.
// This keeps the surface small: callers create a per-secret adapter via
// SecretAPI.NewSecret(name) and then call GetCurrent/GetVersion without
// passing the name repeatedly.
type SecretAdapter interface {
	// Name returns the secret name this adapter is bound to.
	Name() string

	// GetCurrent returns the current/active version of the secret.
	GetCurrent(ctx context.Context) (*SecretValue, error)

	// GetVersion returns a specific version by provider-specific id.
	GetVersion(ctx context.Context, version string) (*SecretValue, error)
}

// SecretAPI constructs per-secret accessors. Keeping the provider surface
// minimal reduces accidental broad permissions and keeps implementations
// focused on read access patterns.
type SecretAPI interface {
	NewSecret(name string) (SecretAdapter, error)
	NewSecretWithClient(name string, client SecretClientAPI) (SecretAdapter, error)
}
