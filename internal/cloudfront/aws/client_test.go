package aws

import (
	"context"
	"testing"

	mockaws "github.com/brunojet/go-infra-adapters/internal/cloudfront/aws/mock"
	"github.com/golang/mock/gomock"
)

func TestMockCloudFrontAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mockaws.NewMockCloudFrontAPI(ctrl)

	m.EXPECT().CreatePublicKey(gomock.Any(), "name", "pemdata").Return("pkid", nil)
	id, err := m.CreatePublicKey(context.Background(), "name", "pemdata")
	if err != nil {
		t.Fatalf("CreatePublicKey error: %v", err)
	}
	if id != "pkid" {
		t.Fatalf("unexpected id: %s", id)
	}

	m.EXPECT().CreateKeyGroup(gomock.Any(), "kg", []string{"pkid"}).Return("kgid", nil)
	kg, err := m.CreateKeyGroup(context.Background(), "kg", []string{"pkid"})
	if err != nil {
		t.Fatalf("CreateKeyGroup error: %v", err)
	}
	if kg != "kgid" {
		t.Fatalf("unexpected keygroup id: %s", kg)
	}
}
