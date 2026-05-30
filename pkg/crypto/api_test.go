package crypto

import (
	"context"
	"testing"
)

var bg = context.Background()

func TestNewRSAKeyGenerator(t *testing.T) {
	gen := NewRSAKeyGenerator(2048)
	if gen == nil {
		t.Fatal("expected non-nil KeyGenerator")
	}

	var _ = gen
}

func TestNewRSAKeyGenerator_Generate(t *testing.T) {
	gen := NewRSAKeyGenerator(2048)
	kp, err := gen.Generate(bg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if kp == nil {
		t.Fatal("expected non-nil KeyPair")
	}
	if len(kp.PrivatePEM) == 0 {
		t.Fatal("expected non-empty PrivatePEM")
	}
	if len(kp.PublicPEM) == 0 {
		t.Fatal("expected non-empty PublicPEM")
	}
}

func TestNewRSASignerFromPEM(t *testing.T) {
	gen := NewRSAKeyGenerator(2048)
	kp, _ := gen.Generate(bg)

	signer, err := NewRSASignerFromPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("NewRSASignerFromPEM: %v", err)
	}
	if signer == nil {
		t.Fatal("expected non-nil Signer")
	}

	var _ = signer
}

func TestNewRSASignerFromPEM_InvalidPEM(t *testing.T) {
	_, err := NewRSASignerFromPEM([]byte("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestNewRSAVerifierFromPEM(t *testing.T) {
	gen := NewRSAKeyGenerator(2048)
	kp, _ := gen.Generate(bg)

	verifier, err := NewRSAVerifierFromPEM(kp.PublicPEM)
	if err != nil {
		t.Fatalf("NewRSAVerifierFromPEM: %v", err)
	}
	if verifier == nil {
		t.Fatal("expected non-nil Verifier")
	}

	var _ = verifier
}

func TestNewRSAVerifierFromPEM_InvalidPEM(t *testing.T) {
	_, err := NewRSAVerifierFromPEM([]byte("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestRSA_SignAndVerify(t *testing.T) {
	gen := NewRSAKeyGenerator(2048)
	kp, _ := gen.Generate(bg)

	signer, _ := NewRSASignerFromPEM(kp.PrivatePEM)
	verifier, _ := NewRSAVerifierFromPEM(kp.PublicPEM)

	payload := []byte("test message")
	sig, err := signer.Sign(bg, payload)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if err := verifier.Verify(bg, payload, sig); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}
