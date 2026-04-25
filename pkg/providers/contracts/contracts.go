package contracts

import (
	"github.com/brunojet/go-infra-adapters/pkg/secrets/contracts"
)

// Provider exposes feature constructors for a cloud provider.
// Add methods here as features are implemented (Storage, PubSub, etc.).
type Provider interface {
	// Secret returns a default SecretAPI instance or nil if unsupported.
	Secret() contracts.SecretAPI
}
