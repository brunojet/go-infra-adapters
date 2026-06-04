// Package cdn exposes CDN adapters and signing functionality
// without leaking internal implementation details.
// Callers work exclusively against the contracts.CdnAdapter and contracts.URLSigner interfaces.
package cdn

import (
	internal "github.com/brunojet/go-infra-adapters/v3/internal/cdn/aws"
	"github.com/brunojet/go-infra-adapters/v3/pkg/cdn/contracts"
)

// NewCloudFrontSignerFromPEM returns a URLSigner for AWS CloudFront signed URLs.
// Uses SHA1 hashing with Canned Policy format and AWS-specific base64 URL-safe encoding.
// keyID: AWS CloudFront public key ID (e.g., "K31UKMLKEO2DC4").
// Complexity: O(n) where n = len(privateKeyPEM). Memory: ~1-5 KB for parsed key.
func NewCloudFrontSignerFromPEM(keyID string, privateKeyPEM []byte) (contracts.URLSigner, error) {
	return internal.NewCloudFrontSignerFromPEM(keyID, privateKeyPEM)
}
