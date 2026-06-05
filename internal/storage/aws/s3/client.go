package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

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

	// Store S3 API for HeadObject operations
	if cfg.s3api == nil {
		cfg.s3api = client
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
	s3api := client // Default to client if config not available
	if c.config != nil && c.config.s3api != nil {
		s3api = c.config.s3api
	}

	return &bucketAdapter{
		client:          client,
		bucket:          name,
		transferManager: c.transferManager,
		s3api:           s3api,
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
	s3api           S3API
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

	// Use raw S3 API for HEAD operations
	out, err := b.s3api.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &b.bucket, Key: &key})
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

// lockExistsError indicates that a lock already exists for a key.
type lockExistsError struct {
	key string
}

func (e *lockExistsError) Error() string {
	return fmt.Sprintf("lock already exists for key %s", e.key)
}

func (b *bucketAdapter) GetLock(ctx context.Context, key string, lockTTL time.Duration) error {
	lockKey := key + ".lock"
	lockExpires := time.Now().Add(lockTTL).Format(time.RFC3339)

	_, err := b.s3api.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(lockKey),
		Body:   bytes.NewReader([]byte("")),
		Metadata: map[string]string{
			"lock-expires": lockExpires,
			"lock-created": time.Now().Format(time.RFC3339),
		},
		IfNoneMatch: aws.String("*"),
	})

	if err != nil {
		if !isIfNoneMatchError(err) {
			return fmt.Errorf("failed to acquire lock: %w", err)
		}

		// Lock exists. Check if expired and clean up if necessary.
		objInfo := &contracts.ObjectInfo{}
		if headErr := b.HeadObject(ctx, lockKey, objInfo); headErr == nil {
			if lockIsExpired(objInfo.Metadata["lock-expires"]) {
				// Lock expired - delete it and return error so caller can retry
				_, _ = b.s3api.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: aws.String(b.bucket),
					Key:    aws.String(lockKey),
				})
			}
		}

		// Lock exists (or was expired and cleaned)
		return &lockExistsError{key: key}
	}

	return nil
}

func lockIsExpired(expiredStr string) bool {
	if expiredStr == "" {
		return false
	}
	expiredTime, err := time.Parse(time.RFC3339, expiredStr)
	if err != nil {
		return false
	}
	return time.Now().After(expiredTime)
}

func (b *bucketAdapter) GetLockWait(ctx context.Context, key string, lockTTL, waitTimeout time.Duration) error {
	deadline := time.Now().Add(waitTimeout)
	backoff := 100 * time.Millisecond
	maxBackoff := 2 * time.Second

	for {
		err := b.GetLock(ctx, key, lockTTL)
		if err == nil {
			return nil
		}

		// Only retry on lock exists error, propagate other errors
		var lockErr *lockExistsError
		if !errors.As(err, &lockErr) {
			return err
		}

		if time.Now().Add(backoff).After(deadline) {
			return fmt.Errorf("timeout waiting for lock after %v", waitTimeout)
		}

		select {
		case <-time.After(backoff):
			// Continue loop
		case <-ctx.Done():
			return ctx.Err()
		}

		backoff = backoff * 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (b *bucketAdapter) ReleaseLock(ctx context.Context, key string) error {
	lockKey := key + ".lock"
	objInfo := &contracts.ObjectInfo{}
	err := b.HeadObject(ctx, lockKey, objInfo)

	eTag := ""
	if err == nil && len(objInfo.Metadata) > 0 {
		eTag = objInfo.Metadata["etag"]
	}

	deleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(lockKey),
	}

	if eTag != "" {
		deleteInput.IfMatch = aws.String(eTag)
	}

	_, err = b.s3api.DeleteObject(ctx, deleteInput)

	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey" {
			return nil
		}
		return fmt.Errorf("failed to release lock: %w", err)
	}

	return nil
}

func isIfNoneMatchError(err error) bool {
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == "PreconditionFailed"
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

	return transfermanager.New(s3client, func(opts *transfermanager.Options) {
		opts.Concurrency = concurrency
		opts.PartSizeBytes = partSize
		opts.MultipartUploadThreshold = threshold
	})
}
