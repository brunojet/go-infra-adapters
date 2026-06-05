package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	"github.com/brunojet/go-infra-adapters/v4/pkg/storage/contracts"
)

const (
	// Lock file suffix appended to key
	lockSuffix = ".lock"

	// Lock metadata keys
	lockExpiresKey = "lock-expires"
	lockCreatedKey = "lock-created"
	etagKey        = "etag"

	// S3 conditional operation wildcards
	ifNoneMatchWildcard = "*"

	// AWS error codes
	preconditionFailedCode = "PreconditionFailed"

	// Backoff configuration for lock wait
	initialBackoff    = 100 * time.Millisecond
	maxBackoff        = 2 * time.Second
	backoffMultiplier = 2
)

var (
	// Error messages for validation
	ErrBucketNameRequired  = errors.New("bucket name required")
	ErrNilObject           = errors.New("nil object")
	ErrObjectKeyRequired   = errors.New("object key required")
	ErrNilObjectInfo       = errors.New("nil objectInfo")
	ErrFailedToAcquireLock = errors.New("failed to acquire lock")
)

type S3Client struct {
	client          S3API
	transferManager TransferManagerAPI
}

// NewStorageAPI constructs an S3-backed StorageAPI using the provided options.
// Transfer manager is always used for automatic retry and multipart support.
func NewStorageAPI(opts ...Option) (contracts.StorageAPI, error) {
	cfg := newConfig(opts...)
	client, err := newS3Client(cfg)
	if err != nil {
		return nil, err
	}
	transferMgr, err := newTransferManager(client, cfg)
	if err != nil {
		return nil, err
	}
	return &S3Client{
		client:          client,
		transferManager: transferMgr,
	}, nil
}

func (c *S3Client) NewBucket(name string) (contracts.BucketAdapter, error) {
	if name == "" {
		return nil, ErrBucketNameRequired
	}
	return &bucketAdapter{
		client:          c.client,
		transferManager: c.transferManager,
		bucket:          name,
	}, nil
}

type bucketAdapter struct {
	client          S3API
	transferManager TransferManagerAPI
	bucket          string
}

func (b *bucketAdapter) BucketName() string { return b.bucket }

// deleteObjectSafe deletes an object, ignoring NoSuchKey errors (idempotent).
// If eTag is provided, uses conditional delete (IfMatch) for atomicity.
func (b *bucketAdapter) deleteObjectSafe(ctx context.Context, key, eTag string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	}

	if eTag != "" {
		input.IfMatch = aws.String(eTag)
	}

	_, err := b.s3api.DeleteObject(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey" {
			return nil // Object not found is OK (idempotent)
		}
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

func (b *bucketAdapter) GetObject(ctx context.Context, key string, obj *contracts.BucketObject) error {
	if obj == nil {
		return ErrNilObject
	}
	getResult, err := b.transferManager.GetObject(ctx, &transfermanager.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	fillObjectInfoUnsafe(key, getResult.ContentLength, getResult.ContentType, getResult.ETag, &obj.Info)
	obj.Body = io.NopCloser(getResult.Body)
	return nil
}

func (b *bucketAdapter) PutObject(ctx context.Context, obj *contracts.BucketObject) error {
	if obj == nil {
		return ErrNilObject
	}
	key := obj.Info.Key
	if key == "" {
		return ErrObjectKeyRequired
	}
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
	if obj.Body != nil {
		_ = obj.Body.Close()
	}
	return err
}

func (b *bucketAdapter) HeadObject(ctx context.Context, key string, objInfo *contracts.ObjectInfo) error {
	if objInfo == nil {
		return ErrNilObjectInfo
	}
	out, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &b.bucket, Key: &key})
	if err != nil {
		return err
	}
	fillObjectInfoUnsafe(key, out.ContentLength, out.ContentType, out.ETag, objInfo)
	return nil
}

func (b *bucketAdapter) GetLock(ctx context.Context, key string, lockTTL time.Duration) error {
	lockKey := key + lockSuffix
	lockExpires := time.Now().Add(lockTTL).Format(time.RFC3339)
	_, err := b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(lockKey),
		Body:   bytes.NewReader([]byte("")),
		Metadata: map[string]string{
			lockExpiresKey: lockExpires,
			lockCreatedKey: time.Now().Format(time.RFC3339),
		},
		IfNoneMatch: aws.String(ifNoneMatchWildcard),
	})
	if err != nil {
		if !isIfNoneMatchError(err) {
			return fmt.Errorf("%w: %w", ErrFailedToAcquireLock, err)
		}
		objInfo := &contracts.ObjectInfo{}
		if headErr := b.HeadObject(ctx, lockKey, objInfo); headErr == nil {
			if lockIsExpired(objInfo.Metadata[lockExpiresKey]) {
				// Lock expired - delete with ETag for consistency
				_ = b.deleteObjectSafe(ctx, lockKey, objInfo.Metadata[etagKey])
			}
		}
		return &lockExistsError{key: strings.TrimSuffix(lockKey, lockSuffix)}
	}
	return nil
}

func (b *bucketAdapter) GetLockWait(ctx context.Context, key string, lockTTL, waitTimeout time.Duration) error {
	if waitTimeout < lockTTL {
		return fmt.Errorf("waitTimeout (%v) must be >= lockTTL (%v)", waitTimeout, lockTTL)
	}
	deadline := time.Now().Add(waitTimeout)
	backoff := initialBackoff
	for {
		err := b.GetLock(ctx, key, lockTTL)
		if err == nil {
			return nil
		}
		var lockErr *lockExistsError
		if !errors.As(err, &lockErr) {
			return err
		}
		if time.Now().Add(backoff).After(deadline) {
			return fmt.Errorf("timeout waiting for lock after %v", waitTimeout)
		}
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
		backoff *= backoffMultiplier
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (b *bucketAdapter) ReleaseLock(ctx context.Context, key string) error {
	lockKey := key + lockSuffix
	return b.deleteObjectSafe(ctx, lockKey, "")
}
