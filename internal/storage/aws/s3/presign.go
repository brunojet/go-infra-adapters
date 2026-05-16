//nolint:dupl // PresignGet/PresignPut contain similar plumbing for different input types.
package s3

import (
	"context"
	"errors"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func handlePresignResponse(resp *v4.PresignedHTTPRequest, err error) (string, error) {
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", errors.New("empty presigned response")
	}
	return resp.URL, nil
}

// PresignGet returns a presigned GET URL for the given key if the underlying
// client supports presigning. It attempts to create a presign client from a
// concrete *s3.Client; if the configured client implements a compatible
// presign method it will be used as well.
func (b *bucketAdapter) PresignGet(ctx context.Context, key string, expires time.Duration) (string, error) {
	// Prefer creating a presign client from the concrete SDK client when
	// available.
	if c, ok := b.client.(*s3.Client); ok {
		presigner := s3.NewPresignClient(c)
		resp, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &b.bucket, Key: &key}, func(po *s3.PresignOptions) {
			if expires > 0 {
				po.Expires = expires
			}
		})
		return handlePresignResponse(resp, err)
	}
	return "", errors.New("presign not supported by client")
}

// PresignPut returns a presigned PUT URL for the given key if supported.
func (b *bucketAdapter) PresignPut(ctx context.Context, key string, expires time.Duration) (string, error) {
	if c, ok := b.client.(*s3.Client); ok {
		presigner := s3.NewPresignClient(c)
		resp, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{Bucket: &b.bucket, Key: &key}, func(po *s3.PresignOptions) {
			if expires > 0 {
				po.Expires = expires
			}
		})
		return handlePresignResponse(resp, err)
	}
	return "", errors.New("presign not supported by client")
}
