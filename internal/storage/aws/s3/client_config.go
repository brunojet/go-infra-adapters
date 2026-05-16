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

func newConfig(opts ...Option) *adapterConfig {
	cfg := &adapterConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func newS3Client(cfg *adapterConfig) (*s3.Client, error) {
	var loadOpts []func(*awsconfig.LoadOptions) error
	if cfg.region != "" {
		loadOpts = append(loadOpts, awsconfig.WithRegion(cfg.region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(awsCfg)
	return client, nil
}
