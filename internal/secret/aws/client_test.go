package aws

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smTypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/golang/mock/gomock"

	mocksm "github.com/brunojet/go-infra-adapters/internal/secret/aws/mock"
)

type testPayload struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// newMockSvc creates a mock client and a SecretsService[testPayload] that uses it.
// The SecretAPI is constructed directly to avoid calling newConfig / AWS SDK.
func newMockSvc(t *testing.T) (*gomock.Controller, *mocksm.MockSecretsManagerClient, *SecretsService[testPayload]) {
	t.Helper()
	ctrl := gomock.NewController(t)
	client := mocksm.NewMockSecretsManagerClient(ctrl)
	api := &SecretAPI{client: client, logger: slog.Default()}
	svc := NewSecrets[testPayload](api, "my-secret")
	return ctrl, client, svc
}

// ── constructor ───────────────────────────────────────────────────────────────

func TestNewSecretsService(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	_ = client
	if svc == nil || svc.logger == nil {
		t.Fatal("expected initialized SecretsService")
	}
}

func TestSecretsServiceWithLogger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	logger := slog.New(slog.NewJSONHandler(nil, nil)) // distinct instance from slog.Default()
	api, err := NewSecretAPI(WithClient(mocksm.NewMockSecretsManagerClient(ctrl)), WithLogger(logger))
	if err != nil {
		t.Fatalf("NewSecretAPI: %v", err)
	}
	svc := NewSecrets[testPayload](api, "")
	if svc.logger != logger {
		t.Fatal("expected custom logger")
	}
}

func TestWithLoggerNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// WithLogger(nil) must not overwrite the default logger
	api, err := NewSecretAPI(WithClient(mocksm.NewMockSecretsManagerClient(ctrl)), WithLogger(nil))
	if err != nil {
		t.Fatalf("NewSecretAPI: %v", err)
	}
	svc := NewSecrets[testPayload](api, "")
	if svc.logger == nil {
		t.Fatal("expected fallback logger, got nil")
	}
}

// ── Name ──────────────────────────────────────────────────────────────────────

func TestSecretsService_Name(t *testing.T) {
	ctrl, _, svc := newMockSvc(t)
	defer ctrl.Finish()
	if svc.Name() != "my-secret" {
		t.Fatalf("expected my-secret, got %s", svc.Name())
	}
}

// ── GetCurrent / GetVersion ───────────────────────────────────────────────────

func TestSecretsService_GetCurrent_Delegates(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().GetSecretValue(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.GetSecretValueOutput{SecretString: aws.String(`{"key":"k","value":"v"}`)}, nil,
	)
	got, err := svc.GetCurrent(context.Background())
	if err != nil || got.Key != "k" || got.Value != "v" {
		t.Fatalf("unexpected result: %+v %v", got, err)
	}
}

func TestSecretsService_GetVersion_NotFound_ReturnsZero(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().GetSecretValue(gomock.Any(), gomock.Any()).Return(
		nil, &smTypes.ResourceNotFoundException{Message: aws.String("not found")},
	)
	got, err := svc.GetVersion(context.Background(), "")
	if err != nil || got == nil {
		t.Fatalf("expected zero-value on not-found: got=%v err=%v", got, err)
	}
}

func TestSecretsService_GetVersion_NilSecretString_ReturnsZero(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().GetSecretValue(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.GetSecretValueOutput{SecretString: nil}, nil,
	)
	got, err := svc.GetVersion(context.Background(), "")
	if err != nil || got == nil {
		t.Fatalf("expected zero-value on nil string: got=%v err=%v", got, err)
	}
}

func TestSecretsService_GetVersion_WithVersionID(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().GetSecretValue(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.GetSecretValueOutput{SecretString: aws.String(`{"key":"x","value":"y"}`)}, nil,
	)
	got, err := svc.GetVersion(context.Background(), "v1")
	if err != nil || got.Key != "x" {
		t.Fatalf("unexpected: %+v %v", got, err)
	}
}

func TestSecretsService_GetVersion_Error(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().GetSecretValue(gomock.Any(), gomock.Any()).Return(nil, errors.New("aws error"))
	_, err := svc.GetVersion(context.Background(), "v1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// ── SetVersion ────────────────────────────────────────────────────────────────

func TestSecretsService_SetVersion_Success(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().PutSecretValue(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.PutSecretValueOutput{VersionId: aws.String("v-new")}, nil,
	)
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.UpdateSecretVersionStageOutput{}, nil,
	)
	id, err := svc.SetVersion(context.Background(), &testPayload{Key: "k"}, "v-new")
	if err != nil || id != "v-new" {
		t.Fatalf("expected v-new: id=%s err=%v", id, err)
	}
}

func TestSecretsService_SetVersion_EmptyVersionToken(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().PutSecretValue(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.PutSecretValueOutput{VersionId: aws.String("aws-gen")}, nil,
	)
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.UpdateSecretVersionStageOutput{}, nil,
	)
	id, err := svc.SetVersion(context.Background(), &testPayload{}, "")
	if err != nil || id != "aws-gen" {
		t.Fatalf("expected aws-gen: id=%s err=%v", id, err)
	}
}

