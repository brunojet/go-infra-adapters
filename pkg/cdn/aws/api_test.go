package aws

import (
	"log/slog"
	"testing"

	"github.com/golang/mock/gomock"

	mockcf "github.com/brunojet/go-infra-adapters/v3/internal/cdn/aws/mock"
	"github.com/brunojet/go-infra-adapters/v3/pkg/retry"
)

func TestWithClient_NotNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := mockcf.NewMockCloudFrontClient(ctrl)
	if WithClient(mock) == nil {
		t.Fatalf("WithClient returned nil")
	}
}

func TestWithMaxKeys_NotNil(t *testing.T) {
	if WithMaxKeys(5) == nil {
		t.Fatalf("WithMaxKeys returned nil")
	}
}

func TestWithConcurrency_NotNil(t *testing.T) {
	if WithConcurrency(3) == nil {
		t.Fatalf("WithConcurrency returned nil")
	}
}

func TestWithLogger_NotNil(t *testing.T) {
	if WithLogger(slog.Default()) == nil {
		t.Fatalf("WithLogger returned nil")
	}
}

func TestWithRetryStrategy_NotNil(t *testing.T) {
	if WithRetryStrategy(retry.NewStandard()) == nil {
		t.Fatalf("WithRetryStrategy returned nil")
	}
}

func TestNewCdn_WithClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := mockcf.NewMockCloudFrontClient(ctrl)
	cdn := NewCdn(WithClient(mock))
	if cdn == nil {
		t.Fatalf("NewCdn returned nil")
	}
}
