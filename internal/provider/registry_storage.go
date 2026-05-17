//nolint:dupl // small delegating wrapper intentionally mirrors secret counterpart
package provider

import (
	"github.com/brunojet/go-infra-adapters/pkg/storage/contracts"
)

// storageWrapper delegates StorageAPI calls to a lazily-constructed concrete
// implementation provided by the registry's Lazy helper.
//
//nolint:dupl // small delegating wrapper intentionally mirrors secret counterpart
type storageWrapper struct {
	*lazyWrapper[contracts.StorageAPI]
}

func (w *storageWrapper) NewBucketWithClient(name string, client contracts.StorageClientAPI) (contracts.BucketAdapter, error) {
	impl, err := w.get()
	if err != nil {
		return nil, err
	}
	return impl.NewBucketWithClient(name, client)
}

func (w *storageWrapper) NewBucket(name string) (contracts.BucketAdapter, error) {
	impl, err := w.get()
	if err != nil {
		return nil, err
	}
	return impl.NewBucket(name)
}

// RegisterStorageProvider registers or updates only the storage factory for a provider.
// It accepts a constructor that returns (StorageAPI, error). The registry wraps
// the constructor with a shared Lazy helper and exposes a delegating wrapper
// that will construct the real implementation on first use.
func RegisterStorageProvider(name string, ctor func() (contracts.StorageAPI, error)) {
	mu.Lock()
	defer mu.Unlock()
	e, ok := providers[name]
	if !ok {
		e = &providerEntry{}
		providers[name] = e
	}
	lazy := newLazyWrapper(ctor)
	e.storageFactory = func() contracts.StorageAPI {
		return &storageWrapper{lazy}
	}
}
