package s3

import "testing"

func TestOptionsWrappers_NotNil(t *testing.T) {
	if WithRegion("r") == nil {
		t.Fatalf("WithRegion returned nil")
	}
	if WithEndpoint("e") == nil {
		t.Fatalf("WithEndpoint returned nil")
	}
}

func TestExists_pkg_storage_s3_options(t *testing.T) {}

func TestNewS3Client_Wrapper(t *testing.T) {
	c, err := NewS3Client(WithRegion("us-east-1"), WithEndpoint("http://localhost"))
	if err != nil || c == nil {
		t.Fatalf("NewS3Client failed: %v", err)
	}
}
