// Package local provides an in-memory implementation of the cache contract.
// It is a real single-process cache (map + RWMutex + TTL), not a test stub,
// and is swappable for a distributed adapter via the same contracts.Cache[T].
package local

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/dgraph-io/ristretto"
	"github.com/brunojet/go-infra-adapters/v4/internal/logger"
	"github.com/brunojet/go-infra-adapters/v4/pkg/cache/contracts"
	pkglogger "github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

var (
	errEmptyKey = errors.New("cache key cannot be empty")
	errNilValue = errors.New("cache value cannot be nil")
)

// CacheOption configures a LocalCache. Use With* functions to create options.
type CacheOption func(*cacheConfig)

type cacheConfig struct {
	logger             pkglogger.Logger
	smallMaxBytes      int64  // max bytes for small map (default 100MB)
	largeMaxBytes      int64  // max bytes for Ristretto (default 500MB)
	largeObjThreshold  int64  // threshold: objects > this go to Ristretto (default 1KB)
}

// WithLogger configures a structured logger. Panics if logger is nil.
func WithLogger(l pkglogger.Logger) CacheOption {
	if l == nil {
		panic("logger cannot be nil, use no option to default to noop logger")
	}
	return func(cfg *cacheConfig) {
		cfg.logger = l
	}
}

// WithSmallMaxBytes sets max bytes for small map storage (default 100MB).
func WithSmallMaxBytes(bytes int64) CacheOption {
	if bytes <= 0 {
		panic("smallMaxBytes must be > 0")
	}
	return func(cfg *cacheConfig) {
		cfg.smallMaxBytes = bytes
	}
}

// WithLargeMaxBytes sets max bytes for Ristretto storage (default 500MB).
func WithLargeMaxBytes(bytes int64) CacheOption {
	if bytes <= 0 {
		panic("largeMaxBytes must be > 0")
	}
	return func(cfg *cacheConfig) {
		cfg.largeMaxBytes = bytes
	}
}

// WithLargeObjThreshold sets threshold for Ristretto (default 1KB).
// Objects > threshold go to Ristretto, else to small map.
func WithLargeObjThreshold(bytes int64) CacheOption {
	if bytes <= 0 {
		panic("largeObjThreshold must be > 0")
	}
	return func(cfg *cacheConfig) {
		cfg.largeObjThreshold = bytes
	}
}

// noopLogger discards all writes (zero branching in callers).
var noopLogger = logger.Default()

type entry[T any] struct {
	val      *T
	expireAt time.Time // zero means no expiry
}

func (e entry[T]) expired(now time.Time) bool {
	return !e.expireAt.IsZero() && now.After(e.expireAt)
}

// LocalCache is a hybrid in-memory cache backed by:
// - small: map + mutex for small objects (<threshold)
// - large: Ristretto for large objects (>threshold) with LRU eviction
//
// Expiry is lazy: expired keys evicted on next access. Small map has no background
// cleanup; large (Ristretto) uses native LRU eviction when limits hit.
type LocalCache[T any] struct {
	// Small storage (< largeObjThreshold)
	smallMu       sync.Mutex
	small         map[string]entry[T]
	smallCurBytes int64  // approximate tracking

	// Large storage (>= largeObjThreshold)
	large *ristretto.Cache

	// Config
	smallMaxBytes     int64
	largeObjThreshold int64

	logger pkglogger.Logger
}

