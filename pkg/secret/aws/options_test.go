package aws

import "testing"

func TestOptionsWrappers_NotNil(t *testing.T) {
	if WithRegion("r") == nil {
		t.Fatalf("WithRegion returned nil")
	}
	if WithEndpoint("e") == nil {
		t.Fatalf("WithEndpoint returned nil")
	}
}

func TestExists_pkg_secret_aws_options(t *testing.T) {}

func TestNewSecretClient_Wrapper(t *testing.T) {
	c, err := NewSecretClient(WithRegion("us-east-1"), WithEndpoint("http://localhost"))
	if err != nil || c == nil {
		t.Fatalf("NewSecretClient failed: %v", err)
	}
}
