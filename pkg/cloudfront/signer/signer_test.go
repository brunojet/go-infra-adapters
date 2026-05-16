package signer_test

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
	"testing"
	"time"

	pkgsigner "github.com/brunojet/go-infra-adapters/pkg/cloudfront/signer"
)

func TestSigner_SignURLCanned(t *testing.T) {
	// Generate RSA key for test
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	pemBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}
	pemData := pem.EncodeToMemory(pemBlock)

	keyPairID := "K_TEST"
	s, err := pkgsigner.NewSignerFromPEM(pemData, keyPairID)
	if err != nil {
		t.Fatalf("NewSignerFromPEM: %v", err)
	}

	rawURL := "https://d111111abcdef8.cloudfront.net/images/image.jpg?size=large"
	expires := time.Now().Add(1 * time.Hour)
	signed, err := s.SignURLCanned(rawURL, expires)
	if err != nil {
		t.Fatalf("SignURLCanned: %v", err)
	}

	u, err := url.Parse(signed)
	if err != nil {
		t.Fatalf("parse signed url: %v", err)
	}
	q := u.Query()

	if got := q.Get("Key-Pair-Id"); got != keyPairID {
		t.Fatalf("Key-Pair-Id mismatch: got %q want %q", got, keyPairID)
	}
	if got := q.Get("Hash-Algorithm"); got != "RSA-SHA256" {
		t.Fatalf("Hash-Algorithm mismatch: got %q want %q", got, "RSA-SHA256")
	}

	sigEnc := q.Get("Signature")
	if sigEnc == "" {
		t.Fatalf("missing Signature param")
	}

	// decode CloudFront URL-safe base64 variant
	s2 := strings.ReplaceAll(sigEnc, "-", "+")
	s2 = strings.ReplaceAll(s2, "_", "=")
	s2 = strings.ReplaceAll(s2, "~", "/")
	sig, err := base64.StdEncoding.DecodeString(s2)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}

	epoch := expires.Unix()
	policy := fmt.Sprintf(`{"Statement":[{"Resource":%q,"Condition":{"DateLessThan":{"AWS:EpochTime":%d}}}]}`, rawURL, epoch)
	h := sha256.Sum256([]byte(policy))

	if err := rsa.VerifyPKCS1v15(&priv.PublicKey, crypto.SHA256, h[:], sig); err != nil {
		t.Fatalf("verify signature: %v", err)
	}
}
