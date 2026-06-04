package aws

import (
	"context"
	"testing"

	cryptopkg "github.com/brunojet/go-infra-adapters/v4/pkg/crypto"
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
	if signer.keyID != "K31UKMLKEO2DC4" {
		t.Errorf("keyID mismatch: got %s, want K31UKMLKEO2DC4", signer.keyID)
	}
}

func TestNewCloudFrontSignerFromPEM_InvalidKey(t *testing.T) {
	_, err := NewCloudFrontSignerFromPEM("K31UKMLKEO2DC4", []byte("invalid pem"))
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestCloudFrontSigner_SignURL(t *testing.T) {
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
	expiresAt := int64(1609459200) // 2021-01-01 00:00:00 UTC

	signedURL, err := signer.SignURL(ctx, resourceURL, expiresAt)
	if err != nil {
		t.Fatalf("SignURL failed: %v", err)
	}

	// Verify signed URL contains expected components
	if signedURL == "" {
		t.Fatal("signed URL is empty")
	}
	if !contains(signedURL, "Expires=1609459200") {
		t.Errorf("signed URL missing Expires parameter: %s", signedURL)
	}
	if !contains(signedURL, "Signature=") {
		t.Errorf("signed URL missing Signature parameter: %s", signedURL)
	}
	if !contains(signedURL, "Key-Pair-Id=K31UKMLKEO2DC4") {
		t.Errorf("signed URL missing Key-Pair-Id parameter: %s", signedURL)
	}
	if !contains(signedURL, resourceURL) {
		t.Errorf("signed URL missing resource URL: %s", signedURL)
	}
}

func TestCloudFrontSigner_SignURL_DeterministicWithSameSigner(t *testing.T) {
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

	url1, err := signer.SignURL(ctx, resourceURL, expiresAt)
	if err != nil {
		t.Fatalf("first SignURL failed: %v", err)
	}

	url2, err := signer.SignURL(ctx, resourceURL, expiresAt)
	if err != nil {
		t.Fatalf("second SignURL failed: %v", err)
	}

	if url1 != url2 {
		t.Errorf("signatures not deterministic: %s != %s", url1, url2)
	}
}

func TestCloudFrontSigner_Base64URLSafe_Encoding(t *testing.T) {
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

	// Test data with characters that would be affected by AWS-specific encoding
	testData := []byte{0xff, 0xfe, 0xfd, 0xfc, 0xfb, 0xfa}

	encoded := signer.base64URLSafe(testData)

	// AWS-specific encoding should NOT contain standard base64 characters: + / =
	if contains(encoded, "+") {
		t.Errorf("encoded string contains '+' (should be '-'): %s", encoded)
	}
	if contains(encoded, "/") {
		t.Errorf("encoded string contains '/' (should be '~'): %s", encoded)
	}
	if contains(encoded, "=") {
		t.Errorf("encoded string contains '=' (should be '_'): %s", encoded)
	}
}

func TestCloudFrontSigner_SignURL_DifferentExpiresProducesDifferentSignatures(t *testing.T) {
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

	url1, err := signer.SignURL(ctx, resourceURL, int64(1609459200))
	if err != nil {
		t.Fatalf("first SignURL failed: %v", err)
	}

	url2, err := signer.SignURL(ctx, resourceURL, int64(1609459300))
	if err != nil {
		t.Fatalf("second SignURL failed: %v", err)
	}

	// Different expiry times should produce different signatures
	if url1 == url2 {
		t.Error("different expiry times produced identical signatures")
	}
}

func TestCloudFrontSigner_SignURL_DifferentResourcesProduceDifferentSignatures(t *testing.T) {
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

	expiresAt := int64(1609459200)

	url1, err := signer.SignURL(ctx, "https://example.com/video1.mp4", expiresAt)
	if err != nil {
		t.Fatalf("first SignURL failed: %v", err)
	}

	url2, err := signer.SignURL(ctx, "https://example.com/video2.mp4", expiresAt)
	if err != nil {
		t.Fatalf("second SignURL failed: %v", err)
	}

	// Different resources should produce different signatures
	if url1 == url2 {
		t.Error("different resources produced identical signatures")
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
