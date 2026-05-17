package provider

import (
	scrcts "github.com/brunojet/go-infra-adapters/pkg/secret/contracts"
	strcts "github.com/brunojet/go-infra-adapters/pkg/storage/contracts"
)

// providerEntry holds per-feature factories for a provider.
type providerEntry struct {
	secretFactory  func() scrcts.SecretAPI
	storageFactory func() strcts.StorageAPI
}

func (e *providerEntry) Secret() scrcts.SecretAPI {
	if e == nil || e.secretFactory == nil {
		return nil
	}
	return e.secretFactory()
}

func (e *providerEntry) Storage() strcts.StorageAPI {
	if e == nil || e.storageFactory == nil {
		return nil
	}
	return e.storageFactory()
}
