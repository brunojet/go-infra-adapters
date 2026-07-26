// Package local provides an in-memory implementation of the cache contract.
// It is a real single-process cache (map + RWMutex + TTL), not a test stub,
// and is swappable for a distributed adapter via the same contracts.Cache[T].
package local

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unsafe"

	"github.com/brunojet/go-infra-adapters/v4/internal/logger"
	"github.com/brunojet/go-infra-adapters/v4/pkg/cache/contracts"
	pkglogger "github.com/brunojet/go-infra-adapters/v4/pkg/logger"
	"github.com/dgraph-io/ristretto"
)

var (
	errEmptyKey = errors.New("cache key cannot be empty")
	errNilValue = errors.New("cache value cannot be nil")
)

// CacheOption configures a LocalCache. Use With* functions to create options.
type CacheOption func(*cacheConfig)

type cacheConfig struct {
	logger   pkglogger.Logger
	maxBytes int64 // max bytes for Ristretto cache (default 100MB)
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

// WithMaxBytes sets max bytes for Ristretto storage (default 100MB).
func WithMaxBytes(bytes int64) CacheOption {
	if bytes <= 0 {
		panic("maxBytes must be > 0")
	}
	return func(cfg *cacheConfig) {
		cfg.maxBytes = bytes
	}
}

// noopLogger discards all writes (zero branching in callers).
var noopLogger = logger.Default()

// LocalCache is an in-memory cache backed by Ristretto with LRU eviction.
// Type-safe, bounded, and production-grade.
type LocalCache[T any] struct {
	cache  *ristretto.Cache
	logger pkglogger.Logger
}

// NewLocalCache constructs a cache backed by Ristretto.
// Default: maxBytes=100MB (configured via WithMaxBytes).
func NewLocalCache[T any](opts ...CacheOption) *LocalCache[T] {
	cfg := &cacheConfig{
		logger:   noopLogger,
		maxBytes: 100 * 1024 * 1024, // 100MB default
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	// Create Ristretto cache
	ristrettoConfig := &ristretto.Config{
		NumCounters: cfg.maxBytes / 100, // Ristretto tuning
		MaxCost:     cfg.maxBytes,
		BufferItems: 64,
	}
	c, err := ristretto.NewCache(ristrettoConfig)
	if err != nil {
		panic("failed to create Ristretto cache: " + err.Error())
	}

	return &LocalCache[T]{
		cache:  c,
		logger: cfg.logger,
	}
}

// Get returns the value for key and whether it was a live hit.
func (c *LocalCache[T]) Get(_ context.Context, key string) (val *T, hit bool, err error) {
	if key == "" {
		return nil, false, errEmptyKey
	}

	raw, ok := c.cache.Get(key)
	if !ok {
		return nil, false, nil
	}

	typed, ok := raw.(*T)
	if !ok {
		c.logger.Warn(context.Background(), "type assertion failed in Get",
			pkglogger.String("key", key))
		return nil, false, nil
	}
	return typed, true, nil
}

// Set stores val under key. A ttl of 0 means no expiry.
// Blocks until Ristretto processes the write (synchronous guarantee).
func (c *LocalCache[T]) Set(_ context.Context, key string, val *T, ttl time.Duration) error {
	if key == "" {
		return errEmptyKey
	}
	if val == nil {
		return errNilValue
	}

	cost := estimateSize(val)
	ok := c.cache.SetWithTTL(key, val, cost, ttl)
	if !ok {
		c.logger.Warn(context.Background(), "cache set rejected by admission policy",
			pkglogger.String("key", key),
			pkglogger.String("cost_bytes", fmt.Sprintf("%d", cost)))
	}
	c.cache.Wait() // Ensure write is processed before returning
	return nil
}

// Delete removes key. Removing an absent key is a no-op.
func (c *LocalCache[T]) Delete(_ context.Context, key string) error {
	if key == "" {
		return errEmptyKey
	}
	c.cache.Del(key)
	return nil
}

// Exists reports whether key is present.
func (c *LocalCache[T]) Exists(_ context.Context, key string) (bool, error) {
	if key == "" {
		return false, errEmptyKey
	}
	_, ok := c.cache.Get(key)
	return ok, nil
}

// HealthCheck always succeeds: an in-memory cache has no external backend.
func (c *LocalCache[T]) HealthCheck(_ context.Context) error {
	return nil
}

// estimateSize estimates memory cost of value for Ristretto (in bytes).
// Conservative estimate: value size + overhead buffer.
func estimateSize[T any](val *T) int64 {
	if val == nil {
		return 0
	}
	// Rough estimate: value size + 50% buffer for allocator overhead
	return int64(float64(unsafe.Sizeof(*val)) * 1.5)
}

// compile-time guard: *LocalCache[T] satisfies the cache contract.
var _ contracts.Cache[int] = (*LocalCache[int])(nil)