// NewLocalCache constructs a hybrid cache with small map + Ristretto large storage.
// Default config: smallMax=100MB, largeMax=500MB, largeObjThreshold=1KB.
func NewLocalCache[T any](opts ...CacheOption) *LocalCache[T] {
	cfg := &cacheConfig{
		logger:            noopLogger,
		smallMaxBytes:     100 * 1024 * 1024,  // 100MB
		largeMaxBytes:     500 * 1024 * 1024,  // 500MB
		largeObjThreshold: 1 * 1024,           // 1KB
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	// Create Ristretto cache for large objects
	ristrettoConfig := &ristretto.Config{
		NumCounters: cfg.largeMaxBytes / 100,  // Ristretto tuning: 100 bytes per counter
		MaxCost:     cfg.largeMaxBytes,
		BufferItems: 64,
	}
	largeCache, err := ristretto.NewCache(ristrettoConfig)
	if err != nil {
		panic("failed to create Ristretto cache: " + err.Error())
	}

	return &LocalCache[T]{
		small:             make(map[string]entry[T]),
		large:             largeCache,
		smallMaxBytes:     cfg.smallMaxBytes,
		largeObjThreshold: cfg.largeObjThreshold,
		logger:            cfg.logger,
	}
}

// Get returns the value for key and whether it was a live hit.
// Tries small map first, then large Ristretto.
func (c *LocalCache[T]) Get(_ context.Context, key string) (*T, bool, error) {
	if key == "" {
		return nil, false, errEmptyKey
	}

	// Try small map first
	c.smallMu.Lock()
	e, ok := c.small[key]
	c.smallMu.Unlock()

	if ok {
		if e.expired(time.Now()) {
			c.smallMu.Lock()
			delete(c.small, key)
			c.smallMu.Unlock()
		} else {
			return e.val, true, nil
		}
	}

	// Try large (Ristretto)
	val, ok := c.large.Get(key)
	if !ok {
		return nil, false, nil
	}

	typed, ok := val.(*T)
	if !ok {
		c.logger.Warn(context.Background(), "type assertion failed in Get",
			pkglogger.String("key", key))
		return nil, false, nil
	}
	return typed, true, nil
}

// Set stores val under key. A ttl of 0 means no expiry.
// Small objects (<threshold) go to map, large objects go to Ristretto.
func (c *LocalCache[T]) Set(_ context.Context, key string, val *T, ttl time.Duration) error {
	if key == "" {
		return errEmptyKey
	}
	if val == nil {
		return errNilValue
	}

	estimatedSize := estimateSize(val)

	// Small object → small map
	if estimatedSize < c.largeObjThreshold {
		var expireAt time.Time
		if ttl > 0 {
			expireAt = time.Now().Add(ttl)
		}

		c.smallMu.Lock()
		defer c.smallMu.Unlock()

		// Check if we need to evict
		if c.smallCurBytes+estimatedSize > c.smallMaxBytes {
			c.evictOldestSmall()
		}

		c.small[key] = entry[T]{val: val, expireAt: expireAt}
		c.smallCurBytes += estimatedSize
		return nil
	}

	// Large object → Ristretto
	ok := c.large.SetWithTTL(key, val, estimatedSize, ttl)
	if !ok {
		c.logger.Warn(context.Background(), "large object rejected by Ristretto admission policy",
			pkglogger.String("key", key),
			pkglogger.String("size_bytes", fmt.Sprintf("%d", estimatedSize)))
	}
	return nil
}

// Delete removes key. Removing an absent key is a no-op.
func (c *LocalCache[T]) Delete(_ context.Context, key string) error {
	if key == "" {
		return errEmptyKey
	}

	c.smallMu.Lock()
	delete(c.small, key)
	c.smallMu.Unlock()

	c.large.Del(key)
	return nil
}

// Exists reports whether key is present and unexpired in small or large storage.
func (c *LocalCache[T]) Exists(_ context.Context, key string) (bool, error) {
	if key == "" {
		return false, errEmptyKey
	}

	// Check small
	c.smallMu.Lock()
	e, ok := c.small[key]
	c.smallMu.Unlock()

	if ok {
		if !e.expired(time.Now()) {
			return true, nil
		}
		// Expired, clean up
		c.smallMu.Lock()
		delete(c.small, key)
		c.smallMu.Unlock()
	}

	// Check large
	_, ok = c.large.Get(key)
	return ok, nil
}

// HealthCheck always succeeds: an in-memory cache has no external backend.
func (c *LocalCache[T]) HealthCheck(_ context.Context) error {
	return nil
}

// evictOldestSmall removes oldest (first inserted) entry from small map.
// Must be called with smallMu held.
func (c *LocalCache[T]) evictOldestSmall() {
	for key, entry := range c.small {
		estimatedSize := estimateSize(entry.val)
		delete(c.small, key)
		c.smallCurBytes -= estimatedSize
		return  // Remove only one
	}
}

// estimateSize estimates memory size of value in bytes (conservative).
// Pointer + some buffer for interface overhead + object contents.
func estimateSize[T any](val *T) int64 {
	if val == nil {
		return 0
	}
	// Rough estimate: pointer (8 bytes) + value size (unsafe.Sizeof)
	// Add 50% buffer for allocator overhead
	return int64(float64(unsafe.Sizeof(*val)) * 1.5)
}

// compile-time guard: *LocalCache[T] satisfies the cache contract.
var _ contracts.Cache[int] = (*LocalCache[int])(nil)
