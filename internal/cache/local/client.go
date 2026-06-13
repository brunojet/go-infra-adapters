// Package local provides an in-memory implementation of the cache contract.
// It is a real single-process cache (map + RWMutex + TTL), not a test stub,
// and is swappable for a distributed adapter via the same contracts.Cache[T].
package local

import (
	"context"
	"errors"
	"sync"
	"time"

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
	logger pkglogger.Logger
}

// WithLogger configures a structured logger. Panics if logger is nil.
func WithLogger(logger pkglogger.Logger) CacheOption {
	if logger == nil {
		panic("logger cannot be nil, use no option to default to noop logger")
	}
	return func(cfg *cacheConfig) {
		cfg.logger = logger
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

// LocalCache is an in-memory contracts.Cache[T] backed by a map guarded by a
// Mutex. Expiry is lazy: an expired key is evicted on the next access. Keys
// that are written with a TTL but never read again persist until overwritten;
// a background janitor can be added later if memory pressure warrants it.
type LocalCache[T any] struct {
	mu     sync.Mutex
	store  map[string]entry[T]
	logger pkglogger.Logger
}

// NewLocalCache constructs an empty in-memory cache for value type T.
func NewLocalCache[T any](opts ...CacheOption) *LocalCache[T] {
	cfg := &cacheConfig{logger: noopLogger} // noop by default, zero log branching
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	return &LocalCache[T]{
		store:  make(map[string]entry[T]),
		logger: cfg.logger,
	}
}

// Get returns the value for key and whether it was a live hit.
func (c *LocalCache[T]) Get(_ context.Context, key string) (*T, bool, error) {
	if key == "" {
		return nil, false, errEmptyKey
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.store[key]
	if !ok {
		return nil, false, nil
	}
	if e.expired(time.Now()) {
		delete(c.store, key)
		return nil, false, nil
	}
	return e.val, true, nil
}

// Set stores val under key. A ttl of 0 means no expiry.
func (c *LocalCache[T]) Set(_ context.Context, key string, val *T, ttl time.Duration) error {
	if key == "" {
		return errEmptyKey
	}
	if val == nil {
		return errNilValue
	}
	var expireAt time.Time
	if ttl > 0 {
		expireAt = time.Now().Add(ttl)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = entry[T]{val: val, expireAt: expireAt}
	return nil
}

// Delete removes key. Removing an absent key is a no-op.
func (c *LocalCache[T]) Delete(_ context.Context, key string) error {
	if key == "" {
		return errEmptyKey
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, key)
	return nil
}

// Exists reports whether key is present and unexpired.
func (c *LocalCache[T]) Exists(_ context.Context, key string) (bool, error) {
	if key == "" {
		return false, errEmptyKey
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.store[key]
	if !ok {
		return false, nil
	}
	if e.expired(time.Now()) {
		delete(c.store, key)
		return false, nil
	}
	return true, nil
}

// HealthCheck always succeeds: an in-memory cache has no external backend.
func (c *LocalCache[T]) HealthCheck(_ context.Context) error {
	return nil
}

// compile-time guard: *LocalCache[T] satisfies the cache contract.
var _ contracts.Cache[int] = (*LocalCache[int])(nil)
