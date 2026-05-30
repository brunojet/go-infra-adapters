package aws

import (
	"log/slog"
	"testing"

	"github.com/golang/mock/gomock"

	mocksm "github.com/brunojet/go-infra-adapters/v3/internal/secret/aws/mock"
)

func TestOptionsWrappers_NotNil(t *testing.T) {
	if WithRegion("r") == nil {
		t.Fatalf("WithRegion returned nil")
	}
	if WithEndpoint("e") == nil {
		t.Fatalf("WithEndpoint returned nil")
	}
}

func TestExists_pkg_secret_aws_options(t *testing.T) {}

func TestWithLogger_NotNil(t *testing.T) {
	if WithLogger(slog.Default()) == nil {
		t.Fatalf("WithLogger returned nil")
	}
}

func TestNewSecretAPI_WithClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := mocksm.NewMockSecretsManagerClient(ctrl)
	api, err := NewSecretAPI(WithClient(mock))
	if err != nil || api == nil {
		t.Fatalf("NewSecretAPI with injected client: %v", err)
	}
}

func TestNewSecretAPI_WithRegionEndpoint(t *testing.T) {
	api, err := NewSecretAPI(WithRegion("us-east-1"), WithEndpoint("http://localhost"))
	if err != nil || api == nil {
		t.Fatalf("NewSecretAPI failed: %v", err)
	}
}

func TestNewSecrets_ReturnsAdapter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := mocksm.NewMockSecretsManagerClient(ctrl)
	api, _ := NewSecretAPI(WithClient(mock))
	adapter := NewSecrets[any](api, "test-secret")
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
}
