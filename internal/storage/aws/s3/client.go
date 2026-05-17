package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/brunojet/go-infra-adapters/pkg/storage/contracts"
)

type S3Client struct {
	client S3API
	mu     sync.Mutex
}

// NewS3Client constructs a provider-side client. The returned type is the
// opaque client API used by callers when they want to provide their own
// preconfigured client (mirrors other adapters in the repo).
func NewS3Client(opts ...Option) (contracts.StorageClientAPI, error) {
	cfg := newConfig(opts...)
	client, err := newS3Client(cfg)
	if err != nil {
		return nil, err
	}
	return &S3Client{client: client}, nil
}

func (c *S3Client) NewBucketWithClient(name string, client contracts.StorageClientAPI) (contracts.BucketAdapter, error) {
	if name == "" {
		return nil, errors.New("bucket name required")
	}
	if client, ok := client.(S3API); ok {
		return &bucketAdapter{client: client, bucket: name}, nil
	}
	return nil, errors.New("invalid client type provided")
}

func (c *S3Client) NewBucket(name string) (contracts.BucketAdapter, error) {
	client, err := c.defaultClient()
	if err != nil {
		return nil, err
	}
	return c.NewBucketWithClient(name, client)
}

func (c *S3Client) defaultClient() (S3API, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		return c.client, nil
	}
	cfg := newConfig()
	client, err := newS3Client(cfg)
	if err != nil {
		return nil, err
	}
	c.client = client
	return c.client, nil
}

type bucketAdapter struct {
	client S3API
	bucket string
}

func (b *bucketAdapter) BucketName() string { return b.bucket }

func (b *bucketAdapter) GetObject(ctx context.Context, key string, obj *contracts.BucketObject) error {
	if obj == nil {
		return errors.New("nil object")
	}
	out, err := b.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &b.bucket, Key: &key})
	if err != nil {
		return err
	}
	// Build metadata/info to return alongside the stream.
	meta := map[string]string{}
	if out.ETag != nil {
		meta["etag"] = *out.ETag
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	var contentType string
	if out.ContentType != nil {
		contentType = *out.ContentType
	}
	obj.Info = contracts.ObjectInfo{Key: key, Size: size, ContentType: contentType, Metadata: meta}

	// If the provider returned a nil body, supply an empty reader.
	if out.Body == nil {
		obj.Body = io.NopCloser(bytes.NewReader(nil))
	} else {
		obj.Body = out.Body
	}
	return nil
}

func (b *bucketAdapter) PutObject(ctx context.Context, obj *contracts.BucketObject) error {
	if obj == nil {
		return errors.New("nil object")
	}
	key := obj.Info.Key
	if key == "" {
		return errors.New("object key required")
	}
	input := &s3.PutObjectInput{Bucket: &b.bucket, Key: &key, Body: obj.Body}
	if obj.Info.Size > 0 {
		input.ContentLength = aws.Int64(obj.Info.Size)
	}
	if obj.Info.ContentType != "" {
		input.ContentType = aws.String(obj.Info.ContentType)
	}
	if obj.Info.Metadata != nil {
		input.Metadata = obj.Info.Metadata
	}
	_, err := b.client.PutObject(ctx, input)
	// Close the provided body if present to avoid leaking resources.
	if obj.Body != nil {
		_ = obj.Body.Close()
	}
	return err
}

func (b *bucketAdapter) HeadObject(ctx context.Context, key string, objInfo *contracts.ObjectInfo) error {
	if objInfo == nil {
		return errors.New("nil objectInfo")
	}
	out, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &b.bucket, Key: &key})
	if err != nil {
		return err
	}
	meta := map[string]string{}
	if out.ETag != nil {
		meta["etag"] = *out.ETag
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	var contentType string
	if out.ContentType != nil {
		contentType = *out.ContentType
	}
	*objInfo = contracts.ObjectInfo{Key: key, Size: size, ContentType: contentType, Metadata: meta}
	return nil
}
