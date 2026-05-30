package aws

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/golang/mock/gomock"

	mocksm "github.com/brunojet/go-infra-adapters/internal/secret/aws/mock"
)

// ── getVersionWithStage ───────────────────────────────────────────────────────

func TestGetVersionWithStage_Found(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mocksm.NewMockSecretsManagerClient(ctrl)
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.DescribeSecretOutput{
			VersionIdsToStages: map[string][]string{
				"v1": {"AWSCURRENT"},
				"v2": {"AWSPENDING"},
			},
		}, nil,
	)
	api := &SecretAPI{client: client, logger: slog.Default()}
	svc := NewSecrets[testPayload](api, "s")
	got, err := svc.getVersionWithStage(context.Background(), "AWSCURRENT")
	if err != nil || got != "v1" {
		t.Fatalf("expected v1: got=%s err=%v", got, err)
	}
}

func TestGetVersionWithStage_NotFound_ReturnsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mocksm.NewMockSecretsManagerClient(ctrl)
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.DescribeSecretOutput{
			VersionIdsToStages: map[string][]string{"v1": {"AWSPENDING"}},
		}, nil,
	)
	api := &SecretAPI{client: client, logger: slog.Default()}
	svc := NewSecrets[testPayload](api, "s")
	got, err := svc.getVersionWithStage(context.Background(), "AWSCURRENT")
	if err != nil || got != "" {
		t.Fatalf("expected empty string: got=%s err=%v", got, err)
	}
}

func TestGetVersionWithStage_DescribeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mocksm.NewMockSecretsManagerClient(ctrl)
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(nil, errors.New("describe fail"))
	api := &SecretAPI{client: client, logger: slog.Default()}
	svc := NewSecrets[testPayload](api, "s")
	_, err := svc.getVersionWithStage(context.Background(), "AWSCURRENT")
	if err == nil {
		t.Fatal("expected error")
	}
}

// ── movePendingStage ──────────────────────────────────────────────────────────

func TestMovePendingStage_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mocksm.NewMockSecretsManagerClient(ctrl)
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.UpdateSecretVersionStageOutput{}, nil,
	)
	api := &SecretAPI{client: client, logger: slog.Default()}
	svc := NewSecrets[testPayload](api, "s")
	if err := svc.movePendingStage(context.Background(), "v1"); err != nil {
		t.Fatalf("movePendingStage: %v", err)
	}
}

func TestMovePendingStage_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mocksm.NewMockSecretsManagerClient(ctrl)
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(nil, errors.New("stage fail"))
	api := &SecretAPI{client: client, logger: slog.Default()}
	svc := NewSecrets[testPayload](api, "s")
	if err := svc.movePendingStage(context.Background(), "v1"); err == nil {
		t.Fatal("expected error")
	}
}
