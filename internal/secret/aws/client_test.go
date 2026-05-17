package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	mock_aws "github.com/brunojet/go-infra-adapters/internal/secret/aws/mock"
)

func TestConvert_WithSecretString(t *testing.T) {
	s := "value"
	name := "mysecret"
	arn := "arn:aws:secrets:example"
	vid := "v1"
	in := &secretsmanager.GetSecretValueOutput{
		SecretString: &s,
		ARN:          aws.String(arn),
		VersionId:    aws.String(vid),
		Name:         aws.String(name),
	}
	out := convert(in)
	if string(out.Data) != s {
		t.Fatalf("unexpected data: %s", string(out.Data))
	}
	if out.Metadata["arn"] != arn {
		t.Fatalf("unexpected arn: %v", out.Metadata["arn"])
	}
	if out.Metadata["versionId"] != vid {
		t.Fatalf("unexpected versionId: %v", out.Metadata["versionId"])
	}
	if out.Metadata["name"] != name {
		t.Fatalf("unexpected name: %v", out.Metadata["name"])
	}
}

func TestSecretAdapter_GetCurrentAndGetVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_aws.NewMockSecretsManagerAPI(ctrl)

	// Return a simple secret for any GetSecretValue call.
	secret := "somedata"
	mockClient.EXPECT().GetSecretValue(gomock.Any(), gomock.AssignableToTypeOf(&secretsmanager.GetSecretValueInput{})).Return(&secretsmanager.GetSecretValueOutput{
		SecretString: &secret,
	}, nil).AnyTimes()

	sa := &secretAdapter{client: mockClient, name: "ns"}

	// GetCurrent
	v, err := sa.GetCurrent(context.Background())
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}
	if string(v.Data) != secret {
		t.Fatalf("unexpected data: %s", string(v.Data))
	}

	// GetVersion with empty should error
	if _, err := sa.GetVersion(context.Background(), ""); err == nil {
		t.Fatalf("expected error for empty version")
	}

	// GetVersion with value
	if _, err := sa.GetVersion(context.Background(), "v1"); err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}
}

func TestNewSecretWithClient_ErrorsAndSuccess(t *testing.T) {
	// empty name
	sc := &SecretsClient{}
	if _, err := sc.NewSecretWithClient("", nil); err == nil {
		t.Fatalf("expected error for empty name")
	}

	// invalid client type
	type bad struct{}
	if _, err := sc.NewSecretWithClient("n", bad{}); err == nil {
		t.Fatalf("expected error for invalid client type")
	}

	// valid client
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mock_aws.NewMockSecretsManagerAPI(ctrl)
	ad, err := sc.NewSecretWithClient("sname", m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ad.Name() != "sname" {
		t.Fatalf("unexpected adapter name: %s", ad.Name())
	}
}

func TestSecretsClient_NewSecretWithClient_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_aws.NewMockSecretsManagerAPI(ctrl)

	sc := &SecretsClient{}
	ad, err := sc.NewSecretWithClient("sname", mockClient)
	require.NoError(t, err)
	require.NotNil(t, ad)
	require.Equal(t, "sname", ad.Name())
}

func TestSecretsClient_NewSecretWithClient_Errors(t *testing.T) {
	sc := &SecretsClient{}
	_, err := sc.NewSecretWithClient("", nil)
	require.Error(t, err)

	_, err = sc.NewSecretWithClient("n", "not-a-client")
	require.Error(t, err)
}

func TestSecretsClient_NewSecret_WithPreSetClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_aws.NewMockSecretsManagerAPI(ctrl)
	sc := &SecretsClient{client: mockClient}
	ad, err := sc.NewSecret("sname")
	require.NoError(t, err)
	require.NotNil(t, ad)
	require.Equal(t, "sname", ad.Name())
}

func TestNewSecretClient_WrapperIntegration(t *testing.T) {
	c, err := NewSecretClient(WithRegion("us-east-1"), WithEndpoint("http://localhost"))
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestSecretsClient_defaultSecretAdapter_Integration(t *testing.T) {
	sc := &SecretsClient{}
	client, err := sc.defaultSecretAdapter()
	if err != nil {
		t.Skipf("skipping defaultSecretAdapter integration: %v", err)
	}
	require.NotNil(t, client)
}
