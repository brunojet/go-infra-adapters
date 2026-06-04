// Package contracts defines the public cryptographic contracts.
// Implementations live in internal/crypto; callers import via pkg/crypto.
package contracts

import "context"

// HashAlgorithm specifies the hash algorithm for signing and verification.
type HashAlgorithm int

const (
	// SHA256 uses SHA-256 hashing (default for general-purpose signing).
	SHA256 HashAlgorithm = iota
	// SHA1 uses SHA-1 hashing (required for AWS CloudFront signed URLs).
	SHA1
	// SHA512 uses SHA-512 hashing.
	SHA512
)

// KeyPair holds a generated asymmetric key pair in PEM format.
type KeyPair struct {
	// PrivatePEM is the PEM-encoded private key.
	// For RSA: PKCS1 "RSA PRIVATE KEY" or PKCS8 "PRIVATE KEY".
	PrivatePEM []byte
	// PublicPEM is the PEM-encoded public key (PKIX "PUBLIC KEY").
	PublicPEM []byte
	// Fingerprint is the hex-encoded SHA-256 digest of the DER-encoded public key.
	// Serves as a stable, compact identifier — safe to log and store.
	Fingerprint string
}

// KeyGenerator generates asymmetric key pairs.
// The algorithm and key size are determined by the implementation.
type KeyGenerator interface {
	Generate(ctx context.Context) (*KeyPair, error)
}

// Signer signs arbitrary byte payloads with the specified hash algorithm.
// The signing algorithm (e.g. RSA-PKCS1v15) is implementation-specific.
// hashAlgo specifies which hash to use (SHA256, SHA1, SHA512, etc).
type Signer interface {
	Sign(ctx context.Context, hashAlgo HashAlgorithm, payload []byte) (signature []byte, err error)
}

// Verifier verifies signatures produced by the matching Signer with the specified hash algorithm.
// Returns nil when the signature is valid.
type Verifier interface {
	Verify(ctx context.Context, hashAlgo HashAlgorithm, payload, signature []byte) error
}
