package contracts

import "testing"

// Test that CdnKey is properly structured.
func TestCdnKey(t *testing.T) {
	key := CdnKey{
		Name:      "test-key",
		PEM:       "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----",
		GroupName: "test-group",
	}
	if key.Name != "test-key" {
		t.Fatalf("expected Name=test-key, got %s", key.Name)
	}
}
