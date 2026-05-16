// Package contracts defines the small public Provider contract that allows
// consumers to obtain feature-specific APIs (Secrets, Storage, etc.) from
// provider implementations.
package contracts

import (
	"github.com/brunojet/go-infra-adapters/pkg/secret/contracts"
	strcts "github.com/brunojet/go-infra-adapters/pkg/storage/contracts"
)

// Provider exposes feature constructors for a cloud provider. Add methods here
// as features are implemented (Storage, PubSub, etc.).
type Provider interface {
	// Secret returns a default SecretAPI instance or nil if unsupported.
	Secret() contracts.SecretAPI

	// Storage returns a default StorageAPI instance or nil if unsupported.
	Storage() strcts.StorageAPI
}
