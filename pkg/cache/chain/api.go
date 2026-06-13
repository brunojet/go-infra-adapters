// Package chain provides a thin facade over the internal ChainedCache orchestrator
// so consumers can construct multi-layer caches without importing internal packages directly.
package chain

import (
	internalchain "github.com/brunojet/go-infra-adapters/v4/internal/cache/chain"
	"github.com/brunojet/go-infra-adapters/v4/pkg/cache/contracts"
	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

// Layer represents one cache tier in the chain (exported from internal).
type Layer[T any] = internalchain.Layer[T]

// Option configures a ChainedCache. Use With* functions to create options.
type Option = internalchain.ChainedCacheOption

// WithLogger configures a structured logger. Panics if logger is nil.
// Omit this option to use the noop default logger.
func WithLogger(l logger.Logger) Option { return internalchain.WithLogger(l) }

// NewChainedCache constructs a chain from layers (order matters: fastest first).
// Panics if layers is empty or if any With* option panics.
//
// Example:
//
//	chain := chain.NewChainedCache(
//		[]chain.Layer[User]{
//			{Name: "local", Cache: localCache, TTL: 5*time.Minute},
//			{Name: "valkey", Cache: valkeyCache, TTL: 1*time.Hour},
//		},
//		chain.WithLogger(logger),
//	)
//	user, err := chain.GetOrSet(ctx, "user:123", 5*time.Minute, loadFromDB)
func NewChainedCache[T any](layers []Layer[T], opts ...Option) contracts.Cache[T] {
	return internalchain.NewChainedCache[T](layers, opts...)
}
