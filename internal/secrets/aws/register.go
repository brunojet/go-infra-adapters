package aws

import (
	"github.com/brunojet/go-infra-adapters/internal/providers"
	"github.com/brunojet/go-infra-adapters/pkg/secrets/contracts"
)

func init() {
	// Register only the Secret factory; the registry will wrap this ctor with
	// a shared Lazy and delegating wrapper so providers don't need to duplicate
	// lazy/validation logic.
	providers.RegisterSecretProvider("aws", func() (contracts.SecretAPI, error) {
		return &SecretsClient{}, nil
	})
}
