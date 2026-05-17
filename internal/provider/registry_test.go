package provider

import (
	"fmt"
	"testing"

	scrcts "github.com/brunojet/go-infra-adapters/pkg/secret/contracts"
	strcts "github.com/brunojet/go-infra-adapters/pkg/storage/contracts"
)

func TestGetProviderAndSupportedProviders(t *testing.T) {
	// Ensure a clean registry for the test and restore after.
	mu.Lock()
	old := providers
	providers = map[string]*providerEntry{}
	mu.Unlock()
	defer func() {
		mu.Lock()
		providers = old
		mu.Unlock()
	}()

	// Add entries directly and verify retrieval and listing.
	mu.Lock()
	providers["p1"] = &providerEntry{}
	providers["p2"] = &providerEntry{}
	mu.Unlock()

	if _, ok := GetProvider("p1"); !ok {
		t.Fatalf("expected provider p1 to be present")
	}

	list := SupportedProviders()
	if len(list) != 2 {
		t.Fatalf("expected 2 supported providers, got %d: %#v", len(list), list)
	}
}

func TestSupportedProviders_EmptyAndPopulated(t *testing.T) {
	mu.Lock()
	old := providers
	providers = map[string]*providerEntry{}
	mu.Unlock()
	defer func() {
		mu.Lock()
		providers = old
		mu.Unlock()
	}()

	if got := SupportedProviders(); len(got) != 0 {
		t.Fatalf("expected no providers, got %v", got)
	}

	RegisterSecretProvider("p", func() (scrcts.SecretAPI, error) { return nil, nil })
	got := SupportedProviders()
	if len(got) != 1 || got[0] != "p" {
		t.Fatalf("unexpected supported providers: %v", got)
	}
}

func TestRegisterAndGetProviders_MultipleFeatures(t *testing.T) {
	// reset registry
	mu.Lock()
	old := providers
	providers = map[string]*providerEntry{}
	mu.Unlock()
	defer func() {
		mu.Lock()
		providers = old
		mu.Unlock()
	}()

	RegisterSecretProvider("p", func() (scrcts.SecretAPI, error) { return nil, nil })
	RegisterStorageProvider("p", func() (strcts.StorageAPI, error) { return nil, nil })

	p, ok := GetProvider("p")
	if !ok {
		t.Fatalf("expected provider to be present")
	}
	if p.Secret() == nil {
		t.Fatalf("expected secret factory present")
	}
	if p.Storage() == nil {
		t.Fatalf("expected storage factory present")
	}
}

func TestRegisterSecretProvider_ConstructorError_Propagates(t *testing.T) {
	mu.Lock()
	old := providers
	providers = map[string]*providerEntry{}
	mu.Unlock()
	defer func() {
		mu.Lock()
		providers = old
		mu.Unlock()
	}()

	RegisterSecretProvider("err-secret", func() (scrcts.SecretAPI, error) { return nil, fmt.Errorf("fail") })
	p, ok := GetProvider("err-secret")
	if !ok {
		t.Fatalf("expected provider to be present")
	}
	s := p.Secret()
	if s == nil {
		t.Fatalf("expected SecretAPI wrapper")
	}
	if _, err := s.NewSecret("sname"); err == nil {
		t.Fatalf("expected error from constructor")
	}
	if _, err := s.NewSecretWithClient("sname", nil); err == nil {
		t.Fatalf("expected error from constructor")
	}
}

func TestRegisterStorageProvider_ConstructorError_Propagates(t *testing.T) {
	mu.Lock()
	old := providers
	providers = map[string]*providerEntry{}
	mu.Unlock()
	defer func() {
		mu.Lock()
		providers = old
		mu.Unlock()
	}()

	RegisterStorageProvider("err-store", func() (strcts.StorageAPI, error) { return nil, fmt.Errorf("fail") })
	p, ok := GetProvider("err-store")
	if !ok {
		t.Fatalf("expected provider to be present")
	}
	s := p.Storage()
	if s == nil {
		t.Fatalf("expected StorageAPI wrapper")
	}
	if _, err := s.NewBucket("bn"); err == nil {
		t.Fatalf("expected error from constructor")
	}
	if _, err := s.NewBucketWithClient("bn", nil); err == nil {
		t.Fatalf("expected error from constructor")
	}
}
