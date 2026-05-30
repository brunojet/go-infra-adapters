package aws

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/golang/mock/gomock"

	mocksm "github.com/brunojet/go-infra-adapters/v3/internal/secret/aws/mock"
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

// ── moveStage ────────────────────────────────────────────────────────────────

func TestMoveStage_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mocksm.NewMockSecretsManagerClient(ctrl)
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.DescribeSecretOutput{
			VersionIdsToStages: map[string][]string{
				"v0": {"AWSPENDING"},
				"v1": {},
			},
		}, nil,
	).Times(1)
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.UpdateSecretVersionStageOutput{}, nil,
	).Times(1)
	api := &SecretAPI{client: client, logger: slog.Default()}
	svc := NewSecrets[testPayload](api, "s")
	if err := svc.moveStage(context.Background(), "AWSPENDING", "v1"); err != nil {
		t.Fatalf("moveStage: %v", err)
	}
}

func TestMoveStage_Idempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mocksm.NewMockSecretsManagerClient(ctrl)
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.DescribeSecretOutput{
			VersionIdsToStages: map[string][]string{
				"v1": {"AWSCURRENT"},
			},
		}, nil,
	).Times(1)
	// No UpdateSecretVersionStage call (idempotent)
	api := &SecretAPI{client: client, logger: slog.Default()}
	svc := NewSecrets[testPayload](api, "s")
	if err := svc.moveStage(context.Background(), "AWSCURRENT", "v1"); err != nil {
		t.Fatalf("moveStage idempotent: %v", err)
	}
}

// ── movePendingStage ──────────────────────────────────────────────────────────

func TestMovePendingStage_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mocksm.NewMockSecretsManagerClient(ctrl)
	// First call: DescribeSecret to find existing AWSPENDING
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.DescribeSecretOutput{
			VersionIdsToStages: map[string][]string{
				"v0": {"AWSPENDING"},
				"v1": {},
			},
		}, nil,
	).Times(1)
	// Second call: UpdateSecretVersionStage to move AWSPENDING
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.UpdateSecretVersionStageOutput{}, nil,
	).Times(1)
	api := &SecretAPI{client: client, logger: slog.Default()}
	svc := NewSecrets[testPayload](api, "s")
	if err := svc.movePendingStage(context.Background(), "v1"); err != nil {
		t.Fatalf("movePendingStage: %v", err)
	}
}

func TestMovePendingStage_NoPreviousPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mocksm.NewMockSecretsManagerClient(ctrl)
	// DescribeSecret with no existing AWSPENDING
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.DescribeSecretOutput{
			VersionIdsToStages: map[string][]string{
				"v1": {},
			},
		}, nil,
	).Times(1)
	// UpdateSecretVersionStage without RemoveFromVersionId
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.UpdateSecretVersionStageOutput{}, nil,
	).Times(1)
	api := &SecretAPI{client: client, logger: slog.Default()}
	svc := NewSecrets[testPayload](api, "s")
	if err := svc.movePendingStage(context.Background(), "v1"); err != nil {
		t.Fatalf("movePendingStage: %v", err)
	}
}

func TestMovePendingStage_AlreadyHasPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mocksm.NewMockSecretsManagerClient(ctrl)
	// DescribeSecret shows v1 already has AWSPENDING
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.DescribeSecretOutput{
			VersionIdsToStages: map[string][]string{
				"v1": {"AWSPENDING"},
			},
		}, nil,
	).Times(1)
	// No UpdateSecretVersionStage call needed (idempotent)
	api := &SecretAPI{client: client, logger: slog.Default()}
	svc := NewSecrets[testPayload](api, "s")
	if err := svc.movePendingStage(context.Background(), "v1"); err != nil {
		t.Fatalf("movePendingStage: %v", err)
	}
}

func TestMovePendingStage_DescribeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mocksm.NewMockSecretsManagerClient(ctrl)
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(nil, errors.New("describe fail"))
	api := &SecretAPI{client: client, logger: slog.Default()}
	svc := NewSecrets[testPayload](api, "s")
	if err := svc.movePendingStage(context.Background(), "v1"); err == nil {
		t.Fatal("expected error from DescribeSecret")
	}
}

func TestMovePendingStage_UpdateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mocksm.NewMockSecretsManagerClient(ctrl)
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.DescribeSecretOutput{
			VersionIdsToStages: map[string][]string{"v0": {"AWSPENDING"}},
		}, nil,
	)
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(nil, errors.New("stage fail"))
	api := &SecretAPI{client: client, logger: slog.Default()}
	svc := NewSecrets[testPayload](api, "s")
	if err := svc.movePendingStage(context.Background(), "v1"); err == nil {
		t.Fatal("expected error from UpdateSecretVersionStage")
	}
}
