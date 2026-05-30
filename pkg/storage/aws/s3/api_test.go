package s3

import (
	"testing"

	"github.com/golang/mock/gomock"

	mocksm "github.com/brunojet/go-infra-adapters/internal/storage/aws/s3/mock"
)

func TestOptionsWrappers_NotNil(t *testing.T) {
	if WithRegion("r") == nil {
		t.Fatalf("WithRegion returned nil")
	}
	if WithEndpoint("e") == nil {
		t.Fatalf("WithEndpoint returned nil")
	}
}

func TestExists_pkg_storage_s3_options(t *testing.T) {}

func TestWithClient_UsedByNewStorageAPI(t *testing.T) {
	mock := mocksm.NewMockS3API(gomock.NewController(t))
	api, err := NewStorageAPI(WithClient(mock))
	if err != nil || api == nil {
		t.Fatalf("NewStorageAPI with injected client: %v", err)
	}
}

func TestNewStorageAPI_Wrapper(t *testing.T) {
	api, err := NewStorageAPI(WithRegion("us-east-1"), WithEndpoint("http://localhost"))
	if err != nil || api == nil {
		t.Fatalf("NewStorageAPI failed: %v", err)
	}
}
