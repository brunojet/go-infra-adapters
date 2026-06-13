// Package contracts defines the public cache contracts used by callers to
// obtain and interact with provider-specific cache adapters (in-memory, Valkey, ...).
package contracts

import (
	"context"
	"time"
)

// Loader produces the value for a key on cache miss. Used by Cache.GetOrSet.
type Loader[T any] func(ctx context.Context) (*T, error)

// Cache is a type-safe key/value cache bound to a value type T.
// Implementations may be single-process (in-memory) or distributed (Valkey).
// T must be JSON-serialisable for distributed adapters that marshal at the boundary.
type Cache[T any] interface {
	// Get returns the cached value for key. The bool reports a hit: false means
	// the key is absent or expired, with a nil value and nil error. A non-nil
	// error signals a backend failure, not a miss.
	Get(ctx context.Context, key string) (*T, bool, error)

	// Set stores val under key with the given ttl. A ttl of 0 means no expiry.
	Set(ctx context.Context, key string, val *T, ttl time.Duration) error

	// Delete removes key. Deleting an absent key is not an error.
	Delete(ctx context.Context, key string) error

	// Exists reports whether key is present and unexpired.
	Exists(ctx context.Context, key string) (bool, error)

	// GetOrSet returns the cached value for key, or invokes load on a miss,
	// stores the result under key with ttl, and returns it. The load callback
	// runs only on a miss; its error is propagated and nothing is cached.
	GetOrSet(ctx context.Context, key string, ttl time.Duration, load Loader[T]) (*T, error)

	// HealthCheck verifies the backend is reachable. Safe for initialization checks.
	HealthCheck(ctx context.Context) error
}
