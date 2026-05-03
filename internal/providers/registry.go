package providers

import (
	"sync"

	providerscontracts "github.com/brunojet/go-infra-adapters/pkg/providers/contracts"
)

type (
	// Provider is the runtime view exposed to consumers (keeps existing API).
	Provider = providerscontracts.Provider
)

// Lazy is a small, generic helper that constructs a value on first use in a
// concurrency-safe way. The constructor may return an error which will be
// cached and returned on subsequent calls.
type Lazy[T any] struct {
	once sync.Once
	val  T
	err  error
	ctor func() (T, error)
}

// NewLazy creates a Lazy wrapper around the provided constructor.
func NewLazy[T any](ctor func() (T, error)) *Lazy[T] {
	return &Lazy[T]{ctor: ctor}
}

// Get ensures the constructor has run and returns the constructed value or
// the error observed during construction.
func (l *Lazy[T]) Get() (T, error) {
	l.once.Do(func() {
		l.val, l.err = l.ctor()
	})
	return l.val, l.err
}

// Validate attempts construction and returns any construction error.
func (l *Lazy[T]) Validate() error {
	_, err := l.Get()
	return err
}

var (
	mu        sync.RWMutex
	providers = map[string]*providerEntry{}
)

// GetProvider returns a runtime view of the registered provider.
func GetProvider(name string) (Provider, bool) {
	mu.RLock()
	e, ok := providers[name]
	mu.RUnlock()
	if !ok {
		return nil, false
	}
	return e, true
}

// SupportedProviders lists registered providers.
func SupportedProviders() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(providers))
	for k := range providers {
		out = append(out, k)
	}
	return out
}
