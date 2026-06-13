// Package chain provides a composable multi-layer cache orchestrator.
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

// ChainedCache orchestrates multiple cache layers as a transparent stack.
// Get/Set/Delete/Exists pass through to all layers. GetOrSet implements
// cache-aside with concurrent deduplication and automatic layer population.
type ChainedCache[T any] struct {
	layers []*Layer[T]
	group  singleflight.Group  // dedupes concurrent GetOrSet on same key
	logger pkglogger.Logger
}

// Config configures a ChainedCache.
type Config struct {
	Logger pkglogger.Logger
}

// NewChainedCache constructs a chain from layers (order matters: fastest first).
// Panics if layers is empty.
func NewChainedCache[T any](cfg Config, layers ...Layer[T]) *ChainedCache[T] {
	if len(layers) == 0 {
		panic("ChainedCache requires at least one layer")
	}
	if cfg.Logger == nil {
		cfg.Logger = logger.Default()
	}
	return &ChainedCache[T]{
		layers: sliceLayers(layers),
		logger: cfg.Logger,
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

// Set writes to all layers asynchronously (best-effort). Errors are logged
// but not propagated (cache is an optimization, not critical path).
func (cc *ChainedCache[T]) Set(ctx context.Context, key string, val *T, ttl time.Duration) error {
	for _, layer := range cc.layers {
		layer := layer
		go func() {
			ttlForLayer := layer.TTL
			if ttlForLayer == 0 {
				ttlForLayer = ttl
			}
			if err := layer.Cache.Set(ctx, key, val, ttlForLayer); err != nil {
				cc.logger.Warn(ctx, "cache layer set failed",
					pkglogger.String("layer", layer.Name),
					pkglogger.Error("err", err))
			}
		}()
	}
	return nil
}

// Delete removes key from all layers (best-effort, async).
func (cc *ChainedCache[T]) Delete(ctx context.Context, key string) error {
	for _, layer := range cc.layers {
		layer := layer
		go func() {
			if err := layer.Cache.Delete(ctx, key); err != nil {
				cc.logger.Warn(ctx, "cache layer delete failed",
					pkglogger.String("layer", layer.Name),
					pkglogger.Error("err", err))
			}
		}()
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
		i := i
		go func() {
			layer := cc.layers[i]
			if err := layer.Cache.Set(ctx, key, val, layer.TTL); err != nil {
				cc.logger.Warn(ctx, "cache warmup failed",
					pkglogger.String("layer", layer.Name),
					pkglogger.Error("err", err))
			}
		}()
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
		layer := layer
		go func() {
			ttl := layer.TTL
			if ttl == 0 {
				ttl = defaultTTL
			}
			if err := layer.Cache.Set(ctx, key, val, ttl); err != nil {
				cc.logger.Warn(ctx, "cache layer set failed during populate",
					pkglogger.String("layer", layer.Name),
					pkglogger.Error("err", err))
			}
		}()
	}
}

// Helper to convert []Layer to []*Layer
func sliceLayers[T any](layers []Layer[T]) []*Layer[T] {
	result := make([]*Layer[T], len(layers))
	for i := range layers {
		result[i] = &layers[i]
	}
	return result
}
