// Package chain provides a composable multi-layer cache orchestrator (internal).
// ChainedCache coordinates Get, Set, and GetOrSet across multiple Cache[T]
// layers (local, Valkey, origin) with transparent warmup and deduplication.
package chain

import (
	"context"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/brunojet/go-infra-adapters/v4/internal/logger"
	"github.com/brunojet/go-infra-adapters/v4/pkg/cache/contracts"
	pkglogger "github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

// Layer represents one cache tier in the chain.
type Layer[T any] struct {
	Name  string              // "local", "valkey", "origin" (for logging)
	Cache contracts.Cache[T]  // The actual cache storage
	TTL   time.Duration       // Default TTL for this layer (0 = no expiry)
}

// ChainedCacheOption[T] configures a ChainedCache. Use With* functions to create options.
type ChainedCacheOption[T any] func(*chainBuilder[T])

type chainBuilder[T any] struct {
	layers []*Layer[T]
	logger pkglogger.Logger
}

// WithLayers adds one or more cache layers to the chain (order matters: fastest first).
// Each Layer specifies its own TTL; 0 means no expiry. Can be called multiple times
// to add layers incrementally.
func WithLayers[T any](layers ...Layer[T]) ChainedCacheOption[T] {
	return func(cb *chainBuilder[T]) {
		for i := range layers {
			cb.layers = append(cb.layers, &layers[i])
		}
	}
}

// WithLogger configures a structured logger. Panics if logger is nil.
func WithLogger[T any](l pkglogger.Logger) ChainedCacheOption[T] {
	if l == nil {
		panic("logger cannot be nil, use no option to default to noop logger")
	}
	return func(cb *chainBuilder[T]) {
		cb.logger = l
	}
}

// ChainedCache orchestrates multiple cache layers as a transparent stack.
// Get/Set/Delete/Exists pass through to all layers. GetOrSet implements
// cache-aside with concurrent deduplication and automatic layer population.
type ChainedCache[T any] struct {
	layers []*Layer[T]
	group  singleflight.Group  // dedupes concurrent GetOrSet on same key
	logger pkglogger.Logger
}

// NewChainedCache constructs a chain from layers and options.
// Panics if no layers were added via WithLayers options.
// Order matters: fastest layer first (local before Valkey before origin).
//
// Example:
//
//	cache := NewChainedCache[User](
//	    WithLayers(
//	        Layer[User]{Cache: local, TTL: 5*time.Minute},
//	        Layer[User]{Cache: valkey, TTL: 1*time.Hour},
//	    ),
//	    WithLogger(logger),
//	)
func NewChainedCache[T any](opts ...ChainedCacheOption[T]) *ChainedCache[T] {
	cb := &chainBuilder[T]{layers: make([]*Layer[T], 0), logger: logger.Default()}
	for _, opt := range opts {
		if opt != nil {
			opt(cb)
		}
	}
	if len(cb.layers) == 0 {
		panic("ChainedCache requires at least one layer via WithLayers option")
	}

	return &ChainedCache[T]{
		layers: cb.layers,
		logger: cb.logger,
	}
}

// Get returns the value, trying layers in order until a hit. A hit in layer N
// triggers async population of layers 0..N-1 (warmup).
func (cc *ChainedCache[T]) Get(ctx context.Context, key string) (*T, bool, error) {
	for i, layer := range cc.layers {
		val, hit, err := layer.Cache.Get(ctx, key)
		if err != nil {
			cc.logger.Warn(ctx, "cache layer get failed",
				pkglogger.String("layer", layer.Name),
				pkglogger.Error("err", err))
			continue  // try next layer
		}
		if hit {
			// Hit in layer i: warm up layers 0..i-1 asynchronously
			cc.warmupPriorsAt(ctx, key, val, i)
			return val, true, nil
		}
	}
	return nil, false, nil
}

// Set writes to all layers synchronously (best-effort). Each layer uses its own
// configured TTL, falling back to the provided ttl. Errors are logged but not
// propagated (cache is an optimization, not critical path).
func (cc *ChainedCache[T]) Set(ctx context.Context, key string, val *T, ttl time.Duration) error {
	for _, layer := range cc.layers {
		layerTTL := resolveTTL(layer.TTL, ttl)
		if err := layer.Cache.Set(ctx, key, val, layerTTL); err != nil {
			cc.logger.Warn(ctx, "cache layer set failed",
				pkglogger.String("layer", layer.Name),
				pkglogger.Error("err", err))
		}
	}
	return nil
}

// Delete removes key from all layers (best-effort, async).
func (cc *ChainedCache[T]) Delete(ctx context.Context, key string) error {
	for _, layer := range cc.layers {
		cc.deleteAsync(ctx, layer, key)
	}
	return nil
}

// Exists checks whether key exists in any layer (without triggering warmup).
func (cc *ChainedCache[T]) Exists(ctx context.Context, key string) (bool, error) {
	for _, layer := range cc.layers {
		exists, err := layer.Cache.Exists(ctx, key)
		if err != nil {
			cc.logger.Warn(ctx, "cache layer exists check failed",
				pkglogger.String("layer", layer.Name),
				pkglogger.Error("err", err))
			continue
		}
		if exists {
			return true, nil
		}
	}
	return false, nil
}

// GetOrSet returns the cached value across all layers, or invokes load on a
// total miss, caches the result in all layers, and returns it. Concurrent
// GetOrSet calls on the same key are deduplicated: only one load runs,
// others block and reuse the result.
func (cc *ChainedCache[T]) GetOrSet(
	ctx context.Context,
	key string,
	ttl time.Duration,
	load contracts.Loader[T],
) (*T, error) {
	// Fast path: try all layers without singleflight (no dedup needed on hit)
	if val, hit, _ := cc.Get(ctx, key); hit {
		return val, nil
	}

	// Miss: dedup via singleflight. One caller (leader) runs loadAndStore,
	// others block and reuse the result.
	v, err, _ := cc.group.Do(key, func() (any, error) {
		return cc.loadAndStore(ctx, key, ttl, load)
	})
	if err != nil {
		return nil, err
	}
	return v.(*T), nil
}

// loadAndStore runs under the singleflight guard for a single key.
// Double-check: re-verify cache (a prior leader may have populated it while
// this caller was queued). Then load origin and populate all layers.
func (cc *ChainedCache[T]) loadAndStore(
	ctx context.Context,
	key string,
	ttl time.Duration,
	load contracts.Loader[T],
) (any, error) {
	// Double-check: another leader may have finished and populated while we waited
	if val, hit, _ := cc.Get(ctx, key); hit {
		return val, nil
	}

	// Load origin (only once per batch of concurrent requests)
	val, err := load(ctx)
	if err != nil {
		return nil, err  // origin error: propagate, don't cache
	}

	// Populate all layers with the loaded value (async, best-effort)
	cc.populateLayers(ctx, key, val, ttl)
	return val, nil
}

// HealthCheck returns the first error from any layer, or nil if all succeed.
func (cc *ChainedCache[T]) HealthCheck(ctx context.Context) error {
	for _, layer := range cc.layers {
		if err := layer.Cache.HealthCheck(ctx); err != nil {
			return err
		}
	}
	return nil
}

// warmupPriorsAt populates layers 0..beforeLayer-1 with val (triggered by a hit in beforeLayer).
func (cc *ChainedCache[T]) warmupPriorsAt(ctx context.Context, key string, val *T, beforeLayer int) {
	for i := 0; i < beforeLayer; i++ {
		cc.setAsync(ctx, cc.layers[i], key, val, cc.layers[i].TTL)
	}
}

// populateLayers writes val to all layers asynchronously.
func (cc *ChainedCache[T]) populateLayers(
	ctx context.Context,
	key string,
	val *T,
	defaultTTL time.Duration,
) {
	for _, layer := range cc.layers {
		cc.setAsync(ctx, layer, key, val, resolveTTL(layer.TTL, defaultTTL))
	}
}

// setAsync writes to a single layer asynchronously. Errors are logged but not propagated.
func (cc *ChainedCache[T]) setAsync(ctx context.Context, layer *Layer[T], key string, val *T, ttl time.Duration) {
	go func() {
		if err := layer.Cache.Set(ctx, key, val, ttl); err != nil {
			cc.logger.Warn(ctx, "cache layer set failed",
				pkglogger.String("layer", layer.Name),
				pkglogger.Error("err", err))
		}
	}()
}

// deleteAsync removes from a single layer asynchronously. Errors are logged.
func (cc *ChainedCache[T]) deleteAsync(ctx context.Context, layer *Layer[T], key string) {
	go func() {
		if err := layer.Cache.Delete(ctx, key); err != nil {
			cc.logger.Warn(ctx, "cache layer delete failed",
				pkglogger.String("layer", layer.Name),
				pkglogger.Error("err", err))
		}
	}()
}

// resolveTTL returns layer TTL if set, otherwise falls back to default.
func resolveTTL(layerTTL, defaultTTL time.Duration) time.Duration {
	if layerTTL == 0 {
		return defaultTTL
	}
	return layerTTL
}

// Helper to convert []Layer to []*Layer
func sliceLayers[T any](layers []Layer[T]) []*Layer[T] {
	result := make([]*Layer[T], len(layers))
	for i := range layers {
		result[i] = &layers[i]
	}
	return result
}
