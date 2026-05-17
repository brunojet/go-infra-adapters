package s3

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
	// newS3Client uses aws config and may fail in tests depending on environment; we only validate option wiring here
	_ = cfg
}

func TestNewS3Client_Wiring(t *testing.T) {
	cfg := newConfig(WithRegion("us-east-1"), WithEndpoint("http://localhost"))
	c, err := newS3Client(cfg)
	require.NoError(t, err)
	if c == nil {
		t.Fatalf("expected s3 client")
	}
}
