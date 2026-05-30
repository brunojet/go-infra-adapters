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
	client   S3API //nolint:unused // reserved for injection in tests/extensions
	region   string
	endpoint string
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
