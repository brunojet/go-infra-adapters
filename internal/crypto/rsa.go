// Package crypto provides cryptographic primitives.
// The package name shadows the stdlib "crypto" package; stdlib sub-packages
// (crypto/rand, crypto/rsa, etc.) are imported without conflict. The top-level
// stdlib crypto package is aliased as stdcrypto where needed.
package crypto

import (
	stdcrypto "crypto"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"

	"context"

	"github.com/brunojet/go-infra-adapters/pkg/crypto/contracts"
)

// cryptoRandReader is the random source used by Generate and Sign.
// Swapped in tests to trigger error-path coverage.
var cryptoRandReader io.Reader = cryptorand.Reader

// rsaSignPKCS1v15 wraps rsa.SignPKCS1v15; replaceable in tests to force errors.
var rsaSignPKCS1v15 = rsa.SignPKCS1v15

// x509MarshalPKIXPublicKey wraps x509.MarshalPKIXPublicKey; replaceable in tests to force errors.
var x509MarshalPKIXPublicKey = x509.MarshalPKIXPublicKey

// ── Key generator ────────────────────────────────────────────────────────────

// RSAKeyGenerator generates RSA key pairs.
type RSAKeyGenerator struct {
	bits int
}

// NewRSAKeyGenerator creates a KeyGenerator that produces RSA key pairs of the
// given bit size. Panics for sizes below 2048 to prevent insecure keys.
func NewRSAKeyGenerator(bits int) *RSAKeyGenerator {
	if bits < 2048 {
		panic(fmt.Sprintf("RSA key size must be at least 2048 bits, got %d", bits))
	}
	return &RSAKeyGenerator{bits: bits}
}

// Generate creates a new RSA key pair.
// It encodes the private key in PKCS1 PEM format and the public key in PKIX PEM
// format, and computes the SHA-256 fingerprint of the DER-encoded public key.
func (g *RSAKeyGenerator) Generate(_ context.Context) (*contracts.KeyPair, error) {
	key, err := rsa.GenerateKey(cryptoRandReader, g.bits)
	if err != nil {
		return nil, fmt.Errorf("rsa generate: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	pubDER, err := x509MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	fp := sha256.Sum256(pubDER)

	return &contracts.KeyPair{
		PrivatePEM:  privPEM,
		PublicPEM:   pubPEM,
		Fingerprint: hex.EncodeToString(fp[:]),
	}, nil
}

// ── Signer ───────────────────────────────────────────────────────────────────

// RSASigner signs payloads using RSA-PKCS1v15 with SHA-256.
type RSASigner struct {
	priv *rsa.PrivateKey
}

// NewRSASignerFromPEM creates an RSASigner from a PEM-encoded RSA private key.
// Accepts PKCS1 ("RSA PRIVATE KEY") and PKCS8 ("PRIVATE KEY") formats.
func NewRSASignerFromPEM(privateKeyPEM []byte) (*RSASigner, error) {
	priv, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}
	return &RSASigner{priv: priv}, nil
}

// Sign hashes the payload with SHA-256 and produces an RSA-PKCS1v15 signature.
func (s *RSASigner) Sign(_ context.Context, payload []byte) ([]byte, error) {
	h := sha256.Sum256(payload)
	sig, err := rsaSignPKCS1v15(cryptoRandReader, s.priv, stdcrypto.SHA256, h[:])
	if err != nil {
		return nil, fmt.Errorf("rsa sign: %w", err)
	}
	return sig, nil
}

// ── Verifier ─────────────────────────────────────────────────────────────────

// RSAVerifier verifies RSA-PKCS1v15-SHA256 signatures.
type RSAVerifier struct {
	pub *rsa.PublicKey
}

// NewRSAVerifierFromPEM creates an RSAVerifier from a PEM-encoded RSA public key
// in PKIX ("PUBLIC KEY") format.
func NewRSAVerifierFromPEM(publicKeyPEM []byte) (*RSAVerifier, error) {
	block, _ := pem.Decode(publicKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM data")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	rsaPub, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("unsupported public key type: %T", key)
	}
	return &RSAVerifier{pub: rsaPub}, nil
}

// Verify reports nil if signature is a valid RSA-PKCS1v15-SHA256 signature over payload.
func (v *RSAVerifier) Verify(_ context.Context, payload, signature []byte) error {
	h := sha256.Sum256(payload)
	if err := rsa.VerifyPKCS1v15(v.pub, stdcrypto.SHA256, h[:], signature); err != nil {
		return fmt.Errorf("verify signature: %w", err)
	}
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

// parseRSAPrivateKey decodes a PEM-encoded RSA private key.
// Tries PKCS1 first (block type "RSA PRIVATE KEY"), then PKCS8 ("PRIVATE KEY").
func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM data")
	}
	if block.Type == "RSA PRIVATE KEY" {
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		p, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse private key (pkcs8: %v, pkcs1: %v)", err, err2)
		}
		return p, nil
	}
	rsaKey, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("unsupported private key type: %T", k)
	}
	return rsaKey, nil
}