func TestSecretsService_SetVersion_PutError(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().PutSecretValue(gomock.Any(), gomock.Any()).Return(nil, errors.New("put failed"))
	_, err := svc.SetVersion(context.Background(), &testPayload{}, "v1")
	if err == nil {
		t.Fatal("expected error from PutSecretValue")
	}
}

func TestSecretsService_SetVersion_MovePendingError(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().PutSecretValue(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.PutSecretValueOutput{VersionId: aws.String("v1")}, nil,
	)
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(nil, errors.New("stage error"))
	_, err := svc.SetVersion(context.Background(), &testPayload{}, "v1")
	if err == nil {
		t.Fatal("expected error from movePendingStage")
	}
}

// unmarshalable type to force marshal failure
type unmarshalablePayload struct {
	Ch chan int
}

func TestSecretsService_SetVersion_MarshalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api := &SecretAPI{client: mocksm.NewMockSecretsManagerClient(ctrl), logger: slog.Default()}
	svc := NewSecrets[unmarshalablePayload](api, "test")
	_, err := svc.SetVersion(context.Background(), &unmarshalablePayload{Ch: make(chan int)}, "v1")
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

// ── PromoteVersion ────────────────────────────────────────────────────────────

func TestSecretsService_PromoteVersion_Success(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.DescribeSecretOutput{
			VersionIdsToStages: map[string][]string{"curr-v1": {"AWSCURRENT"}},
		}, nil,
	)
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.UpdateSecretVersionStageOutput{}, nil,
	)
	if err := svc.PromoteVersion(context.Background(), "pend-v2"); err != nil {
		t.Fatalf("PromoteVersion: %v", err)
	}
}

func TestSecretsService_PromoteVersion_NoPreviousCurrent(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.DescribeSecretOutput{VersionIdsToStages: map[string][]string{}}, nil,
	)
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.UpdateSecretVersionStageOutput{}, nil,
	)
	if err := svc.PromoteVersion(context.Background(), "pend-v1"); err != nil {
		t.Fatalf("PromoteVersion no current: %v", err)
	}
}

func TestSecretsService_PromoteVersion_AlreadyCurrent(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.DescribeSecretOutput{
			VersionIdsToStages: map[string][]string{"v1": {"AWSCURRENT"}},
		}, nil,
	)
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.UpdateSecretVersionStageOutput{}, nil,
	)
	if err := svc.PromoteVersion(context.Background(), "v1"); err != nil {
		t.Fatalf("PromoteVersion already current: %v", err)
	}
}

func TestSecretsService_PromoteVersion_DescribeError(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(nil, errors.New("describe fail"))
	if err := svc.PromoteVersion(context.Background(), "v1"); err == nil {
		t.Fatal("expected error from DescribeSecret")
	}
}

func TestSecretsService_PromoteVersion_UpdateError(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.DescribeSecretOutput{
			VersionIdsToStages: map[string][]string{"curr": {"AWSCURRENT"}},
		}, nil,
	)
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(nil, errors.New("update fail"))
	if err := svc.PromoteVersion(context.Background(), "pend"); err == nil {
		t.Fatal("expected error from UpdateSecretVersionStage")
	}
}

// ── DiscardVersion ────────────────────────────────────────────────────────────

func TestSecretsService_DiscardVersion_Success(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.UpdateSecretVersionStageOutput{}, nil,
	)
	if err := svc.DiscardVersion(context.Background(), "v1"); err != nil {
		t.Fatalf("DiscardVersion: %v", err)
	}
}

func TestSecretsService_DiscardVersion_Error(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().UpdateSecretVersionStage(gomock.Any(), gomock.Any()).Return(nil, errors.New("err"))
	if err := svc.DiscardVersion(context.Background(), "v1"); err == nil {
		t.Fatal("expected error")
	}
}

// ── HealthCheck ───────────────────────────────────────────────────────────────

func TestSecretsService_HealthCheck_Success(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		&secretsmanager.DescribeSecretOutput{}, nil,
	)
	if err := svc.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
}

func TestSecretsService_HealthCheck_NotFound(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
		nil, &smTypes.ResourceNotFoundException{Message: aws.String("no")},
	)
	err := svc.HealthCheck(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error, got %v", err)
	}
}

func TestSecretsService_HealthCheck_ConnectivityError(t *testing.T) {
	ctrl, client, svc := newMockSvc(t)
	defer ctrl.Finish()
	client.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(nil, errors.New("network failure"))
	if err := svc.HealthCheck(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
