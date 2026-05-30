// Package crypto exposes RSA key-pair generation and signing/verification
// without leaking internal implementation details.
// Callers work exclusively against the contracts.KeyGenerator, contracts.Signer
// and contracts.Verifier interfaces.
package crypto

import (
	internal "github.com/brunojet/go-infra-adapters/internal/crypto"
	"github.com/brunojet/go-infra-adapters/pkg/crypto/contracts"
)

// NewRSAKeyGenerator returns a KeyGenerator that produces RSA key pairs of the
// given bit size (minimum 2048).
// Complexity: O(1) for construction; O(n) per key generation where n = bits.
func NewRSAKeyGenerator(bits int) contracts.KeyGenerator {
	return internal.NewRSAKeyGenerator(bits)
}

// NewRSASignerFromPEM returns a Signer backed by the PEM-encoded RSA private key.
// Accepts PKCS1 ("RSA PRIVATE KEY") and PKCS8 ("PRIVATE KEY") formats.
// Complexity: O(n) where n = len(privateKeyPEM). Memory: ~1-5 KB for parsed key.
func NewRSASignerFromPEM(privateKeyPEM []byte) (contracts.Signer, error) {
	return internal.NewRSASignerFromPEM(privateKeyPEM)
}

// NewRSAVerifierFromPEM returns a Verifier backed by the PEM-encoded RSA public key
// in PKIX ("PUBLIC KEY") format.
// Complexity: O(n) where n = len(publicKeyPEM). Memory: ~500 bytes for parsed key.
func NewRSAVerifierFromPEM(publicKeyPEM []byte) (contracts.Verifier, error) {
	return internal.NewRSAVerifierFromPEM(publicKeyPEM)
}
