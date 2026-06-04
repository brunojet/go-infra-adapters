package s3

import (
	"bytes"
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
	transferManager *transfermanager.Client
	config          *adapterConfig
	mu              sync.Mutex
}

// NewStorageAPI constructs an S3-backed StorageAPI using the provided options.
func NewStorageAPI(opts ...Option) (contracts.StorageAPI, error) {
	cfg := newConfig(opts...)
	client, err := newS3Client(cfg)
	if err != nil {
		return nil, err
	}

	// Initialize transfer manager with defaults
	var transferMgr *transfermanager.Client
	if !cfg.disableTransferManager {
		s3client, ok := client.(*s3.Client)
		if ok {
			transferMgr = initTransferManager(s3client, cfg)
		}
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
		client:             client,
		bucket:             name,
		transferManager:    c.transferManager,
		disableTransferMgr: c.config != nil && c.config.disableTransferManager,
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
	client             S3API
	bucket             string
	transferManager    *transfermanager.Client
	disableTransferMgr bool
}

func (b *bucketAdapter) BucketName() string { return b.bucket }

func (b *bucketAdapter) GetObject(ctx context.Context, key string, obj *contracts.BucketObject) error {
	if obj == nil {
		return errors.New("nil object")
	}

	var body io.ReadCloser
	var size int64
	var contentType string
	var meta map[string]string

	// Use transfer manager if available for better retry and streaming
	if b.transferManager != nil && !b.disableTransferMgr {
		getResult, err := b.transferManager.GetObject(ctx, &transfermanager.GetObjectInput{
			Bucket: aws.String(b.bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return err
		}

		body = io.NopCloser(getResult.Body)
		if getResult.ContentLength != nil {
			size = *getResult.ContentLength
		}
		if getResult.ContentType != nil {
			contentType = *getResult.ContentType
		}
		meta = make(map[string]string)
		if getResult.ETag != nil {
			meta["etag"] = *getResult.ETag
		}
	} else {
		// Fallback to raw S3 API
		out, err := b.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &b.bucket, Key: &key})
		if err != nil {
			return err
		}

		if out.Body == nil {
			body = io.NopCloser(bytes.NewReader(nil))
		} else {
			body = out.Body
		}

		if out.ContentLength != nil {
			size = *out.ContentLength
		}
		if out.ContentType != nil {
			contentType = *out.ContentType
		}

		meta = map[string]string{}
		if out.ETag != nil {
			meta["etag"] = *out.ETag
		}
	}

	obj.Info = contracts.ObjectInfo{Key: key, Size: size, ContentType: contentType, Metadata: meta}
	obj.Body = body
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

	var err error

	// Use transfer manager if available for multipart upload with retry
	if b.transferManager != nil && !b.disableTransferMgr {
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
		_, err = b.transferManager.UploadObject(ctx, input)
	} else {
		// Fallback to raw S3 API
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
		_, err = b.client.PutObject(ctx, input)
	}

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

// initTransferManager initializes transfer manager with defaults optimized for Lambda
func initTransferManager(s3client *s3.Client, cfg *adapterConfig) *transfermanager.Client {
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
