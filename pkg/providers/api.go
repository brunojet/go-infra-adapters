package providers

import (
	"github.com/brunojet/go-infra-adapters/internal/providers"
)

type (
	Provider = providers.Provider
)

func GetProvider(name string) (Provider, bool) {
	return providers.GetProvider(name)
}

func SupportedProviders() []string {
	return providers.SupportedProviders()
}
