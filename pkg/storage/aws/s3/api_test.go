package s3

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/golang/mock/gomock"

	mocksm "github.com/brunojet/go-infra-adapters/v4/internal/storage/aws/s3/mock"
)

// mockTransferManagerForTest delegates to S3 API mock
type mockTransferManagerForTest struct {
	s3api ClientAPI
}

func (m *mockTransferManagerForTest) GetObject(ctx context.Context, input *transfermanager.GetObjectInput, opts ...func(*transfermanager.Options)) (*transfermanager.GetObjectOutput, error) {
	// Not used in this test
	return nil, nil
}

func (m *mockTransferManagerForTest) UploadObject(ctx context.Context, input *transfermanager.UploadObjectInput, opts ...func(*transfermanager.Options)) (*transfermanager.UploadObjectOutput, error) {
	// Not used in this test
	return nil, nil
}

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
	mockTM := &mockTransferManagerForTest{s3api: mock}
	// For testing with mocked client, must also provide mocked transfer manager
	api, err := NewStorageAPI(WithClient(mock), WithTransferManager(mockTM))
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
