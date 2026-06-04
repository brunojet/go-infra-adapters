// Package s3 exposes a small facade for configuring and constructing
// the internal S3 storage adapter used by callers.
package s3

import (
	internal "github.com/brunojet/go-infra-adapters/v4/internal/storage/aws/s3"
	storagecontracts "github.com/brunojet/go-infra-adapters/v4/pkg/storage/contracts"
)

// Option configures the S3 storage adapter. Use With* functions to create options.
type Option = internal.Option

// ClientAPI is the interface for a custom AWS S3 client, useful for testing or local mocks.
type ClientAPI = internal.S3API

// WithRegion configures the AWS region for the S3 client.
// Complexity: O(n) where n = len(region). Memory: ~16 bytes + string.
func WithRegion(region string) Option { return internal.WithRegion(region) }

// WithEndpoint configures a custom S3 endpoint (useful for local testing).
// Complexity: O(n) where n = len(endpoint). Memory: ~16 bytes + string.
func WithEndpoint(endpoint string) Option { return internal.WithEndpoint(endpoint) }

// WithClient configures a custom S3 client implementation.
func WithClient(client ClientAPI) Option { return internal.WithClient(client) }

// WithTransferManagerConcurrency sets the concurrency level for multipart operations.
// Default: 1 (sequential). Higher values enable parallel uploads/downloads.
// Recommended: 1 for Lambda (avoids memory overhead), 4-8 for servers.
func WithTransferManagerConcurrency(concurrency int) Option {
	return internal.WithTransferManagerConcurrency(concurrency)
}

// WithTransferManagerPartSize sets the size of each part in multipart uploads.
// Default: 5MB. Must be >= 5MB for S3.
func WithTransferManagerPartSize(bytes int64) Option {
	return internal.WithTransferManagerPartSize(bytes)
}

// WithTransferManagerThreshold sets the size threshold for using multipart uploads.
// Default: 10MB. Files smaller than this use single PutObject.
func WithTransferManagerThreshold(bytes int64) Option {
	return internal.WithTransferManagerThreshold(bytes)
}

// NewStorageAPI constructs an S3-backed StorageAPI using the provided options.
// Uses transfer manager by default with Concurrency=1 for optimal Lambda performance.
// Complexity: O(1). Memory: ~1-2 KB for client + state. Callers should reuse across requests.
func NewStorageAPI(opts ...Option) (storagecontracts.StorageAPI, error) {
	return internal.NewStorageAPI(opts...)
}
