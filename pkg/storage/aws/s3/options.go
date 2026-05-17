// Package s3 exposes a small facade for configuring and constructing
// the internal S3 storage adapter used by callers.
package s3

import (
	internal "github.com/brunojet/go-infra-adapters/internal/storage/aws/s3"
	storagecontracts "github.com/brunojet/go-infra-adapters/pkg/storage/contracts"
)

// WithRegion configures the AWS region for the S3 client.
func WithRegion(region string) internal.Option { return internal.WithRegion(region) }

// WithEndpoint configures a custom S3 endpoint (useful for local testing).
func WithEndpoint(endpoint string) internal.Option { return internal.WithEndpoint(endpoint) }

// NewS3Client constructs an S3 client using the provided options.
func NewS3Client(opts ...internal.Option) (storagecontracts.StorageClientAPI, error) {
	return internal.NewS3Client(opts...)
}
