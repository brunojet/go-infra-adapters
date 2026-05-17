package provider

import (
	"testing"

	"github.com/brunojet/go-infra-adapters/pkg/storage/contracts"
	"github.com/brunojet/go-infra-adapters/pkg/storage/mock"
	"github.com/golang/mock/gomock"
)

func TestRegisterStorageProvider_DelegatesToCtor(t *testing.T) {
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

	ms := mock.NewMockStorageAPI(ctrl)
	mb := mock.NewMockBucketAdapter(ctrl)

	ms.EXPECT().NewBucket("bn").Return(mb, nil)
	mb.EXPECT().BucketName().Return("bn")

	RegisterStorageProvider("test-store", func() (contracts.StorageAPI, error) { return ms, nil })

	p, ok := GetProvider("test-store")
	if !ok {
		t.Fatalf("provider not registered")
	}
	s := p.Storage()
	if s == nil {
		t.Fatalf("expected StorageAPI wrapper")
	}
	b, err := s.NewBucket("bn")
	if err != nil {
		t.Fatalf("NewBucket failed: %v", err)
	}
	if b.BucketName() != "bn" {
		t.Fatalf("unexpected bucket name")
	}
}

func TestRegisterStorageProvider_NewBucketWithClient_Delegates(t *testing.T) {
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

	ms := mock.NewMockStorageAPI(ctrl)
	mb := mock.NewMockBucketAdapter(ctrl)

	ms.EXPECT().NewBucketWithClient("bn", gomock.Any()).Return(mb, nil)
	mb.EXPECT().BucketName().Return("bn")

	RegisterStorageProvider("test-store", func() (contracts.StorageAPI, error) { return ms, nil })

	p, ok := GetProvider("test-store")
	if !ok {
		t.Fatalf("provider not registered")
	}
	s := p.Storage()
	if s == nil {
		t.Fatalf("expected StorageAPI wrapper")
	}
	b, err := s.NewBucketWithClient("bn", nil)
	if err != nil {
		t.Fatalf("NewBucketWithClient failed: %v", err)
	}
	if b.BucketName() != "bn" {
		t.Fatalf("unexpected bucket name")
	}
}
