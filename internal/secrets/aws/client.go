package aws

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/brunojet/go-infra-adapters/pkg/secrets/contracts"
)

type SecretsClient struct {
	client SecretsManagerAPI
	mu     sync.Mutex
}

func NewSecretClient(opts ...Option) (contracts.SecretClientAPI, error) {
	cfg := newConfig(opts...)
	return newSecretClient(cfg)
}

func (c *SecretsClient) NewSecretWithClient(name string, client contracts.SecretClientAPI) (contracts.SecretAdapter, error) {
	if name == "" {
		return nil, errors.New("secret name required")
	}
	if client, ok := client.(SecretsManagerAPI); ok {
		return &secretAdapter{client: client, name: name}, nil
	}
	return nil, errors.New("invalid client type provided")
}

func (c *SecretsClient) NewSecret(name string) (contracts.SecretAdapter, error) {
	client, err := c.defaultSecretAdapter()
	if err != nil {
		return nil, err
	}
	return c.NewSecretWithClient(name, client)
}

func (c *SecretsClient) defaultSecretAdapter() (SecretsManagerAPI, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		return c.client, nil
	}
	cfg := newConfig()
	client, err := newSecretClient(cfg)
	if err != nil {
		return nil, err
	}
	c.client = client
	return c.client, nil
}

type secretAdapter struct {
	client SecretsManagerAPI
	name   string
}

func (s *secretAdapter) Name() string { return s.name }

func (s *secretAdapter) GetCurrent(ctx context.Context) (*contracts.SecretValue, error) {
	out, err := s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &s.name})
	if err != nil {
		return nil, err
	}
	return convert(out), nil
}

func (s *secretAdapter) GetVersion(ctx context.Context, version string) (*contracts.SecretValue, error) {
	if version == "" {
		return nil, errors.New("version required")
	}
	out, err := s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:  &s.name,
		VersionId: &version,
	})
	if err != nil {
		return nil, err
	}
	return convert(out), nil
}

func fillMeta(meta map[string]string, key string, value *string) {
	if value != nil {
		meta[key] = *value
	}
}

func convert(in *secretsmanager.GetSecretValueOutput) *contracts.SecretValue {
	var data []byte
	if in.SecretString != nil {
		data = []byte(*in.SecretString)
	} else if in.SecretBinary != nil {
		data = in.SecretBinary
	}
	meta := map[string]string{}
	fillMeta(meta, "arn", in.ARN)
	fillMeta(meta, "versionId", in.VersionId)
	fillMeta(meta, "name", in.Name)
	if len(in.VersionStages) > 0 {
		var stages strings.Builder
		for i, s := range in.VersionStages {
			if i > 0 {
				stages.WriteString(",")
			}
			stages.WriteString(string(s))
		}
		meta["versionStages"] = stages.String()
	}
	return &contracts.SecretValue{Data: data, Metadata: meta}
}
