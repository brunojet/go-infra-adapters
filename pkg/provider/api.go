// Package provider exposes a small public facade for retrieving registered
// provider implementations. The package keeps the public surface intentionally
// minimal.
package provider

import (
	"github.com/brunojet/go-infra-adapters/internal/provider"
)

// Provider is an alias for the internal provider type.
type Provider = provider.Provider

// GetProvider returns the named provider and a boolean indicating presence.
func GetProvider(name string) (Provider, bool) { return provider.GetProvider(name) }

// SupportedProviders lists the names of providers currently registered.
func SupportedProviders() []string { return provider.SupportedProviders() }
