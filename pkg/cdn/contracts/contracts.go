// Package contracts defines the CDN distribution contracts.
package contracts

import "context"

// CdnKey holds only the data a CDN key distribution adapter needs.
// It contains no private key material.
type CdnKey struct {
	Name      string // key name in the CDN (e.g. "myapp-abc12345")
	PEM       string // PEM-encoded public key
	GroupName string // key group to associate with
}

// CdnAdapter abstracts the CDN key distribution layer (CloudFront, Fastly, Akamai, etc).
// Implementations own the SDK translation — callers pass only domain-level CdnKey.
type CdnAdapter interface {
	// CreatePublicKey uploads the PEM-encoded public key and returns the newly created key's ID.
	CreatePublicKey(ctx context.Context, key CdnKey) (keyID string, err error)

	// EnsureKeyGroup guarantees a KeyGroup named name exists and contains keyID.
	// Creates the group when absent, updates it otherwise.
	// Returns the KeyGroup ID.
	EnsureKeyGroup(ctx context.Context, name, keyID string) (groupID string, err error)

	// VerifyKeyInGroup reports whether the public key described by key exists
	// in the KeyGroup identified by key.GroupName.
	VerifyKeyInGroup(ctx context.Context, key CdnKey) (bool, error)

	// HealthCheck verifies connectivity to the CDN provider.
	HealthCheck(ctx context.Context) error
}
