package provider

import (
	"testing"

	"github.com/brunojet/go-infra-adapters/pkg/secret/contracts"
	"github.com/brunojet/go-infra-adapters/pkg/secret/mock"
	"github.com/golang/mock/gomock"
)

func TestRegisterSecretProvider_DelegatesToCtor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// clean registry
	mu.Lock()
	old := providers
	providers = map[string]*providerEntry{}
	mu.Unlock()
	defer func() {
		mu.Lock()
		providers = old
		mu.Unlock()
	}()

	ms := mock.NewMockSecretAPI(ctrl)
	ma := mock.NewMockSecretAdapter(ctrl)

	// Expect underlying API to be called when adapter is requested.
	ms.EXPECT().NewSecret("sname").Return(ma, nil)
	ma.EXPECT().Name().Return("sname")

	RegisterSecretProvider("test-secret", func() (contracts.SecretAPI, error) { return ms, nil })

	p, ok := GetProvider("test-secret")
	if !ok {
		t.Fatalf("provider not registered")
	}
	s := p.Secret()
	if s == nil {
		t.Fatalf("expected SecretAPI wrapper")
	}
	ad, err := s.NewSecret("sname")
	if err != nil {
		t.Fatalf("NewSecret failed: %v", err)
	}
	if ad.Name() != "sname" {
		t.Fatalf("unexpected adapter name")
	}
}

func TestRegisterSecretProvider_NewSecretWithClient_Delegates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mu.Lock()
	old := providers
	providers = map[string]*providerEntry{}
	mu.Unlock()
	defer func() {
		mu.Lock()
		providers = old
		mu.Unlock()
	}()

	ms := mock.NewMockSecretAPI(ctrl)
	ma := mock.NewMockSecretAdapter(ctrl)

	ms.EXPECT().NewSecretWithClient("sname", gomock.Any()).Return(ma, nil)
	ma.EXPECT().Name().Return("sname")

	RegisterSecretProvider("test-secret", func() (contracts.SecretAPI, error) { return ms, nil })

	p, ok := GetProvider("test-secret")
	if !ok {
		t.Fatalf("provider not registered")
	}
	s := p.Secret()
	if s == nil {
		t.Fatalf("expected SecretAPI wrapper")
	}
	ad, err := s.NewSecretWithClient("sname", nil)
	if err != nil {
		t.Fatalf("NewSecretWithClient failed: %v", err)
	}
	if ad.Name() != "sname" {
		t.Fatalf("unexpected adapter name")
	}
}
