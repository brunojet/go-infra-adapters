package aws

import (
	"log/slog"
	"testing"

	"github.com/golang/mock/gomock"

	mockcf "github.com/brunojet/go-infra-adapters/internal/cdn/aws/mock"
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

func TestNewCdn_WithClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := mockcf.NewMockCloudFrontClient(ctrl)
	cdn := NewCdn(WithClient(mock))
	if cdn == nil {
		t.Fatalf("NewCdn returned nil")
	}
}
