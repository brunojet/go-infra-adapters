package aws

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign"
)

// CloudFrontSigner signs CloudFront URLs using AWS native API.
type CloudFrontSigner struct {
	signer *sign.URLSigner
}

// NewCloudFrontSignerFromPEM creates a CloudFront signer from PEM-encoded private key.
// keyID is the AWS CloudFront public key ID (e.g., "K31UKMLKEO2DC4").
func NewCloudFrontSignerFromPEM(keyID string, privateKeyPEM []byte) (*CloudFrontSigner, error) {
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("parse private key: invalid PEM data")
	}

	var privKey *rsa.PrivateKey
	if block.Type == "RSA PRIVATE KEY" {
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		privKey = key
	} else {
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		var ok bool
		privKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("parse private key: not RSA")
		}
	}

	signer := sign.NewURLSigner(keyID, privKey)
	return &CloudFrontSigner{signer: signer}, nil
}

// SignURL generates a CloudFront signed URL with Canned Policy format.
// expiresAt is Unix timestamp when the signature expires.
// Format: URL?Expires=TIMESTAMP&Signature=BASE64_SAFE&Key-Pair-Id=KEYID
func (c *CloudFrontSigner) SignURL(ctx context.Context, resourceURL string, expiresAt int64) (string, error) {
	signedURL, err := c.signer.Sign(resourceURL, time.Unix(expiresAt, 0))
	if err != nil {
		return "", fmt.Errorf("sign url: %w", err)
	}
	return signedURL, nil
}
