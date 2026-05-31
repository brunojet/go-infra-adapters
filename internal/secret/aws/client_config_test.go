package aws

import (
	"context"
	"errors"
	"testing"

	goaws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	mocksm "github.com/brunojet/go-infra-adapters/v3/internal/secret/aws/mock"
)

func TestNewConfigOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg, err := newConfig(WithClient(mocksm.NewMockSecretsManagerClient(ctrl)), WithRegion("r"), WithEndpoint("e"))
	require.NoError(t, err)
	if cfg.region != "r" {
		t.Fatalf("expected region r, got %s", cfg.region)
	}
	if cfg.endpoint != "e" {
		t.Fatalf("expected endpoint e, got %s", cfg.endpoint)
	}
}

// TestNewConfig_NilOption verifies that a nil option in the slice is skipped.
func TestNewConfig_NilOption(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg, err := newConfig(WithClient(mocksm.NewMockSecretsManagerClient(ctrl)), nil)
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

// TestNewSecretClient_Wiring verifies that newConfig succeeds in CI.
func TestNewSecretClient_Wiring(t *testing.T) {
	api, err := newConfig(WithRegion("us-east-1"), WithEndpoint("http://localhost"))
	require.NoError(t, err)
	if api == nil {
		t.Fatalf("expected secretsmanager client")
	}
}

// TestSmLoadConfig_AwsLoadDefaultConfigError verifies the error-return path inside
// smLoadConfig when awsLoadDefaultConfig itself fails.
func TestSmLoadConfig_AwsLoadDefaultConfigError(t *testing.T) {
	orig := awsLoadDefaultConfig
	awsLoadDefaultConfig = func(_ context.Context, _ ...func(*awsconfig.LoadOptions) error) (goaws.Config, error) {
		return goaws.Config{}, errors.New("injected aws config error")
	}
	defer func() { awsLoadDefaultConfig = orig }()

	_, err := smLoadConfig(&SecretsConfig{region: "us-east-1"})
	if err == nil || err.Error() != "injected aws config error" {
		t.Fatalf("expected injected aws config error, got %v", err)
	}
}

// TestNewConfig_LoadConfigError verifies that newConfig returns an error when
// the AWS config loader fails (no panic).
func TestNewConfig_LoadConfigError(t *testing.T) {
	orig := smLoadConfig
	smLoadConfig = func(_ *SecretsConfig) (SecretsManagerClient, error) {
		return nil, errors.New("injected load error")
	}
	defer func() { smLoadConfig = orig }()

	_, err := newConfig() // no WithClient → hits smLoadConfig
	if err == nil || err.Error() != "injected load error" {
		t.Fatalf("expected injected error, got %v", err)
	}
}

// TestWithClient_NilPanics verifies that applying WithClient(nil) panics.
func TestWithClient_NilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil client")
		}
	}()
	opt := WithClient(nil)
	opt(&SecretsConfig{}) // trigger the inner function
}

// TestWithRegion_EmptyPanics verifies that applying WithRegion("") panics.
func TestWithRegion_EmptyPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty region")
		}
	}()
	opt := WithRegion("")
	opt(&SecretsConfig{}) // trigger the inner function
}

// TestNewSecretAPI_WithClient verifies the happy path through NewSecretAPI.
func TestNewSecretAPI_WithClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, err := NewSecretAPI(WithClient(mocksm.NewMockSecretsManagerClient(ctrl)))
	require.NoError(t, err)
	require.NotNil(t, api)
}

// TestNewConfig_LoadConfigSuccess verifies the branch where smLoadConfig succeeds
// and cfg.client is assigned from the returned client.
func TestNewConfig_LoadConfigSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	fakeClient := mocksm.NewMockSecretsManagerClient(ctrl)

	orig := smLoadConfig
	smLoadConfig = func(_ *SecretsConfig) (SecretsManagerClient, error) {
		return fakeClient, nil
	}
	defer func() { smLoadConfig = orig }()

	cfg, err := newConfig() // no WithClient → hits smLoadConfig success path
	require.NoError(t, err)
	if cfg.client != fakeClient {
		t.Fatal("expected cfg.client to be set from smLoadConfig result")
	}
}

// TestNewSecretAPI_LoadConfigError verifies that NewSecretAPI propagates the error.
func TestNewSecretAPI_LoadConfigError(t *testing.T) {
	orig := smLoadConfig
	smLoadConfig = func(_ *SecretsConfig) (SecretsManagerClient, error) {
		return nil, errors.New("no aws config")
	}
	defer func() { smLoadConfig = orig }()

	_, err := NewSecretAPI()
	if err == nil {
		t.Fatal("expected error from NewSecretAPI when config load fails")
	}
}
