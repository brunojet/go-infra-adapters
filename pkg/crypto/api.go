// Package crypto exposes RSA key-pair generation and signing/verification
// without leaking internal implementation details.
// Callers work exclusively against the contracts.KeyGenerator, contracts.Signer
// and contracts.Verifier interfaces.
package crypto

import (
	"github.com/brunojet/go-infra-adapters/v4/internal/crypto"
	"github.com/brunojet/go-infra-adapters/v4/pkg/crypto/contracts"
)

type (
	KeyGenerator = contracts.KeyGenerator
	Signer       = contracts.Signer
	Verifier     = contracts.Verifier
)

// Hash algorithm constants
const (
	SHA1   = contracts.SHA1
	SHA256 = contracts.SHA256
	SHA512 = contracts.SHA512
)

// NewRSAKeyGenerator returns a KeyGenerator that produces RSA key pairs of the
// given bit size (minimum 2048).
// Complexity: O(1) for construction; O(n) per key generation where n = bits.
func NewRSAKeyGenerator(bits int) KeyGenerator {
	return crypto.NewRSAKeyGenerator(bits)
}

// NewRSASignerFromPEM returns a Signer backed by the PEM-encoded RSA private key.
// Accepts PKCS1 ("RSA PRIVATE KEY") and PKCS8 ("PRIVATE KEY") formats.
// Complexity: O(n) where n = len(privateKeyPEM). Memory: ~1-5 KB for parsed key.
func NewRSASignerFromPEM(privateKeyPEM []byte) (Signer, error) {
	return crypto.NewRSASignerFromPEM(privateKeyPEM)
}

// NewRSAVerifierFromPEM returns a Verifier backed by the PEM-encoded RSA public key
// in PKIX ("PUBLIC KEY") format.
// Complexity: O(n) where n = len(publicKeyPEM). Memory: ~500 bytes for parsed key.
func NewRSAVerifierFromPEM(publicKeyPEM []byte) (Verifier, error) {
	return crypto.NewRSAVerifierFromPEM(publicKeyPEM)
}
