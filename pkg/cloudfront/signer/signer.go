// Package signer provides a stable public wrapper around the internal CloudFront signer.
// It exposes NewSignerFromPEM and SignURLCanned for generating CloudFront signed URLs
// using a canned policy. The implementation delegates to internal/cloudfront/signer.
package signer

import (
	"time"

	internal "github.com/brunojet/go-infra-adapters/internal/cloudfront/signer"
)

// Signer is a thin public wrapper around the internal CloudFront signer.
type Signer struct {
	s *internal.Signer
}

// NewSignerFromPEM constructs a new Signer from PEM-encoded RSA private key and a keyPairID.
func NewSignerFromPEM(privateKeyPEM []byte, keyPairID string) (*Signer, error) {
	s, err := internal.NewSignerFromPEM(privateKeyPEM, keyPairID)
	if err != nil {
		return nil, err
	}
	return &Signer{s: s}, nil
}

// SignURLCanned signs the given URL using a canned policy that expires at the provided time.
// It returns the signed URL containing `Expires`, `Signature` and `Key-Pair-Id` query params.
func (s *Signer) SignURLCanned(rawURL string, expires time.Time) (string, error) {
	return s.s.SignURLCanned(rawURL, expires)
}
