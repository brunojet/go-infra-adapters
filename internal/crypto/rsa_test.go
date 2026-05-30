package crypto

import (
	"bytes"
	"context"
	stdcrypto "crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/brunojet/go-infra-adapters/pkg/crypto/contracts"
)

var bg = context.Background()

// ── KeyGenerator ─────────────────────────────────────────────────────────────

func TestRSAKeyGenerator_Generate(t *testing.T) {
	kp, err := NewRSAKeyGenerator(2048).Generate(bg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !bytes.Contains(kp.PrivatePEM, []byte("RSA PRIVATE KEY")) {
		t.Fatalf("expected RSA PRIVATE KEY block in PrivatePEM")
	}
	if !bytes.Contains(kp.PublicPEM, []byte("PUBLIC KEY")) {
		t.Fatalf("expected PUBLIC KEY block in PublicPEM")
	}
	if len(kp.Fingerprint) != 64 {
		t.Fatalf("expected 64-char hex fingerprint, got len=%d", len(kp.Fingerprint))
	}
}

func TestRSAKeyGenerator_UniquePairsPerCall(t *testing.T) {
	g := NewRSAKeyGenerator(2048)
	kp1, _ := g.Generate(bg)
	kp2, _ := g.Generate(bg)
	if kp1.Fingerprint == kp2.Fingerprint {
		t.Fatal("two Generate calls returned the same fingerprint")
	}
}

func TestRSAKeyGenerator_PanicBelowMinBits(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for bits < 2048")
		}
	}()
	NewRSAKeyGenerator(1024)
}

func TestRSAKeyGenerator_ImplementsInterface(t *testing.T) {
	var _ contracts.KeyGenerator = NewRSAKeyGenerator(2048)
}

// ── Signer ───────────────────────────────────────────────────────────────────

func TestRSASigner_Sign(t *testing.T) {
	kp := mustGenKP(t)
	s, err := NewRSASignerFromPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("NewRSASignerFromPEM: %v", err)
	}
	sig, err := s.Sign(bg, []byte("hello world"))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if len(sig) == 0 {
		t.Fatal("expected non-empty signature")
	}
}

func TestRSASigner_DifferentPayloads_DifferentSignatures(t *testing.T) {
	kp := mustGenKP(t)
	s, _ := NewRSASignerFromPEM(kp.PrivatePEM)
	sig1, _ := s.Sign(bg, []byte("payload-a"))
	sig2, _ := s.Sign(bg, []byte("payload-b"))
	if bytes.Equal(sig1, sig2) {
		t.Fatal("different payloads produced the same signature")
	}
}

func TestRSASigner_AcceptsPKCS8(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa: %v", err)
	}
	der, _ := x509.MarshalPKCS8PrivateKey(rsaKey)
	pemData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	s, err := NewRSASignerFromPEM(pemData)
	if err != nil {
		t.Fatalf("expected PKCS8 to parse: %v", err)
	}
	if _, err := s.Sign(bg, []byte("data")); err != nil {
		t.Fatalf("Sign with PKCS8 key: %v", err)
	}
}

func TestRSASigner_InvalidPEM(t *testing.T) {
	_, err := NewRSASignerFromPEM([]byte("not-a-pem"))
	if err == nil || !strings.Contains(err.Error(), "invalid PEM") {
		t.Fatalf("expected invalid PEM error, got %v", err)
	}
}

func TestRSASigner_UnsupportedKeyType(t *testing.T) {
	ecKey, _ := ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
	der, _ := x509.MarshalPKCS8PrivateKey(ecKey)
	pemData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	_, err := NewRSASignerFromPEM(pemData)
	if err == nil || !strings.Contains(err.Error(), "unsupported private key type") {
		t.Fatalf("expected unsupported key type error, got %v", err)
	}
}

func TestRSASigner_ImplementsInterface(t *testing.T) {
	kp := mustGenKP(t)
	s, _ := NewRSASignerFromPEM(kp.PrivatePEM)
	var _ contracts.Signer = s
}

// ── Verifier ─────────────────────────────────────────────────────────────────

