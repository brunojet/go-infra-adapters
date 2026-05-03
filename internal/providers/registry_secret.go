package providers

import (
	secretcontracts "github.com/brunojet/go-infra-adapters/pkg/secrets/contracts"
)

// providerEntry holds per-feature factories for a provider.
type providerEntry struct {
	secretFactory func() secretcontracts.SecretAPI
}

func (e *providerEntry) Secret() secretcontracts.SecretAPI {
	if e == nil || e.secretFactory == nil {
		return nil
	}
	return e.secretFactory()
}

// secretWrapper delegates SecretAPI calls to a lazily-constructed concrete
// implementation provided by the registry's Lazy helper.
type secretWrapper struct {
	lazy *Lazy[secretcontracts.SecretAPI]
}

// NewSecretWithClient implements [contracts.SecretAPI].
func (w *secretWrapper) NewSecretWithClient(name string, client secretcontracts.SecretClientAPI) (secretcontracts.SecretAdapter, error) {
	real, err := w.lazy.Get()
	if err != nil {
		return nil, err
	}
	return real.NewSecretWithClient(name, client)
}

func (w *secretWrapper) NewSecret(name string) (secretcontracts.SecretAdapter, error) {
	real, err := w.lazy.Get()
	if err != nil {
		return nil, err
	}
	return real.NewSecret(name)
}

// RegisterSecretProvider registers or updates only the secret factory for a provider.
// It accepts a constructor that returns (SecretAPI, error). The registry wraps
// the constructor with a shared Lazy helper and exposes a delegating wrapper
// that will construct the real implementation on first use (or during
// validation via Lazy.Validate/Get).
func RegisterSecretProvider(name string, ctor func() (secretcontracts.SecretAPI, error)) {
	mu.Lock()
	defer mu.Unlock()
	e, ok := providers[name]
	if !ok {
		e = &providerEntry{}
		providers[name] = e
	}
	lazy := NewLazy(ctor)
	e.secretFactory = func() secretcontracts.SecretAPI {
		return &secretWrapper{lazy: lazy}
	}
}
