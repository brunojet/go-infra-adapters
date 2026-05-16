package aws

import "context"

// CloudFrontAPI defines the operations the internal helpers need from CloudFront.
type CloudFrontAPI interface {
	CreatePublicKey(ctx context.Context, name string, publicKeyPem string) (publicKeyID string, err error)
	CreateKeyGroup(ctx context.Context, name string, publicKeyIDs []string) (keyGroupID string, err error)
}
