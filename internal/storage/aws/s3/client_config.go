package s3

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Option func(*adapterConfig)

// S3API mirrors the subset of s3.Client methods used by the adapter.
type S3API interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

type adapterConfig struct {
	client                      S3API                              //nolint:unused // reserved for injection in tests/extensions
	region                      string
	endpoint                    string
	transferManagerConcurrency  int
	transferManagerPartSize     int64
	transferManagerThreshold    int64
	disableTransferManager      bool
}

func WithRegion(region string) Option {
	return func(cfg *adapterConfig) {
		cfg.region = region
	}
}

func WithEndpoint(endpoint string) Option {
	return func(cfg *adapterConfig) {
		cfg.endpoint = endpoint
	}
}

// WithClient injects a custom S3 client (useful for testing or local mocks).
func WithClient(client S3API) Option {
	return func(cfg *adapterConfig) {
		cfg.client = client
	}
}

// WithTransferManagerConcurrency sets the concurrency level for multipart operations.
// Default: 1 (sequential). Higher values enable parallel uploads/downloads.
// Recommended: 1 for Lambda (avoids memory overhead), 4-8 for servers.
func WithTransferManagerConcurrency(concurrency int) Option {
	return func(cfg *adapterConfig) {
		if concurrency > 0 {
			cfg.transferManagerConcurrency = concurrency
		}
	}
}

// WithTransferManagerPartSize sets the size of each part in multipart uploads.
// Default: 5MB. Must be >= 5MB for S3.
func WithTransferManagerPartSize(bytes int64) Option {
	return func(cfg *adapterConfig) {
		if bytes >= 5*1024*1024 { // 5MB minimum
			cfg.transferManagerPartSize = bytes
		}
	}
}

// WithTransferManagerThreshold sets the size threshold for using multipart uploads.
// Default: 10MB. Files smaller than this use single PutObject.
func WithTransferManagerThreshold(bytes int64) Option {
	return func(cfg *adapterConfig) {
		if bytes > 0 {
			cfg.transferManagerThreshold = bytes
		}
	}
}

// WithoutTransferManager disables transfer manager and uses raw S3 API.
// Useful for small files or backward compatibility.
func WithoutTransferManager() Option {
	return func(cfg *adapterConfig) {
		cfg.disableTransferManager = true
	}
}

func newConfig(opts ...Option) *adapterConfig {
	cfg := &adapterConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// s3AwsLoadDefaultConfig wraps awsconfig.LoadDefaultConfig; replaceable in tests.
var s3AwsLoadDefaultConfig = awsconfig.LoadDefaultConfig

func newS3Client(cfg *adapterConfig) (S3API, error) {
	if cfg.client != nil {
		return cfg.client, nil
	}
	var loadOpts []func(*awsconfig.LoadOptions) error
	if cfg.region != "" {
		loadOpts = append(loadOpts, awsconfig.WithRegion(cfg.region))
	}
	awsCfg, err := s3AwsLoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(awsCfg), nil
}
