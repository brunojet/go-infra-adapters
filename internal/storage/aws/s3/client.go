package s3

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/brunojet/go-infra-adapters/v4/pkg/storage/contracts"
)

type S3Client struct {
	client          S3API
	transferManager TransferManagerAPI
	config          *adapterConfig
	mu              sync.Mutex
}

// NewStorageAPI constructs an S3-backed StorageAPI using the provided options.
// Transfer manager is always used for automatic retry and multipart support.
func NewStorageAPI(opts ...Option) (contracts.StorageAPI, error) {
	cfg := newConfig(opts...)
	client, err := newS3Client(cfg)
	if err != nil {
		return nil, err
	}

	// Use injected transfer manager or initialize from S3 client
	var transferMgr TransferManagerAPI
	if cfg.transferManager != nil {
		// Transfer manager was injected (e.g., for testing)
		transferMgr = cfg.transferManager
	} else {
		// Initialize from S3 client (required, never nil)
		s3client, ok := client.(*s3.Client)
		if !ok {
			return nil, errors.New("S3 client must be *s3.Client to initialize transfer manager")
		}
		transferMgr = initTransferManager(s3client, cfg)
	}

	return &S3Client{
		client:          client,
		transferManager: transferMgr,
		config:          cfg,
	}, nil
}

func (c *S3Client) NewBucket(name string) (contracts.BucketAdapter, error) {
	if name == "" {
		return nil, errors.New("bucket name required")
	}
	client, err := c.defaultClient()
	if err != nil {
		return nil, err
	}
	return &bucketAdapter{
		client:          client,
		bucket:          name,
		transferManager: c.transferManager,
	}, nil
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
	client          S3API
	bucket          string
	transferManager TransferManagerAPI
}

func (b *bucketAdapter) BucketName() string { return b.bucket }

func (b *bucketAdapter) GetObject(ctx context.Context, key string, obj *contracts.BucketObject) error {
	if obj == nil {
		return errors.New("nil object")
	}

	// Transfer manager always used for automatic retry and streaming
	getResult, err := b.transferManager.GetObject(ctx, &transfermanager.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	meta := make(map[string]string)
	if getResult.ETag != nil {
		meta["etag"] = *getResult.ETag
	}

	var size int64
	if getResult.ContentLength != nil {
		size = *getResult.ContentLength
	}

	var contentType string
	if getResult.ContentType != nil {
		contentType = *getResult.ContentType
	}

	obj.Info = contracts.ObjectInfo{Key: key, Size: size, ContentType: contentType, Metadata: meta}
	obj.Body = io.NopCloser(getResult.Body)
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

	// Transfer manager always used for multipart upload with automatic retry
	input := &transfermanager.UploadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
		Body:   obj.Body,
	}
	if obj.Info.ContentType != "" {
		input.ContentType = aws.String(obj.Info.ContentType)
	}
	if obj.Info.Metadata != nil {
		input.Metadata = obj.Info.Metadata
	}

	_, err := b.transferManager.UploadObject(ctx, input)

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

	// Use transfer manager API which includes HeadObject via S3 client
	out, err := b.transferManager.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &b.bucket, Key: &key})
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

// transferManagerWithHeadObject wraps transfer manager + S3 client to provide HeadObject
type transferManagerWithHeadObject struct {
	tm  *transfermanager.Client
	s3c S3API
}

func (t *transferManagerWithHeadObject) GetObject(ctx context.Context, input *transfermanager.GetObjectInput, opts ...func(*transfermanager.Options)) (*transfermanager.GetObjectOutput, error) {
	return t.tm.GetObject(ctx, input, opts...)
}

func (t *transferManagerWithHeadObject) UploadObject(ctx context.Context, input *transfermanager.UploadObjectInput, opts ...func(*transfermanager.Options)) (*transfermanager.UploadObjectOutput, error) {
	return t.tm.UploadObject(ctx, input, opts...)
}

func (t *transferManagerWithHeadObject) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return t.s3c.HeadObject(ctx, params, optFns...)
}

// initTransferManager initializes transfer manager with defaults optimized for Lambda
func initTransferManager(s3client *s3.Client, cfg *adapterConfig) TransferManagerAPI {
	// Set defaults: these are tuned for Lambda (low memory, sequential processing)
	concurrency := 1                     // Lambda: sequential only
	partSize := int64(5 * 1024 * 1024)   // 5MB per part
	threshold := int64(10 * 1024 * 1024) // Use multipart for >10MB

	// Allow overrides via config
	if cfg.transferManagerConcurrency > 0 {
		concurrency = cfg.transferManagerConcurrency
	}
	if cfg.transferManagerPartSize > 0 {
		partSize = cfg.transferManagerPartSize
	}
	if cfg.transferManagerThreshold > 0 {
		threshold = cfg.transferManagerThreshold
	}

	tm := transfermanager.New(s3client, func(opts *transfermanager.Options) {
		opts.Concurrency = concurrency
		opts.PartSizeBytes = partSize
		opts.MultipartUploadThreshold = threshold
	})

	return &transferManagerWithHeadObject{tm: tm, s3c: s3client}
}
