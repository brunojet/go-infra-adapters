package provider

import (
	"testing"

	scrcts "github.com/brunojet/go-infra-adapters/pkg/secret/contracts"
	scrmock "github.com/brunojet/go-infra-adapters/pkg/secret/mock"
	strcts "github.com/brunojet/go-infra-adapters/pkg/storage/contracts"
	strmock "github.com/brunojet/go-infra-adapters/pkg/storage/mock"
	"github.com/golang/mock/gomock"
)

func TestProviderEntry_NilReceiver(t *testing.T) {
	var e *providerEntry
	if e.Secret() != nil {
		t.Fatalf("expected nil secret for nil receiver")
	}
	if e.Storage() != nil {
		t.Fatalf("expected nil storage for nil receiver")
	}
}

func TestProviderEntry_Factories(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ms := scrmock.NewMockSecretAPI(ctrl)
	mstore := strmock.NewMockStorageAPI(ctrl)

	e := &providerEntry{
		secretFactory:  func() scrcts.SecretAPI { return ms },
		storageFactory: func() strcts.StorageAPI { return mstore },
	}
	if e.Secret() == nil {
		t.Fatalf("expected secret factory to produce non-nil")
	}
	if e.Storage() == nil {
		t.Fatalf("expected storage factory to produce non-nil")
	}
}
