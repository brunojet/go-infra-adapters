package aws

import (
	"context"
	"log/slog"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	awsretryer "github.com/brunojet/go-infra-adapters/v3/internal/retryer/aws"
	"github.com/brunojet/go-infra-adapters/v3/pkg/retry"
)

type SecretsOption func(*SecretsConfig)

type SecretsManagerClient interface {
	GetSecretValue(ctx context.Context, input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	PutSecretValue(ctx context.Context, input *secretsmanager.PutSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error)
	DescribeSecret(ctx context.Context, input *secretsmanager.DescribeSecretInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error)
	UpdateSecretVersionStage(ctx context.Context, input *secretsmanager.UpdateSecretVersionStageInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretVersionStageOutput, error)
}

type SecretsConfig struct {
	client        SecretsManagerClient
	logger        *slog.Logger
	region        string
	endpoint      string
	retryStrategy retry.Strategy
}

func WithClient(client SecretsManagerClient) SecretsOption {
	return func(cfg *SecretsConfig) {
		if client == nil {
			panic("nil SecretsManagerClient provided to WithClient option")
		}
		cfg.client = client
	}
}

func WithRegion(region string) SecretsOption {
	return func(cfg *SecretsConfig) {
		if region == "" {
			panic("empty region provided to WithRegion option")
		}
		cfg.region = region
	}
}

func WithEndpoint(endpoint string) SecretsOption {
	return func(cfg *SecretsConfig) {
		cfg.endpoint = endpoint
	}
}

func WithLogger(logger *slog.Logger) SecretsOption {
	return func(cfg *SecretsConfig) {
		if logger != nil {
			cfg.logger = logger
		}
	}
}

// WithRetryStrategy sets the retry strategy for Secrets Manager API calls.
func WithRetryStrategy(strategy retry.Strategy) SecretsOption {
	return func(cfg *SecretsConfig) {
		if strategy == nil {
			panic("retry strategy cannot be nil")
		}
		cfg.retryStrategy = strategy
	}
}

// awsLoadDefaultConfig wraps awsconfig.LoadDefaultConfig; replaceable in tests.
var awsLoadDefaultConfig = awsconfig.LoadDefaultConfig

// smLoadConfig is the AWS config loader; overridable in tests for error injection.
var smLoadConfig = func(cfg *SecretsConfig) (SecretsManagerClient, error) {
	var loadOpts []func(*awsconfig.LoadOptions) error
	if cfg.region != "" {
		loadOpts = append(loadOpts, awsconfig.WithRegion(cfg.region))
	}
	awsCfg, err := awsLoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, err
	}
	sdkRetryer := awsretryer.NewSDKRetryer(cfg.retryStrategy, cfg.logger)
	cfg.logger.Info("SecretsManager client configured with retry strategy",
		"maxAttempts", cfg.retryStrategy.MaxAttempts())

	return secretsmanager.NewFromConfig(awsCfg, func(o *secretsmanager.Options) {
		if cfg.endpoint != "" {
			o.BaseEndpoint = &cfg.endpoint
		}
		o.Retryer = sdkRetryer
	}), nil
}

// newConfig applies options and returns the resolved config.
// Returns an error if no client was injected and the AWS SDK cannot be initialized.
func newConfig(opts ...SecretsOption) (*SecretsConfig, error) {
	cfg := &SecretsConfig{
		logger:        slog.Default(),
		retryStrategy: retry.NewStandard(), // Default: 3 attempts with exponential backoff
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	if cfg.client == nil {
		client, err := smLoadConfig(cfg)
		if err != nil {
			return nil, err
		}
		cfg.client = client
	}
	return cfg, nil
}
