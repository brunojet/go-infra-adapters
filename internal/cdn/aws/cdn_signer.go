package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	cryptointernal "github.com/brunojet/go-infra-adapters/v4/internal/crypto"
	"github.com/brunojet/go-infra-adapters/v4/pkg/crypto/contracts"
)

// CloudFrontSigner signs CloudFront URLs using AWS Canned Policy format.
// Uses SHA1 hashing with AWS-specific base64 URL-safe encoding.
type CloudFrontSigner struct {
	signer *cryptointernal.RSASigner
	keyID  string
}

// NewCloudFrontSignerFromPEM creates a CloudFront signer from PEM-encoded private key.
// keyID is the AWS CloudFront public key ID (e.g., "K31UKMLKEO2DC4").
// Complexity: O(n) where n = len(privateKeyPEM). Memory: ~1-5 KB for parsed key.
func NewCloudFrontSignerFromPEM(keyID string, privateKeyPEM []byte) (*CloudFrontSigner, error) {
	signer, err := cryptointernal.NewRSASignerFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return &CloudFrontSigner{
		signer: signer,
		keyID:  keyID,
	}, nil
}

// SignURL generates a CloudFront signed URL with Canned Policy format.
// expiresAt is Unix timestamp when the signature expires.
// Format: URL?Expires=TIMESTAMP&Signature=BASE64_SAFE&Key-Pair-Id=KEYID
func (c *CloudFrontSigner) SignURL(ctx context.Context, resourceURL string, expiresAt int64) (string, error) {
	// Canned Policy JSON format (exact AWS CloudFront specification)
	policy := fmt.Sprintf(
		`{"Statement":[{"Resource":%q,"Condition":{"DateLessThan":{"AWS:EpochTime":%d}}}]}`,
		resourceURL,
		expiresAt,
	)

	// Sign with SHA1 (CloudFront requirement)
	signature, err := c.signer.Sign(ctx, contracts.SHA1, []byte(policy))
	if err != nil {
		return "", fmt.Errorf("sign policy: %w", err)
	}

	// AWS-specific base64 URL-safe encoding: + → -, / → ~, = → _
	encodedSig := c.base64URLSafe(signature)

	// Build signed URL with CloudFront parameters
	return fmt.Sprintf("%s?Expires=%d&Signature=%s&Key-Pair-Id=%s",
		resourceURL,
		expiresAt,
		encodedSig,
		c.keyID,
	), nil
}

// base64URLSafe encodes with AWS-specific character replacements.
// Standard base64 characters are mapped: + → -, / → ~, = → _
func (c *CloudFrontSigner) base64URLSafe(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "~")
	encoded = strings.ReplaceAll(encoded, "=", "_")
	return encoded
}
