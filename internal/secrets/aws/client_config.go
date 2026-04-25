package aws

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Option func(*adapterConfig)

type SecretsManagerAPI interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

type adapterConfig struct {
	client   SecretsManagerAPI
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

func newSecretClient(cfg *adapterConfig) (*secretsmanager.Client, error) {
	var loadOpts []func(*awsconfig.LoadOptions) error
	if cfg.region != "" {
		loadOpts = append(loadOpts, awsconfig.WithRegion(cfg.region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, err
	}
	client := secretsmanager.NewFromConfig(awsCfg, func(o *secretsmanager.Options) {
		if cfg.endpoint != "" {
			o.BaseEndpoint = &cfg.endpoint
		}
	})
	return client, nil
}
