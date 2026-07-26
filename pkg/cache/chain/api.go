// Package chain provides a thin facade over the internal ChainedCache orchestrator
// so consumers can construct multi-layer caches without importing internal packages directly.
package chain

import (
	"context"
	"time"

	internalchain "github.com/brunojet/go-infra-adapters/v4/internal/cache/chain"
	"github.com/brunojet/go-infra-adapters/v4/pkg/cache/contracts"
	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

// Layer represents one cache tier in the chain (exported from internal).
type Layer[T any] = internalchain.Layer[T]

// Option configures a ChainedCache. Use With* functions to create options.
type Option[T any] = internalchain.ChainedCacheOption[T]

// ChainedCache is the contract satisfied by NewChainedCache: the base
// contracts.Cache[T] storage operations plus GetOrSet, the cache-aside
// method with singleflight dedup that only the chain orchestrator provides.
type ChainedCache[T any] interface {
	contracts.Cache[T]
	GetOrSet(ctx context.Context, key string, ttl time.Duration, load contracts.Loader[T]) (*T, error)
}

// WithLayers adds one or more cache layers to the chain (order matters: fastest first).
// Each Layer specifies its own TTL; 0 means no expiry. Can be called multiple times
// to add layers incrementally.
func WithLayers[T any](layers ...Layer[T]) Option[T] { return internalchain.WithLayers(layers...) }

// WithLogger configures a structured logger. Panics if logger is nil.
// Omit this option to use the noop default logger.
func WithLogger[T any](l logger.Logger) Option[T] { return internalchain.WithLogger[T](l) }

// NewChainedCache constructs a chain from layers and options.
// Panics if no layers were added via WithLayers options.
// Order matters: fastest layer first (local before Valkey before origin).
//
// Example:
//
//	cache := chain.NewChainedCache[User](
//	    chain.WithLayers(
//	        chain.Layer[User]{Cache: localCache, TTL: 5*time.Minute},
//	        chain.Layer[User]{Cache: valkeyCache, TTL: 1*time.Hour},
//	    ),
//	    chain.WithLogger[User](logger),
//	)
//	user, err := cache.GetOrSet(ctx, "user:123", 5*time.Minute, loadFromDB)
func NewChainedCache[T any](opts ...Option[T]) ChainedCache[T] {
	return internalchain.NewChainedCache[T](opts...)
}
