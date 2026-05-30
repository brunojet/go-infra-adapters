package aws

//go:generate mockgen -destination=mock/mock_cloudfrontclient.go -package=mock . CloudFrontClient

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cfTypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"

	"github.com/brunojet/go-infra-adapters/pkg/cdn/contracts"
)

type cdnAdapter struct {
	client      CloudFrontClient
	maxKeys     int
	concurrency int
	logger      *slog.Logger
}

// NewCdn constructs a CloudFrontDistribution.
func NewCdn(opts ...CdnOption) *cdnAdapter {
	cfg := newCdnConfig(opts...)
	client := newCdnClient(cfg)
	d := &cdnAdapter{
		client:      client,
		maxKeys:     cfg.maxKeys,
		concurrency: cfg.concurrency,
		logger:      cfg.logger,
	}
	return d
}

// CreatePublicKey uploads the PEM-encoded public key and returns the newly created key's ID.
func (d *cdnAdapter) CreatePublicKey(ctx context.Context, key contracts.CdnKey) (string, error) {
	input := &cloudfront.CreatePublicKeyInput{
		PublicKeyConfig: &cfTypes.PublicKeyConfig{
			CallerReference: aws.String(fmt.Sprintf("%s-%d", key.Name, time.Now().UnixNano())),
			Name:            aws.String(key.Name),
			EncodedKey:      aws.String(key.PEM),
		},
	}
	out, err := d.client.CreatePublicKey(ctx, input)
	if err != nil {
		return "", fmt.Errorf("create public key: %w", err)
	}
	id := aws.ToString(out.PublicKey.Id)
	d.logger.Info("CloudFront public key created", "id", id, "name", key.Name)
	return id, nil
}

// EnsureKeyGroup guarantees a KeyGroup named name exists and contains keyID.
// Creates the group when absent, updates it otherwise. Returns the KeyGroup ID.
func (d *cdnAdapter) EnsureKeyGroup(ctx context.Context, name, keyID string) (string, error) {
	kg, err := d.findKeyGroupByName(ctx, name)
	if err != nil {
		return "", err
	}
	if kg == nil {
		d.logger.Info("KeyGroup not found, creating", "name", name)
		return d.createKeyGroup(ctx, name, keyID)
	}
	kgID := aws.ToString(kg.KeyGroup.Id)
	d.logger.Info("KeyGroup found", "id", kgID, "name", name)
	if err := d.updateKeyGroup(ctx, kgID, keyID); err != nil {
		return "", err
	}
	return kgID, nil
}

// VerifyKeyInGroup reports whether the public key described by key exists
// in the KeyGroup identified by key.GroupName.
func (d *cdnAdapter) VerifyKeyInGroup(ctx context.Context, key contracts.CdnKey) (bool, error) {
	id, err := d.findPublicKeyIDByName(ctx, key.GroupName, key.Name)
	return id != "", err
}

// HealthCheck performs a lightweight ListPublicKeys call to confirm
// the CloudFront API is reachable with the current credentials.
func (d *cdnAdapter) HealthCheck(ctx context.Context) error {
	if _, err := d.client.ListPublicKeys(ctx, &cloudfront.ListPublicKeysInput{MaxItems: aws.Int32(1)}); err != nil {
		return fmt.Errorf("cloudfront connectivity check: %w", err)
	}
	d.logger.Info("CloudFront connectivity confirmed")
	return nil
}
