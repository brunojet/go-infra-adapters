# CloudFront Signed URL Implementation Proposal

## Problem Statement

The current `RSASigner` in `internal/crypto/rsa.go` uses **SHA256** hashing with RSA-PKCS1v15, which is optimized for general-purpose signing. However, **AWS CloudFront signed URLs require SHA1** with a specific **Canned Policy** JSON format.

Current gap:
- ✗ No SHA1 support in adapter
- ✗ No CloudFront Canned Policy helper  
- ✗ No URL-safe base64 encoding (AWS-specific: `+→-`, `/→~`, `=→_`)

## Current Adapter Structure

```
pkg/crypto/
  └─ api.go (public API factory functions)
  └─ contracts/contracts.go (interfaces: KeyGenerator, Signer, Verifier)

internal/crypto/
  └─ rsa.go (RSAKeyGenerator, RSASigner, RSAVerifier implementations)
  └─ rsa_test.go (unit tests)
```

## Proposed Solution: Add `CloudFrontSigner` (Recommended)

Create specialized type for CloudFront signing with:
- SHA1 hashing (vs current SHA256)
- Canned Policy JSON generation
- AWS-specific base64 URL-safe encoding

### Step 1: Create `internal/crypto/cloudfront.go`

```go
package crypto

import (
	"crypto"
	stdcrypto "crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"
)

// cloudFrontSignPKCS1v15 wraps rsa.SignPKCS1v15; replaceable in tests
var cloudFrontSignPKCS1v15 = rsa.SignPKCS1v15

// CloudFrontURLSigner signs CloudFront URLs using AWS Canned Policy format
type CloudFrontURLSigner struct {
	keyPairID  string
	privateKey *rsa.PrivateKey
}

// NewCloudFrontURLSignerFromPEM creates a CloudFront signer from PEM-encoded RSA private key
// Accepts PKCS1 ("RSA PRIVATE KEY") and PKCS8 ("PRIVATE KEY") formats.
func NewCloudFrontURLSignerFromPEM(keyPairID string, privateKeyPEM []byte) (*CloudFrontURLSigner, error) {
	priv, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return &CloudFrontURLSigner{
		keyPairID:  keyPairID,
		privateKey: priv,
	}, nil
}

// SignURL generates CloudFront signed URL with Canned Policy (SHA1 + PKCS1v15)
// Format: URL?Expires=TIMESTAMP&Signature=BASE64_SAFE&Key-Pair-Id=KEYID
func (s *CloudFrontURLSigner) SignURL(resourceURL string, expiresAt time.Time) (string, error) {
	// Canned Policy JSON format (exact AWS format)
	policy := fmt.Sprintf(
		`{"Statement":[{"Resource":"%s","Condition":{"DateLessThan":{"AWS:EpochTime":%d}}}]}`,
		resourceURL,
		expiresAt.Unix(),
	)

	// Sign with SHA1 (CloudFront requirement)
	h := sha1.Sum([]byte(policy))
	sig, err := cloudFrontSignPKCS1v15(stdcrypto.Reader, s.privateKey, crypto.SHA1, h[:])
	if err != nil {
		return nil, fmt.Errorf("sign policy: %w", err)
	}

	// AWS-specific base64 URL-safe encoding: + → -, / → ~, = → _
	encodedSig := s.base64URLSafe(sig)

	// Build signed URL
	return fmt.Sprintf("%s?Expires=%d&Signature=%s&Key-Pair-Id=%s",
		resourceURL,
		expiresAt.Unix(),
		encodedSig,
		s.keyPairID,
	), nil
}

// base64URLSafe encodes with AWS-specific replacements
func (s *CloudFrontURLSigner) base64URLSafe(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "~")
	encoded = strings.ReplaceAll(encoded, "=", "_")
	return encoded
}
```

### Step 2: Add public API in `pkg/crypto/api.go`

```go
// NewCloudFrontURLSignerFromPEM returns a CloudFront URL signer.
// Generates signed URLs using AWS Canned Policy format with SHA1 hashing
// and AWS-specific base64 URL-safe encoding.
// keyPairID: AWS CloudFront public key ID (e.g., "K31UKMLKEO2DC4")
// Complexity: O(n) where n = len(privateKeyPEM). Memory: ~1-5 KB for parsed key.
func NewCloudFrontURLSignerFromPEM(keyPairID string, privateKeyPEM []byte) (*internal.CloudFrontURLSigner, error) {
	return internal.NewCloudFrontURLSignerFromPEM(keyPairID, privateKeyPEM)
}
```

### Step 3: Add interface in `pkg/crypto/contracts/contracts.go`

```go
// CloudFrontURLSigner generates AWS CloudFront signed URLs using Canned Policy.
// Uses SHA1 hashing and AWS-specific base64 URL-safe encoding.
type CloudFrontURLSigner interface {
	SignURL(resourceURL string, expiresAt time.Time) (signedURL string, err error)
}
```

### Step 4: Add comprehensive tests `internal/crypto/cloudfront_test.go`

- ✓ Signature matches AWS CLI output (deterministic validation)
- ✓ Base64 URL-safe encoding correctness
- ✓ Canned Policy JSON format validation
- ✓ PKCS1 and PKCS8 private key support
- ✓ Error handling for invalid keys

## Benefits

1. **AWS CloudFront specific**: Optimized with SHA1 + Canned Policy
2. **Zero breaking changes**: Existing `RSASigner` untouched
3. **Clear intent**: `CloudFrontURLSigner` name clarifies purpose
4. **Testable**: Deterministic SHA1 allows AWS CLI comparison
5. **Follows adapter pattern**: Consistent with existing code style
6. **Type-safe**: Uses error returns, no panics

## Implementation Checklist

- [ ] Create `internal/crypto/cloudfront.go` with `CloudFrontURLSigner` impl
- [ ] Create `internal/crypto/cloudfront_test.go` with full coverage
- [ ] Add factory function to `pkg/crypto/api.go`
- [ ] Add interface to `pkg/crypto/contracts/contracts.go`
- [ ] Update `README.md` with CloudFront example
- [ ] Bump version to v3.4.0 in `go.mod`
- [ ] Create PR and verify coverage ≥ 96%

## Backward Compatibility

✓ **Fully backward compatible** — only adds new functionality, zero breaking changes
