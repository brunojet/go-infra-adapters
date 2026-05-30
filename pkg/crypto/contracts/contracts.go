// Package contracts defines the public cryptographic contracts.
// Implementations live in internal/crypto; callers import via pkg/crypto.
package contracts

import "context"

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

// Signer signs arbitrary byte payloads.
// Hashing is the implementation's responsibility — callers pass raw content bytes.
// The signing algorithm (e.g. RSA-PKCS1v15-SHA256) is opaque to the caller.
type Signer interface {
	Sign(ctx context.Context, payload []byte) (signature []byte, err error)
}

// Verifier verifies signatures produced by the matching Signer.
// Returns nil when the signature is valid.
type Verifier interface {
	Verify(ctx context.Context, payload, signature []byte) error
}
