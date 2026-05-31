package aws

import (
	"context"
	"log/slog"

	goaws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"

	"github.com/brunojet/go-infra-adapters/v3/pkg/retry"
)

// CloudFrontClient abstracts the CloudFront SDK operations.
type CloudFrontClient interface {
	ListPublicKeys(ctx context.Context, input *cloudfront.ListPublicKeysInput, opts ...func(*cloudfront.Options)) (*cloudfront.ListPublicKeysOutput, error)
	ListKeyGroups(ctx context.Context, input *cloudfront.ListKeyGroupsInput, opts ...func(*cloudfront.Options)) (*cloudfront.ListKeyGroupsOutput, error)
	CreatePublicKey(ctx context.Context, input *cloudfront.CreatePublicKeyInput, opts ...func(*cloudfront.Options)) (*cloudfront.CreatePublicKeyOutput, error)
	GetPublicKey(ctx context.Context, input *cloudfront.GetPublicKeyInput, opts ...func(*cloudfront.Options)) (*cloudfront.GetPublicKeyOutput, error)
	DeletePublicKey(ctx context.Context, input *cloudfront.DeletePublicKeyInput, opts ...func(*cloudfront.Options)) (*cloudfront.DeletePublicKeyOutput, error)
	GetKeyGroup(ctx context.Context, input *cloudfront.GetKeyGroupInput, opts ...func(*cloudfront.Options)) (*cloudfront.GetKeyGroupOutput, error)
	CreateKeyGroup(ctx context.Context, input *cloudfront.CreateKeyGroupInput, opts ...func(*cloudfront.Options)) (*cloudfront.CreateKeyGroupOutput, error)
	UpdateKeyGroup(ctx context.Context, input *cloudfront.UpdateKeyGroupInput, opts ...func(*cloudfront.Options)) (*cloudfront.UpdateKeyGroupOutput, error)
}

// CdnOption configures a CloudFrontDistribution.
type CdnOption func(*cdnConfig)

// cdnConfig implements KeyDistribution using the AWS CloudFront API.
type cdnConfig struct {
	client        CloudFrontClient
	maxKeys       int
	concurrency   int
	logger        *slog.Logger
	retryStrategy retry.Strategy
}

// WithMaxKeys sets the maximum number of keys to keep in a key group.
func WithMaxKeys(n int) CdnOption {
	return func(d *cdnConfig) {
		if n <= 0 {
			panic("maxKeys must be > 0")
		}
		d.maxKeys = n
	}
}

// WithConcurrency sets the maximum number of concurrent API calls.
func WithConcurrency(n int) CdnOption {
	return func(d *cdnConfig) {
		if n <= 0 {
			panic("concurrency must be > 0")
		}
		d.concurrency = n
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) CdnOption {
	return func(d *cdnConfig) {
		if logger == nil {
			panic("logger cannot be nil")
		}
		d.logger = logger
	}
}

func WithClient(client CloudFrontClient) CdnOption {
	return func(d *cdnConfig) {
		if client == nil {
			panic("client cannot be nil")
		}
		d.client = client
	}
}

// WithRetryStrategy sets the retry strategy for CloudFront API calls.
func WithRetryStrategy(strategy retry.Strategy) CdnOption {
	return func(d *cdnConfig) {
		if strategy == nil {
			panic("retry strategy cannot be nil")
		}
		d.retryStrategy = strategy
	}
}

func newCdnConfig(opts ...CdnOption) *cdnConfig {
	d := &cdnConfig{
		maxKeys:       3,
		concurrency:   5,
		logger:        slog.Default(),
		retryStrategy: retry.NewStandard(), // Default: 3 attempts with exponential backoff
	}
	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}
	return d
}

// cdnAwsConfigLoader is the AWS config loader; overridable in tests for error injection.
var cdnAwsConfigLoader = func() (goaws.Config, error) {
	return awsconfig.LoadDefaultConfig(context.Background())
}

func newCdnClient(cfg *cdnConfig) CloudFrontClient {
	if cfg.client != nil {
		return cfg.client
	}
	awsCfg, err := cdnAwsConfigLoader()
	if err != nil {
		panic("failed to load AWS config: " + err.Error())
	}
	return cloudfront.NewFromConfig(awsCfg)
}
