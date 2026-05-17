package signer

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Signer signs CloudFront URLs using an RSA private key.
type Signer struct {
	priv      *rsa.PrivateKey
	keyPairID string
	hashAlgo  string
}

// NewSignerFromPEM creates a Signer from PEM-encoded RSA private key and a keyPairID.
func NewSignerFromPEM(privateKeyPEM []byte, keyPairID string) (*Signer, error) {
	if keyPairID == "" {
		return nil, fmt.Errorf("keyPairID is required")
	}
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM data")
	}
	var priv *rsa.PrivateKey
	// Try PKCS1
	if block.Type == "RSA PRIVATE KEY" {
		p, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS1 private key: %w", err)
		}
		priv = p
	} else {
		// Try PKCS8
		k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			// fallback to PKCS1 attempt
			p, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err2 != nil {
				return nil, fmt.Errorf("failed to parse private key (pkcs8: %v, pkcs1: %v)", err, err2)
			}
			priv = p
		} else {
			switch t := k.(type) {
			case *rsa.PrivateKey:
				priv = t
			default:
				return nil, fmt.Errorf("unsupported private key type: %T", k)
			}
		}
	}
	return &Signer{priv: priv, keyPairID: keyPairID, hashAlgo: "RSA-SHA256"}, nil
}

// SignURLCanned signs the given URL using a canned policy that expires at the provided time.
// It returns the signed URL with query parameters `Expires`, `Signature`, and `Key-Pair-Id`.
func (s *Signer) SignURLCanned(rawURL string, expires time.Time) (string, error) {
	if s == nil || s.priv == nil {
		return "", fmt.Errorf("invalid signer")
	}
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	epoch := expires.Unix()
	// Build the canned policy (JSON) as CloudFront expects for signature input.
	policy := fmt.Sprintf(`{"Statement":[{"Resource":%q,"Condition":{"DateLessThan":{"AWS:EpochTime":%d}}}]}`, rawURL, epoch)

	hash := sha256.Sum256([]byte(policy))
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.priv, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("signing: %w", err)
	}
	enc := encodeURLSafeSignature(sig)
	q := parsed.Query()
	q.Set("Expires", fmt.Sprintf("%d", epoch))
	q.Set("Signature", enc)
	q.Set("Key-Pair-Id", s.keyPairID)
	q.Set("Hash-Algorithm", s.hashAlgo)
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

// encodeURLSafeSignature encodes a raw signature into the CloudFront URL-safe variant.
// AWS CloudFront historically replaces characters to make the signature safe for URLs.
func encodeURLSafeSignature(sig []byte) string {
	s := base64.StdEncoding.EncodeToString(sig)
	s = strings.ReplaceAll(s, "+", "-")
	s = strings.ReplaceAll(s, "=", "_")
	s = strings.ReplaceAll(s, "/", "~")
	return s
}

// decodeURLSafeSignature reverses encodeURLSafeSignature. Used in tests.
func decodeURLSafeSignature(enc string) ([]byte, error) {
	s := strings.ReplaceAll(enc, "-", "+")
	s = strings.ReplaceAll(s, "_", "=")
	s = strings.ReplaceAll(s, "~", "/")
	return base64.StdEncoding.DecodeString(s)
}
