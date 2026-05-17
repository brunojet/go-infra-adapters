package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/golang/mock/gomock"

	mock_aws "github.com/brunojet/go-infra-adapters/internal/secret/aws/mock"
)

func TestConvert_WithBinaryAndStages(t *testing.T) {
	data := []byte{1, 2, 3}
	name := "n"
	arn := "a"
	vid := "v2"
	stages := []string{"s1", "s2"}

	in := &secretsmanager.GetSecretValueOutput{
		SecretBinary:  data,
		ARN:           aws.String(arn),
		VersionId:     aws.String(vid),
		Name:          aws.String(name),
		VersionStages: stages,
	}
	out := convert(in)
	if string(out.Data) != string(data) {
		t.Fatalf("unexpected data: %v", out.Data)
	}
	if out.Metadata["versionStages"] != "s1,s2" {
		t.Fatalf("unexpected stages: %v", out.Metadata)
	}
	if out.Metadata["arn"] != arn {
		t.Fatalf("unexpected arn: %v", out.Metadata)
	}
	if out.Metadata["versionId"] != vid {
		t.Fatalf("unexpected versionId: %v", out.Metadata)
	}
}

func TestSecretAdapter_ErrorPaths(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mock_aws.NewMockSecretsManagerAPI(ctrl)

	m.EXPECT().GetSecretValue(gomock.Any(), gomock.AssignableToTypeOf(&secretsmanager.GetSecretValueInput{})).Return(nil, errors.New("boom")).Times(2)

	sa := &secretAdapter{client: m, name: "n"}
	if _, err := sa.GetCurrent(context.Background()); err == nil {
		t.Fatalf("expected error from GetCurrent")
	}
	if _, err := sa.GetVersion(context.Background(), "v1"); err == nil {
		t.Fatalf("expected error from GetVersion")
	}
}
