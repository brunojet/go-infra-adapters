// Package contracts defines storage-related public contracts used by callers
// to interact with provider bucket adapters without depending on provider
// implementations.
package contracts

import (
	"context"
	"io"
	"time"
)

// ObjectInfo contains metadata about an object stored in a bucket.
type ObjectInfo struct {
	Key  string
	Size int64
	// ContentType is the MIME type associated with the object, if known.
	ContentType string
	Metadata    map[string]string
}

// Object is a small in-memory representation of an object retrieved from storage.
type Object struct {
	Data []byte
	Info ObjectInfo
}

// BucketObject represents an object transferred to/from a bucket along with
// its associated stream and metadata. For downloads `Body` is non-nil and the
// caller is responsible for closing it (use `Close()` helper). For uploads
// callers should provide a `Body` (an io.ReadCloser); the adapter may close
// the provided `Body` after use — callers must not rely on it remaining open.
type BucketObject struct {
	Info ObjectInfo
	Body io.ReadCloser
}

// Close closes the underlying body if present.
func (o *BucketObject) Close() error {
	if o == nil || o.Body == nil {
		return nil
	}
	return o.Body.Close()
}

// BucketAdapter represents an adapter bound to a specific bucket.
type BucketAdapter interface {
	BucketName() string

	// GetObject fills the provided `obj` with a streaming `Body` and
	// metadata. The caller must supply a non-nil `obj` and call `Close()` on
	// `obj` once finished reading. Returns an error if `obj` is nil.
	GetObject(ctx context.Context, key string, obj *BucketObject) error

	// PutObject accepts a `BucketObject` to allow supplying metadata and
	// content together. The adapter may close `obj.Body` after the upload;
	// callers should not assume the body remains open.
	PutObject(ctx context.Context, obj *BucketObject) error
	// HeadObject fills the provided `objInfo` with metadata about the object
	// identified by `key`. The caller must supply a non-nil `objInfo`. Returns
	// an error if `objInfo` is nil.
	HeadObject(ctx context.Context, key string, objInfo *ObjectInfo) error

	// GetLock attempts to acquire a distributed lock for the given file key.
	// Internally creates a lock file by appending ".lock" to the key.
	// lockTTL specifies how long the lock is valid. CRITICAL: lockTTL MUST be greater than
	// the maximum expected execution time to prevent race conditions. Returns nil if lock
	// is acquired, or an error if the lock already exists or S3 operation fails. Expired
	// locks are automatically cleaned up. The operation is atomic (uses conditional put).
	GetLock(ctx context.Context, key string, lockTTL time.Duration) error

	// GetLockWait attempts to acquire a distributed lock with automatic retry and backoff.
	// Internally creates a lock file by appending ".lock" to the key.
	// lockTTL specifies how long the lock is valid. waitTimeout specifies how long to wait
	// for the lock before returning an error. Uses exponential backoff starting at 100ms
	// (capped at 2s). CRITICAL: Same TTL constraint as GetLock - must exceed max execution
	// time. Returns nil if lock is acquired, or an error if timeout or S3 operation fails.
	GetLockWait(ctx context.Context, key string, lockTTL, waitTimeout time.Duration) error

	// ReleaseLock deletes the lock file for the given key (appends ".lock" internally).
	// This operation is idempotent—it returns nil even if the lock doesn't exist.
	// CRITICAL: Must be called via defer() to ensure cleanup even on panic or early return.
	// Uses conditional delete (IfMatch with ETag) to ensure atomic deletion, falling back
	// to unconditional delete if ETag unavailable.
	ReleaseLock(ctx context.Context, key string) error
}

// StorageAPI constructs per-bucket adapters.
type StorageAPI interface {
	NewBucket(name string) (BucketAdapter, error)
}
