// Package local provides a thin facade over the internal in-memory cache adapter
// so consumers can construct caches without importing internal packages directly.
package local

import (
	internallocal "github.com/brunojet/go-infra-adapters/v4/internal/cache/local"
	"github.com/brunojet/go-infra-adapters/v4/pkg/cache/contracts"
	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

// Option configures an in-memory cache. Use With* functions to create options.
type Option = internallocal.CacheOption

// WithLogger configures a structured logger. Panics if logger is nil.
// Omit this option to use the noop default logger.
func WithLogger(l logger.Logger) Option { return internallocal.WithLogger(l) }

// NewCache creates a type-safe in-memory contracts.Cache[T].
// T must be JSON-serialisable to remain swappable with distributed adapters.
func NewCache[T any](opts ...Option) contracts.Cache[T] {
	return internallocal.NewLocalCache[T](opts...)
}
