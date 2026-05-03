package aws

import (
	"github.com/brunojet/go-infra-adapters/internal/secrets/aws"
	"github.com/brunojet/go-infra-adapters/pkg/secrets/contracts"
)

func WithRegion(region string) aws.Option {
	return aws.WithRegion(region)
}

func WithEndpoint(endpoint string) aws.Option {
	return aws.WithEndpoint(endpoint)
}

func NewSecretClient(opts ...aws.Option) (contracts.SecretClientAPI, error) {
	return aws.NewSecretClient(opts...)
}