func TestRSAVerifier_Verify_Valid(t *testing.T) {
	kp := mustGenKP(t)
	signer, _ := NewRSASignerFromPEM(kp.PrivatePEM)
	verifier, err := NewRSAVerifierFromPEM(kp.PublicPEM)
	if err != nil {
		t.Fatalf("NewRSAVerifierFromPEM: %v", err)
	}
	payload := []byte("content to sign")
	sig, _ := signer.Sign(bg, payload)
	if err := verifier.Verify(bg, payload, sig); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestRSAVerifier_Verify_TamperedPayload(t *testing.T) {
	kp := mustGenKP(t)
	signer, _ := NewRSASignerFromPEM(kp.PrivatePEM)
	verifier, _ := NewRSAVerifierFromPEM(kp.PublicPEM)

	sig, _ := signer.Sign(bg, []byte("original"))
	if err := verifier.Verify(bg, []byte("tampered"), sig); err == nil {
		t.Fatal("expected error for tampered payload")
	}
}

func TestRSAVerifier_Verify_TamperedSignature(t *testing.T) {
	kp := mustGenKP(t)
	signer, _ := NewRSASignerFromPEM(kp.PrivatePEM)
	verifier, _ := NewRSAVerifierFromPEM(kp.PublicPEM)

	sig, _ := signer.Sign(bg, []byte("data"))
	sig[0] ^= 0xFF // flip bits in first byte
	if err := verifier.Verify(bg, []byte("data"), sig); err == nil {
		t.Fatal("expected error for tampered signature")
	}
}

func TestRSAVerifier_Verify_WrongKey(t *testing.T) {
	kp1 := mustGenKP(t)
	kp2 := mustGenKP(t)
	signer, _ := NewRSASignerFromPEM(kp1.PrivatePEM)
	verifier, _ := NewRSAVerifierFromPEM(kp2.PublicPEM) // different key pair

	sig, _ := signer.Sign(bg, []byte("data"))
	if err := verifier.Verify(bg, []byte("data"), sig); err == nil {
		t.Fatal("expected error when verifying with wrong public key")
	}
}

func TestRSAVerifier_InvalidPEM(t *testing.T) {
	_, err := NewRSAVerifierFromPEM([]byte("not-a-pem"))
	if err == nil || !strings.Contains(err.Error(), "invalid PEM") {
		t.Fatalf("expected invalid PEM error, got %v", err)
	}
}

func TestRSAVerifier_ImplementsInterface(t *testing.T) {
	kp := mustGenKP(t)
	v, _ := NewRSAVerifierFromPEM(kp.PublicPEM)
	var _ contracts.Verifier = v
}

// ── error-injection tests ─────────────────────────────────────────────────────

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, errors.New("rand error") }

func TestGenerate_MarshalPublicKeyError(t *testing.T) {
	orig := x509MarshalPKIXPublicKey
	x509MarshalPKIXPublicKey = func(_ any) ([]byte, error) {
		return nil, errors.New("injected marshal error")
	}
	defer func() { x509MarshalPKIXPublicKey = orig }()

	_, err := NewRSAKeyGenerator(2048).Generate(bg)
	if err == nil || !strings.Contains(err.Error(), "marshal public key") {
		t.Fatalf("expected marshal public key error, got %v", err)
	}
}

func TestGenerate_RandError(t *testing.T) {
	orig := cryptoRandReader
	cryptoRandReader = failReader{}
	defer func() { cryptoRandReader = orig }()

	_, err := NewRSAKeyGenerator(2048).Generate(bg)
	if err == nil || !strings.Contains(err.Error(), "rsa generate") {
		t.Fatalf("expected rsa generate error, got %v", err)
	}
}

func TestSign_RandError(t *testing.T) {
	kp := mustGenKP(t)
	s, _ := NewRSASignerFromPEM(kp.PrivatePEM)

	orig := rsaSignPKCS1v15
	rsaSignPKCS1v15 = func(_ io.Reader, _ *rsa.PrivateKey, _ stdcrypto.Hash, _ []byte) ([]byte, error) {
		return nil, errors.New("injected sign error")
	}
	defer func() { rsaSignPKCS1v15 = orig }()

	_, err := s.Sign(bg, []byte("data"))
	if err == nil || !strings.Contains(err.Error(), "rsa sign") {
		t.Fatalf("expected rsa sign error, got %v", err)
	}
}

func TestRSAVerifier_InvalidDER(t *testing.T) {
	pemData := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("garbage")})
	_, err := NewRSAVerifierFromPEM(pemData)
	if err == nil || !strings.Contains(err.Error(), "parse public key") {
		t.Fatalf("expected parse public key error, got %v", err)
	}
}

func TestRSAVerifier_UnsupportedPublicKeyType(t *testing.T) {
	ecKey, _ := ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
	der, _ := x509.MarshalPKIXPublicKey(&ecKey.PublicKey)
	pemData := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	_, err := NewRSAVerifierFromPEM(pemData)
	if err == nil || !strings.Contains(err.Error(), "unsupported public key type") {
		t.Fatalf("expected unsupported public key type error, got %v", err)
	}
}

func TestParseRSAPrivateKey_PKCS1BytesInNonRSABlock(t *testing.T) {
	// PKCS1 bytes wrapped in a "PRIVATE KEY" block:
	// ParsePKCS8 will fail, fallback ParsePKCS1 will succeed.
	rsaKey, _ := rsa.GenerateKey(cryptorand.Reader, 2048)
	der := x509.MarshalPKCS1PrivateKey(rsaKey)
	pemData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	got, err := parseRSAPrivateKey(pemData)
	if err != nil || got == nil {
		t.Fatalf("expected PKCS1 fallback to succeed: %v", err)
	}
}

func TestParseRSAPrivateKey_BothFail(t *testing.T) {
	// Garbage bytes in a "PRIVATE KEY" block: both PKCS8 and PKCS1 fail.
	pemData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("garbage")})

	_, err := parseRSAPrivateKey(pemData)
	if err == nil || !strings.Contains(err.Error(), "failed to parse private key") {
		t.Fatalf("expected both-fail error, got %v", err)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mustGenKP(t *testing.T) *contracts.KeyPair {
	t.Helper()
	kp, err := NewRSAKeyGenerator(2048).Generate(bg)
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}
	return kp
}
