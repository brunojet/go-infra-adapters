package signer

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
)

func generateTestKey(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	b := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: b}
	return pem.EncodeToMemory(block)
}

func TestNewSignerFromPEM_and_SignURLCanned(t *testing.T) {
	pemKey := generateTestKey(t)
	s, err := NewSignerFromPEM(pemKey, "KP-123")
	if err != nil {
		t.Fatalf("NewSignerFromPEM failed: %v", err)
	}

	expires := time.Now().Add(1 * time.Hour)
	signed, err := s.SignURLCanned("https://example.com/path/file.txt", expires)
	if err != nil {
		t.Fatalf("SignURLCanned failed: %v", err)
	}
	u, err := url.Parse(signed)
	if err != nil {
		t.Fatalf("parse signed url: %v", err)
	}
	q := u.Query()
	if q.Get("Key-Pair-Id") != "KP-123" {
		t.Fatalf("missing or wrong Key-Pair-Id: %s", q.Get("Key-Pair-Id"))
	}
	if q.Get("Signature") == "" {
		t.Fatalf("missing Signature")
	}
	if q.Get("Expires") == "" {
		t.Fatalf("missing Expires")
	}
	if q.Get("Hash-Algorithm") != "SHA256" {
		t.Fatalf("unexpected Hash-Algorithm: %s", q.Get("Hash-Algorithm"))
	}
	// signature should be URL-safe (no +, /, =)
	sig := q.Get("Signature")
	if strings.ContainsAny(sig, "+/=") {
		t.Fatalf("signature contains unsafe chars: %s", sig)
	}
}

func TestEncodeDecodeURLSafeSignature(t *testing.T) {
	// use a small byte slice to ensure + and / may appear in base64
	raw := []byte{0xff, 0x00, 0x01, 0x02, 0x03}
	enc := encodeURLSafeSignature(raw)
	dec, err := decodeURLSafeSignature(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(dec) != len(raw) {
		t.Fatalf("decoded length mismatch")
	}
	for i := range dec {
		if dec[i] != raw[i] {
			t.Fatalf("byte %d mismatch", i)
		}
	}
}

func TestNewSignerFromPEM_ErrorsAndPKCS8AndUnsupported(t *testing.T) {
	// empty keyPairID
	pemKey := generateTestKey(t)
	if _, err := NewSignerFromPEM(pemKey, ""); err == nil || !strings.Contains(err.Error(), "keyPairID is required") {
		t.Fatalf("expected keyPairID error, got %v", err)
	}

	// invalid PEM
	if _, err := NewSignerFromPEM([]byte("not-a-pem"), "KP"); err == nil || !strings.Contains(err.Error(), "invalid PEM data") {
		t.Fatalf("expected invalid PEM error, got %v", err)
	}

	// PKCS8 RSA key
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	if err != nil {
		t.Fatalf("marshal pkcs8 rsa: %v", err)
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	pemData := pem.EncodeToMemory(block)
	s, err := NewSignerFromPEM(pemData, "KP-PK8")
	if err != nil {
		t.Fatalf("expected pkcs8 rsa to parse: %v", err)
	}
	// ensure SignURLCanned works for PKCS8 RSA
	if _, err := s.SignURLCanned("https://example.org/file", time.Now().Add(5*time.Minute)); err != nil {
		t.Fatalf("signing with pkcs8 rsa failed: %v", err)
	}

	// unsupported private key type (ECDSA)
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa: %v", err)
	}
	der2, err := x509.MarshalPKCS8PrivateKey(ecdsaKey)
	if err != nil {
		t.Fatalf("marshal pkcs8 ecdsa: %v", err)
	}
	block2 := &pem.Block{Type: "PRIVATE KEY", Bytes: der2}
	pemEcdsa := pem.EncodeToMemory(block2)
	if _, err := NewSignerFromPEM(pemEcdsa, "KP-EC"); err == nil || !strings.Contains(err.Error(), "unsupported private key type") {
		t.Fatalf("expected unsupported key type error, got %v", err)
	}

	// both parsers fail
	bad := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("badbytes")})
	if _, err := NewSignerFromPEM(bad, "KP-BAD"); err == nil || !strings.Contains(err.Error(), "failed to parse private key") {
		t.Fatalf("expected combined parse error, got %v", err)
	}
}

func TestSignURLCanned_ErrorsAndSignatureVerification(t *testing.T) {
	// nil signer
	var nilSigner *Signer
	if _, err := nilSigner.SignURLCanned("https://example.com", time.Now().Add(1*time.Hour)); err == nil || !strings.Contains(err.Error(), "invalid signer") {
		t.Fatalf("expected invalid signer error, got %v", err)
	}

	pemKey := generateTestKey(t)
	s, err := NewSignerFromPEM(pemKey, "KP-1")
	if err != nil {
		t.Fatalf("NewSignerFromPEM: %v", err)
	}

	// empty URL
	if _, err := s.SignURLCanned("", time.Now().Add(10*time.Minute)); err == nil || !strings.Contains(err.Error(), "empty URL") {
		t.Fatalf("expected empty URL error, got %v", err)
	}

	// invalid URL parse
	if _, err := s.SignURLCanned("http://[::1", time.Now().Add(10*time.Minute)); err == nil || !strings.Contains(err.Error(), "parse url:") {
		t.Fatalf("expected parse url error, got %v", err)
	}

	// verify produced signature is valid
	raw := "https://example.com/path/file.txt?x=1"
	expires := time.Now().Add(1 * time.Hour)
	signed, err := s.SignURLCanned(raw, expires)
	if err != nil {
		t.Fatalf("SignURLCanned: %v", err)
	}
	u, err := url.Parse(signed)
	if err != nil {
		t.Fatalf("parse signed url: %v", err)
	}
	q := u.Query()
	sigEnc := q.Get("Signature")
	if sigEnc == "" {
		t.Fatalf("missing signature param")
	}
	decoded, err := decodeURLSafeSignature(sigEnc)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}

	policy := fmt.Sprintf(`{"Statement":[{"Resource":%q,"Condition":{"DateLessThan":{"AWS:EpochTime":%d}}}]}`, raw, expires.Unix())
	h := sha256.Sum256([]byte(policy))
	if err := rsa.VerifyPKCS1v15(&s.priv.PublicKey, crypto.SHA256, h[:], decoded); err != nil {
		t.Fatalf("signature verify failed: %v", err)
	}

	// Expires param should match
	if q.Get("Expires") == "" {
		t.Fatalf("missing Expires param")
	}
}
