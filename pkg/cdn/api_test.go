package cdn

import (
	"context"
	"testing"

	"github.com/brunojet/go-infra-adapters/v4/pkg/cdn/contracts"
)

// testPrivateKeyPEM is a valid RSA private key for testing only.
// It is safe to hardcode in tests and should never be used in production.
// nolint:gosec // G101: hardcoded credentials are acceptable in test files
const testPrivateKeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCqn+Pd40KItr+D
WljXXqHs+VsoD1ccpWsXLMEfY7l/nDq/exhUfsh9jM+AH/bCSW0GCS1NibpjLv1t
C+dbWawwLdpGi+ao+MrRmkM9VaQcOSsP1zv9iagsXLzqYONKnEA3YIDBCsZjuxHl
7wsHt2o9Zg5MFdwtVgYVrYpe5LFVLf4YDI5zTscWizvTpCwxXdub9KxR8IvfvxOK
WadRKyvVBcfV5mGFDEuG/w75EFprN7SK5QAF40QEWYKScxgIyWyjNcueCpTq8qL5
hz1Y6un5ij4VAnM649Ds72pULATdaRrkmFGjiMib5xDkf45f72Yh/RsUhm4BuQru
P7p3xZ2PAgMBAAECggEAHXWw3slT3g4LoA7T4w+0Tpm5Ov/3Dvuis2QnThemWhmr
7Q7AYypmzIKo+xrJvL73w4CHIWmj1GczG4ZgIl4nxEPOebrDDy6xuiHz9R2Z0cOv
IzOK6JpBfrNebOtgoyu6TLVtVadaHLMagoRU97ab8dDyrAFkPDGrqEeH6h17XtTj
Ywb475GziPe+rkL2SLQaOTKZwqIlLavjFfk4WAlbAt9bianMaSfxqHcGraGUWqgt
dAvz+s92DIrup2JQ5C+wIJK6vCiTkWdZZaYZL4ir7QAg/olPDTtrK0NuQZZlNrha
pVwCrwugrdpzA+0N/RQee3lfLnxt2vxhP6LmMPX55QKBgQDXnZuIw9AJ3s3BrcyD
oFL/okCuE6MKnnPC0406TSh9GsM9GUIjlzZ5vFfOks0wo9dR5LsE5Hrcs6Oz/fqg
/rYy9DteF1UYJGy099aSfNX1KYSUGXCqlP6nOgcUUYxu+p7i5+a7CmuLVJTS4Az3
PuGVPhLUDlWh9yzwp7VjRi5azQKBgQDKlQn7mCfOADQf3gUTJMo5gcFo90zL6BNc
WIpsyJtqOpAKuxKqy8RbrLTeLK3KJ2FJrbjzotQs5oOrB+AcIBviFuSLKny1MD8H
963dpa+i6RDC17FJFSYHFIy+MU3iTn04tMqnjau7AtA53mSK+iN3MiyeSZttP89N
KuEhKOwRywKBgAa7NuXYJyCHwiivwljBopW0fQxyNH7aX4bPj/MoAYGWWk4IAdaW
m+7FAIDEeH9yPgCigWwvrd5CBXRTE4X/LbT9hvTzCYcNbA9iRWKhXxSeTTNKcAgD
SsfxudLakOXOETPIRZ3FP4JEC7lhoUX+wpAkNfZE1EuQKekBc1o8EKppAoGBAKOz
wihkcS3/bh+eSt2Iaj4EQ6WtyYow1IxYJCv6A9TY1BNHzrLkDJ3ENzgeRKXKIszm
LEH8/5X1BMtNhuVTcRTHSRHIWJQWE6k9lWs5+28bBWdd4y5af6tTCNSchQJuSLRt
LCIv0mlBwcAxnW/M6KHmkrWqZ4Xl/X+vOdOQ/Qr3AoGAFDkBqRMJZMshiBUQEEa0
sNOxt53HO0JfS4HMiC39yNrg5UOpV95yMG7UPghU2TZhXcfGa9bGKSVfea4vDyfv
WFQU6NyN+GU3fKw5A/khAvKpAUFLbTf+KcuRtHSfhusdRi5h5q1L6TK/82FxM9ph
b7MrKzQ2N1UcaCnnVabdUrE=
-----END PRIVATE KEY-----`

func TestNewCloudFrontSignerFromPEM_Success(t *testing.T) {
	signer, err := NewCloudFrontSignerFromPEM("K31UKMLKEO2DC4", []byte(testPrivateKeyPEM))
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
	signer, err := NewCloudFrontSignerFromPEM("K31UKMLKEO2DC4", []byte(testPrivateKeyPEM))
	if err != nil {
		t.Fatalf("NewCloudFrontSignerFromPEM failed: %v", err)
	}

	ctx := context.Background()
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
	signer1, _ := NewCloudFrontSignerFromPEM("K1", []byte(testPrivateKeyPEM))
	signer2, _ := NewCloudFrontSignerFromPEM("K2", []byte(testPrivateKeyPEM))

	ctx := context.Background()
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
