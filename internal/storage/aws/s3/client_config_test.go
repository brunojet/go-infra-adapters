package s3

import (
	"context"
	"errors"
	"testing"

	goaws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/require"
)

func TestNewConfigOptions(t *testing.T) {
	cfg := newConfig(WithRegion("r"), WithEndpoint("e"))
	if cfg.region != "r" {
		t.Fatalf("expected region r, got %s", cfg.region)
	}
	if cfg.endpoint != "e" {
		t.Fatalf("expected endpoint e, got %s", cfg.endpoint)
	}
	// newS3Client uses aws config and may fail in tests depending on environment; we only validate option wiring here
	_ = cfg
}

func TestNewS3Client_Wiring(t *testing.T) {
	cfg := newConfig(WithRegion("us-east-1"), WithEndpoint("http://localhost"))
	c, err := newS3Client(cfg)
	require.NoError(t, err)
	if c == nil {
		t.Fatalf("expected s3 client")
	}
}

// TestNewS3Client_AwsLoadError verifies the error-return path when the AWS SDK
// config loader fails.
func TestNewS3Client_AwsLoadError(t *testing.T) {
	orig := s3AwsLoadDefaultConfig
	s3AwsLoadDefaultConfig = func(_ context.Context, _ ...func(*awsconfig.LoadOptions) error) (goaws.Config, error) {
		return goaws.Config{}, errors.New("injected s3 aws config error")
	}
	defer func() { s3AwsLoadDefaultConfig = orig }()

	cfg := newConfig(WithRegion("us-east-1"))
	_, err := newS3Client(cfg)
	if err == nil || err.Error() != "injected s3 aws config error" {
		t.Fatalf("expected injected error, got %v", err)
	}
}
