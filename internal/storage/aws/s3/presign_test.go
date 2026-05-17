package s3

import (
	"context"
	"errors"
	"testing"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/brunojet/go-infra-adapters/internal/storage/aws/s3/mock"
)

func TestPresign_FallbackAndUnsupported(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockS3API(ctrl)

	b := &bucketAdapter{client: mockClient, bucket: "b"}
	if _, err := b.PresignGet(context.Background(), "k", time.Minute); err == nil {
		t.Fatalf("expected presign not supported error")
	}
	if _, err := b.PresignPut(context.Background(), "k", time.Minute); err == nil {
		t.Fatalf("expected presign not supported error")
	}
}

func TestHandlePresignResponse_ErrorPropagates(t *testing.T) {
	inErr := errors.New("boom")
	u, err := handlePresignResponse(nil, inErr)
	require.Equal(t, "", u)
	require.Equal(t, inErr, err)
}

func TestHandlePresignResponse_NilResponse(t *testing.T) {
	u, err := handlePresignResponse(nil, nil)
	require.Equal(t, "", u)
	require.Error(t, err)
	require.Equal(t, "empty presigned response", err.Error())
}

func TestHandlePresignResponse_Success(t *testing.T) {
	resp := &v4.PresignedHTTPRequest{URL: "https://example.com/success"}
	u, err := handlePresignResponse(resp, nil)
	require.NoError(t, err)
	require.Equal(t, "https://example.com/success", u)
}
