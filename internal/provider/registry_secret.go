//nolint:dupl // small delegating wrapper intentionally mirrors storage counterpart
package provider

import (
	"github.com/brunojet/go-infra-adapters/pkg/secret/contracts"
)

// secretWrapper delegates SecretAPI calls to a lazily-constructed concrete
// implementation provided by the registry's Lazy helper.
//
//nolint:dupl // small delegating wrapper intentionally mirrors storage counterpart
type secretWrapper struct {
	*lazyWrapper[contracts.SecretAPI]
}

// NewSecretWithClient implements [contracts.SecretAPI].
func (w *secretWrapper) NewSecretWithClient(name string, client contracts.SecretClientAPI) (contracts.SecretAdapter, error) {
	impl, err := w.get()
	if err != nil {
		return nil, err
	}
	return impl.NewSecretWithClient(name, client)
}

func (w *secretWrapper) NewSecret(name string) (contracts.SecretAdapter, error) {
	impl, err := w.get()
	if err != nil {
		return nil, err
	}
	return impl.NewSecret(name)
}

// RegisterSecretProvider registers or updates only the secret factory for a provider.
// It accepts a constructor that returns (SecretAPI, error). The registry wraps
// the constructor with a shared Lazy helper and exposes a delegating wrapper
// that will construct the real implementation on first use (or during
// validation via Lazy.Validate/Get).
func RegisterSecretProvider(name string, ctor func() (contracts.SecretAPI, error)) {
	mu.Lock()
	defer mu.Unlock()
	e, ok := providers[name]
	if !ok {
		e = &providerEntry{}
		providers[name] = e
	}
	lazy := newLazyWrapper(ctor)
	e.secretFactory = func() contracts.SecretAPI {
		return &secretWrapper{lazy}
	}
}
