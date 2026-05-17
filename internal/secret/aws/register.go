package aws

import (
	"github.com/brunojet/go-infra-adapters/internal/provider"
	"github.com/brunojet/go-infra-adapters/pkg/secret/contracts"
)

func init() {
	// Register only the Secret factory; the registry will wrap this ctor with
	// a shared Lazy and delegating wrapper so providers don't need to duplicate
	// lazy/validation logic.
	provider.RegisterSecretProvider("aws", func() (contracts.SecretAPI, error) {
		return &SecretsClient{}, nil
	})
}
