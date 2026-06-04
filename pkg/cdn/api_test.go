package cdn

import (
	"context"
	"testing"

	cryptopkg "github.com/brunojet/go-infra-adapters/v4/pkg/crypto"
	"github.com/brunojet/go-infra-adapters/v4/pkg/cdn/contracts"
)

func TestNewCloudFrontSignerFromPEM_Success(t *testing.T) {
	ctx := context.Background()
	keyGen := cryptopkg.NewRSAKeyGenerator(2048)
	kp, err := keyGen.Generate(ctx)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	signer, err := NewCloudFrontSignerFromPEM("K31UKMLKEO2DC4", kp.PrivatePEM)
	if err != nil {
		t.Fatalf("NewCloudFrontSignerFromPEM failed: %v", err)
	}
	if signer == nil {
		t.Fatal("signer is nil")
	}

	// Verify it implements URLSigner interface
	var _ contracts.URLSigner = signer
}

func TestNewCloudFrontSignerFromPEM_InvalidKey(t *testing.T) {
	_, err := NewCloudFrontSignerFromPEM("K31UKMLKEO2DC4", []byte("invalid pem"))
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestNewCloudFrontSignerFromPEM_SignURL(t *testing.T) {
	ctx := context.Background()
	keyGen := cryptopkg.NewRSAKeyGenerator(2048)
	kp, err := keyGen.Generate(ctx)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	signer, err := NewCloudFrontSignerFromPEM("K31UKMLKEO2DC4", kp.PrivatePEM)
	if err != nil {
		t.Fatalf("NewCloudFrontSignerFromPEM failed: %v", err)
	}

	resourceURL := "https://example.com/test.mp4"
	expiresAt := int64(1609459200)

	signedURL, err := signer.SignURL(ctx, resourceURL, expiresAt)
	if err != nil {
		t.Fatalf("SignURL failed: %v", err)
	}

	if signedURL == "" {
		t.Fatal("signed URL is empty")
	}

	// Verify URL structure
	if !contains(signedURL, "Expires=") {
		t.Error("missing Expires parameter")
	}
	if !contains(signedURL, "Signature=") {
		t.Error("missing Signature parameter")
	}
	if !contains(signedURL, "Key-Pair-Id=K31UKMLKEO2DC4") {
		t.Error("missing Key-Pair-Id parameter")
	}
}

func TestNewCloudFrontSignerFromPEM_DifferentKeyIDsMakeDifferentSignatures(t *testing.T) {
	ctx := context.Background()
	keyGen := cryptopkg.NewRSAKeyGenerator(2048)
	kp, err := keyGen.Generate(ctx)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	signer1, _ := NewCloudFrontSignerFromPEM("K1", kp.PrivatePEM)
	signer2, _ := NewCloudFrontSignerFromPEM("K2", kp.PrivatePEM)

	resourceURL := "https://example.com/test.mp4"
	expiresAt := int64(1609459200)

	url1, _ := signer1.SignURL(ctx, resourceURL, expiresAt)
	url2, _ := signer2.SignURL(ctx, resourceURL, expiresAt)

	if url1 == url2 {
		t.Error("different Key-Pair-Ids should produce different URLs")
	}

	if !contains(url1, "Key-Pair-Id=K1") {
		t.Error("url1 should contain K1")
	}
	if !contains(url2, "Key-Pair-Id=K2") {
		t.Error("url2 should contain K2")
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
