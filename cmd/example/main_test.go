package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"

	internalproviders "github.com/brunojet/go-infra-adapters/internal/provider"
	scrcts "github.com/brunojet/go-infra-adapters/pkg/secret/contracts"
	mocksecret "github.com/brunojet/go-infra-adapters/pkg/secret/mock"
	storagecontracts "github.com/brunojet/go-infra-adapters/pkg/storage/contracts"
	mockstorage "github.com/brunojet/go-infra-adapters/pkg/storage/mock"
)

func TestExampleMain_WithMockSecret(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSecretAPI := mocksecret.NewMockSecretAPI(ctrl)
	mockSecretAdapter := mocksecret.NewMockSecretAdapter(ctrl)

	secretStr := "supersecret"
	mockSecretAdapter.EXPECT().GetCurrent(gomock.Any()).Return(&scrcts.SecretValue{Data: []byte(secretStr), Metadata: map[string]string{"arn": "arn:aws:secrets:example"}}, nil)
	mockSecretAPI.EXPECT().NewSecret("example-secret").Return(mockSecretAdapter, nil)

	internalproviders.RegisterSecretProvider("aws", func() (scrcts.SecretAPI, error) {
		return mockSecretAPI, nil
	})

	secretAPI := p.Secret()
	if secretAPI == nil {
		t.Fatal("expected secret API")
	}
	adapter, err := secretAPI.NewSecret("example-secret")
	if err != nil {
		t.Fatalf("NewSecret failed: %v", err)
	}
	val, err := adapter.GetCurrent(context.Background())
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}
	if string(val.Data) != secretStr {
		t.Fatalf("unexpected secret data: %s", string(val.Data))
	}
	if val.Metadata["arn"] != "arn:aws:secrets:example" {
		t.Fatalf("unexpected metadata: %#v", val.Metadata)
	}
}

func TestExampleMain_Main_FullPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSecretAPI := mocksecret.NewMockSecretAPI(ctrl)
	mockSecretAdapter := mocksecret.NewMockSecretAdapter(ctrl)

	mockSecretAdapter.EXPECT().GetCurrent(gomock.Any()).Return(&scrcts.SecretValue{Data: []byte("s"), Metadata: map[string]string{"arn": "a"}}, nil)
	mockSecretAPI.EXPECT().NewSecret("example-secret").Return(mockSecretAdapter, nil)

	internalproviders.RegisterSecretProvider("aws", func() (scrcts.SecretAPI, error) {
		return mockSecretAPI, nil
	})

	// call main() exercising the happy path
	main()
}

func TestExampleMain_Main_SecretNotExposed(t *testing.T) {
	// ensure main does not panic when Secret() is nil
	old := p
	p = noSecretProvider{}
	defer func() { p = old }()
	main()
}

// noSecretProvider is a test helper implementing providers.Provider but
// returning nil for Secret() so main() will handle "not exposed" path.
type noSecretProvider struct{}

func (noSecretProvider) Secret() scrcts.SecretAPI { return nil }

func (noSecretProvider) Storage() storagecontracts.StorageAPI { return nil }

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	_ = w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old
	return string(out)
}

func TestMain_SecretNewError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSecretAPI := mocksecret.NewMockSecretAPI(ctrl)
	mockSecretAPI.EXPECT().NewSecret("example-secret").Return(nil, os.ErrNotExist)

	internalproviders.RegisterSecretProvider("aws", func() (scrcts.SecretAPI, error) {
		return mockSecretAPI, nil
	})

	out := captureStdout(func() { main() })
	if !strings.Contains(out, "failed to get secret adapter") {
		t.Fatalf("expected secret-new error message, got: %s", out)
	}
}

func TestMain_SecretGetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSecretAPI := mocksecret.NewMockSecretAPI(ctrl)
	mockAdapter := mocksecret.NewMockSecretAdapter(ctrl)
	mockSecretAPI.EXPECT().NewSecret("example-secret").Return(mockAdapter, nil)
	mockAdapter.EXPECT().GetCurrent(gomock.Any()).Return(nil, os.ErrPermission)

	internalproviders.RegisterSecretProvider("aws", func() (scrcts.SecretAPI, error) {
		return mockSecretAPI, nil
	})

	out := captureStdout(func() { main() })
	if !strings.Contains(out, "failed to get secret:") {
		t.Fatalf("expected secret-get error message, got: %s", out)
	}
}

func TestMain_StoragePutErrorAndSuccessFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Secret happy path
	mockSecretAPI := mocksecret.NewMockSecretAPI(ctrl)
	mockSecretAdapter := mocksecret.NewMockSecretAdapter(ctrl)
	mockSecretAPI.EXPECT().NewSecret("example-secret").Return(mockSecretAdapter, nil)
	mockSecretAdapter.EXPECT().GetCurrent(gomock.Any()).Return(&scrcts.SecretValue{Data: []byte("x"), Metadata: map[string]string{}}, nil)
	internalproviders.RegisterSecretProvider("aws", func() (scrcts.SecretAPI, error) { return mockSecretAPI, nil })

	// Storage: first simulate PutObject error path
	mockStorageAPI := mockstorage.NewMockStorageAPI(ctrl)
	mockBucket := mockstorage.NewMockBucketAdapter(ctrl)
	mockStorageAPI.EXPECT().NewBucket("example-bucket").Return(mockBucket, nil)
	mockBucket.EXPECT().PutObject(gomock.Any(), gomock.Any()).Return(os.ErrInvalid)
	internalproviders.RegisterStorageProvider("aws", func() (storagecontracts.StorageAPI, error) { return mockStorageAPI, nil })

	out := captureStdout(func() { main() })
	if !strings.Contains(out, "PutObject failed") {
		t.Fatalf("expected PutObject failure message, got: %s", out)
	}

	// Now simulate full success path to cover prints
	ctrl.Finish()
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockSecretAPI = mocksecret.NewMockSecretAPI(ctrl)
	mockSecretAdapter = mocksecret.NewMockSecretAdapter(ctrl)
	mockSecretAPI.EXPECT().NewSecret("example-secret").Return(mockSecretAdapter, nil)
	mockSecretAdapter.EXPECT().GetCurrent(gomock.Any()).Return(&scrcts.SecretValue{Data: []byte("x")}, nil)
	internalproviders.RegisterSecretProvider("aws", func() (scrcts.SecretAPI, error) { return mockSecretAPI, nil })

	mockStorageAPI = mockstorage.NewMockStorageAPI(ctrl)
	mockBucket = mockstorage.NewMockBucketAdapter(ctrl)
	data := []byte("hello storage example")
	mockStorageAPI.EXPECT().NewBucket("example-bucket").Return(mockBucket, nil)
	mockBucket.EXPECT().PutObject(gomock.Any(), gomock.Any()).Return(nil)
	mockBucket.EXPECT().HeadObject(gomock.Any(), "example.txt", gomock.AssignableToTypeOf(&storagecontracts.ObjectInfo{})).DoAndReturn(
		func(ctx context.Context, key string, info *storagecontracts.ObjectInfo) error {
			*info = storagecontracts.ObjectInfo{Key: "example.txt", Size: int64(len(data)), ContentType: "text/plain"}
			return nil
		})
	mockBucket.EXPECT().GetObject(gomock.Any(), "example.txt", gomock.AssignableToTypeOf(&storagecontracts.BucketObject{})).DoAndReturn(
		func(ctx context.Context, key string, obj *storagecontracts.BucketObject) error {
			obj.Info = storagecontracts.ObjectInfo{Key: "example.txt", Size: int64(len(data)), ContentType: "text/plain"}
			obj.Body = io.NopCloser(bytes.NewReader(data))
			return nil
		})
	internalproviders.RegisterStorageProvider("aws", func() (storagecontracts.StorageAPI, error) { return mockStorageAPI, nil })

	out2 := captureStdout(func() { main() })
	if !strings.Contains(out2, "Get: key=example.txt") {
		t.Fatalf("expected Get printout, got: %s", out2)
	}
}

func TestExampleMain_WithMockStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorageAPI := mockstorage.NewMockStorageAPI(ctrl)
	mockBucket := mockstorage.NewMockBucketAdapter(ctrl)

	data := []byte("hello storage example")

	mockStorageAPI.EXPECT().NewBucket("example-bucket").Return(mockBucket, nil)
	mockBucket.EXPECT().PutObject(gomock.Any(), gomock.Any()).Return(nil)
	mockBucket.EXPECT().HeadObject(gomock.Any(), "example.txt", gomock.AssignableToTypeOf(&storagecontracts.ObjectInfo{})).
		DoAndReturn(func(ctx context.Context, key string, info *storagecontracts.ObjectInfo) error {
			*info = storagecontracts.ObjectInfo{Key: "example.txt", Size: int64(len(data)), ContentType: "text/plain", Metadata: map[string]string{"x": "y"}}
			return nil
		})
	mockBucket.EXPECT().GetObject(gomock.Any(), "example.txt", gomock.AssignableToTypeOf(&storagecontracts.BucketObject{})).
		DoAndReturn(func(ctx context.Context, key string, obj *storagecontracts.BucketObject) error {
			obj.Info = storagecontracts.ObjectInfo{Key: "example.txt", Size: int64(len(data)), ContentType: "text/plain"}
			obj.Body = io.NopCloser(bytes.NewReader(data))
			return nil
		})

	internalproviders.RegisterStorageProvider("aws", func() (storagecontracts.StorageAPI, error) {
		return mockStorageAPI, nil
	})

	storageAPI := p.Storage()
	if storageAPI == nil {
		t.Fatal("expected storage API")
	}
	bkt, err := storageAPI.NewBucket("example-bucket")
	if err != nil {
		t.Fatalf("NewBucket failed: %v", err)
	}

	put := &storagecontracts.BucketObject{Info: storagecontracts.ObjectInfo{Key: "example.txt", Size: int64(len(data)), ContentType: "text/plain"}, Body: io.NopCloser(bytes.NewReader(data))}
	if err := bkt.PutObject(context.Background(), put); err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}

	var info storagecontracts.ObjectInfo
	if err := bkt.HeadObject(context.Background(), "example.txt", &info); err != nil {
		t.Fatalf("HeadObject failed: %v", err)
	}
	if info.Key != "example.txt" || info.Size != int64(len(data)) {
		t.Fatalf("unexpected info: %#v", info)
	}

	got := &storagecontracts.BucketObject{}
	if err := bkt.GetObject(context.Background(), "example.txt", got); err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}
	body, err := io.ReadAll(got.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	if err := got.Close(); err != nil {
		t.Fatalf("close body failed: %v", err)
	}
	if string(body) != string(data) {
		t.Fatalf("unexpected body: %s", string(body))
	}
}
