package contracts

import (
	"context"
	"testing"
)

func TestKeyPair(t *testing.T) {
	kp := &KeyPair{
		PrivatePEM:  []byte("private"),
		PublicPEM:   []byte("public"),
		Fingerprint: "abc123",
	}

	if len(kp.PrivatePEM) == 0 {
		t.Fatal("expected non-empty PrivatePEM")
	}
	if len(kp.PublicPEM) == 0 {
		t.Fatal("expected non-empty PublicPEM")
	}
	if kp.Fingerprint != "abc123" {
		t.Fatalf("expected fingerprint abc123, got %s", kp.Fingerprint)
	}
}

func TestKeyGenerator_Interface(t *testing.T) {
	var _ KeyGenerator = mockKeyGenerator{}
}

func TestSigner_Interface(t *testing.T) {
	var _ Signer = mockSigner{}
}

func TestVerifier_Interface(t *testing.T) {
	var _ Verifier = mockVerifier{}
}

type mockKeyGenerator struct{}

func (m mockKeyGenerator) Generate(context.Context) (*KeyPair, error) {
	return &KeyPair{}, nil
}

type mockSigner struct{}

func (m mockSigner) Sign(context.Context, []byte) ([]byte, error) {
	return nil, nil
}

type mockVerifier struct{}

func (m mockVerifier) Verify(context.Context, []byte, []byte) error {
	return nil
}
