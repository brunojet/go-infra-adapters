package aws

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfigOptions(t *testing.T) {
	cfg := newConfig(WithRegion("r"), WithEndpoint("e"))
	if cfg.region != "r" {
		t.Fatalf("expected region r, got %s", cfg.region)
	}
	if cfg.endpoint != "e" {
		t.Fatalf("expected endpoint e, got %s", cfg.endpoint)
	}
}

// This test exercises newSecretClient wiring (calls awsconfig.LoadDefaultConfig).
// It only validates that a client instance is returned without panicking in CI.
func TestNewSecretClient_Wiring(t *testing.T) {
	cfg := newConfig(WithRegion("us-east-1"), WithEndpoint("http://localhost"))
	c, err := newSecretClient(cfg)
	require.NoError(t, err)
	if c == nil {
		t.Fatalf("expected secretsmanager client")
	}
}
