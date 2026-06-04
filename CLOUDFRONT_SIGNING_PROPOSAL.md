# CloudFront Signed URL Implementation Proposal

## Problem Statement

The current `RSASigner` in `internal/crypto/rsa.go` uses **SHA256** hashing with RSA-PKCS1v15, which is optimized for general-purpose signing. However, **AWS CloudFront signed URLs require SHA1** with a specific **Canned Policy** JSON format.

Current gap:
- ✗ No SHA1 support in adapter
- ✗ No CloudFront Canned Policy helper
- ✗ No URL-safe base64 encoding (AWS-specific: `+→-`, `/→~`, `=→_`)

## Proposed Solution

### Option A: Add `CloudFrontSigner` (Recommended)

Create a new type specifically for CloudFront signing with:
- SHA1 hashing
- Canned Policy JSON generation
- AWS-specific base64 URL-safe encoding

**File:** `internal/crypto/cloudfront.go`

```go
package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/brunojet/go-infra-adapters/pkg/crypto/contracts"
)

// CloudFrontCannedPolicy represents AWS CloudFront Canned Policy structure
type CloudFrontCannedPolicy struct {
	Statement []struct {
		Resource  string `json:"Resource"`
		Condition struct {
			DateLessThan struct {
				EpochTime int64 `json:"AWS:EpochTime"`
			} `json:"DateLessThan"`
		} `json:"Condition"`
	} `json:"Statement"`
}

// CloudFrontSigner creates CloudFront signed URLs using Canned Policy
type CloudFrontSigner struct {
	keyPairID  string
	privateKey *rsa.PrivateKey
}

// NewCloudFrontSignerFromPEM creates a CloudFront signer from PEM-encoded RSA private key
func NewCloudFrontSignerFromPEM(keyPairID string, privateKeyPEM []byte) (*CloudFrontSigner, error) {
	priv, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return &CloudFrontSigner{
		keyPairID:  keyPairID,
		privateKey: priv,
	}, nil
}

// SignURL generates CloudFront signed URL with Canned Policy
func (s *CloudFrontSigner) SignURL(resourceURL string, expiresAt time.Time) (string, error) {
	// Create Canned Policy JSON
	policy := fmt.Sprintf(
		`{"Statement":[{"Resource":"%s","Condition":{"DateLessThan":{"AWS:EpochTime":%d}}}]}`,
		resourceURL,
		expiresAt.Unix(),
	)

	// Sign policy with SHA1
	hash := sha1.Sum([]byte(policy))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA1, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign policy: %w", err)
	}

	// Encode signature with AWS-specific base64
	encodedSig := s.encodeBase64URLSafe(signature)

	// Build final URL
	return fmt.Sprintf("%s?Expires=%d&Signature=%s&Key-Pair-Id=%s",
		resourceURL,
		expiresAt.Unix(),
		encodedSig,
		s.keyPairID,
	), nil
}

// encodeBase64URLSafe encodes with AWS-specific replacements
// AWS: + → -, / → ~, = → _
func (s *CloudFrontSigner) encodeBase64URLSafe(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "~")
	encoded = strings.ReplaceAll(encoded, "=", "_")
	return encoded
}
```

**Public API in** `pkg/crypto/api.go`:

```go
// NewCloudFrontSignerFromPEM returns a CloudFront signer for signed URL generation.
// Produces URLs with Canned Policy, SHA1 hashing, and AWS-specific base64 encoding.
func NewCloudFrontSignerFromPEM(keyPairID string, privateKeyPEM []byte) (CloudFrontSigner, error) {
	return internal.NewCloudFrontSignerFromPEM(keyPairID, privateKeyPEM)
}
```

**Interface in** `pkg/crypto/contracts/contracts.go`:

```go
// CloudFrontSigner generates CloudFront signed URLs using Canned Policy.
// Uses SHA1 hashing and AWS-specific base64 URL-safe encoding.
type CloudFrontSigner interface {
	SignURL(resourceURL string, expiresAt time.Time) (signedURL string, err error)
}
```

### Tests

Create `internal/crypto/cloudfront_test.go` with:
- ✓ Signature matches AWS CLI output for known input
- ✓ Base64 URL-safe encoding correctness
- ✓ Canned Policy JSON format validation
- ✓ PKCS1 and PKCS8 private key support

## Benefits

1. **AWS CloudFront specific**: Optimized for CloudFront signed URLs
2. **Zero breaking changes**: Existing `RSASigner` unchanged
3. **Clear intent**: `CloudFrontSigner` makes purpose obvious
4. **Testable**: Deterministic SHA1 allows AWS CLI comparison
5. **Type-safe**: Follows adapter's contract pattern

## Implementation Checklist

- [ ] Create `internal/crypto/cloudfront.go`
- [ ] Add tests with AWS CLI comparison
- [ ] Update `pkg/crypto/api.go` with public factory
- [ ] Add `CloudFrontSigner` interface to `contracts/contracts.go`
- [ ] Document in README with example
- [ ] Update version to 3.4.0

## Backward Compatibility

✓ **Fully backward compatible** - only adds new functionality, no existing changes
