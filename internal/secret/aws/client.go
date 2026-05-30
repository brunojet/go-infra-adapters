package aws

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// ── SecretAPI ─────────────────────────────────────────────────────────────────

// SecretAPI holds a single AWS Secrets Manager client shared across all secrets
// created from this API instance. This avoids creating one SDK client per secret.
type SecretAPI struct {
	client SecretsManagerClient
	logger *slog.Logger
}

// NewSecretAPI constructs a SecretAPI using the provided options.
// Returns an error if no client is injected and the AWS SDK cannot initialize.
func NewSecretAPI(opts ...SecretsOption) (*SecretAPI, error) {
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, err
	}
	return &SecretAPI{client: cfg.client, logger: cfg.logger}, nil
}

// NewSecrets creates a SecretsService[T] for the named secret, reusing the
// client held by api. T must be JSON-serialisable.
func NewSecrets[T any](api *SecretAPI, name string) *SecretsService[T] {
	return &SecretsService[T]{
		client: api.client,
		logger: api.logger,
		name:   name,
	}
}

// ── SecretsService ────────────────────────────────────────────────────────────

// SecretsService[T] implements SecretStore[T] using AWS Secrets Manager.
// T is the application-specific payload type stored as JSON — no secrets-manager
// coupling leaks into the caller.
type SecretsService[T any] struct {
	client SecretsManagerClient
	logger *slog.Logger
	name   string
}

// Name returns the secret name.
func (s *SecretsService[T]) Name() string {
	return s.name
}

// GetCurrent retrieves and deserialises the current secret value into T.
// Returns a zero-value T when the secret exists but has no value yet (new secret).
func (s *SecretsService[T]) GetCurrent(ctx context.Context) (*T, error) {
	return s.GetVersion(ctx, "")
}

// GetVersion retrieves a specific version by provider-specific id.
func (s *SecretsService[T]) GetVersion(ctx context.Context, version string) (*T, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(s.name),
	}
	if version != "" {
		input.VersionId = aws.String(version)
	}
	out, err := s.client.GetSecretValue(ctx, input)
	if err != nil {
		if isNotFound(err) {
			return new(T), nil
		}
		return nil, fmt.Errorf("get version %s: %w", version, err)
	}
	if out.SecretString == nil {
		return new(T), nil
	}
	s.logger.Info("secret version retrieved", "version", version, "secret", s.name)
	return unmarshal[T](*out.SecretString)
}

// SetVersion serializes payload, writes it as a new version, moves AWSPENDING
// to it and returns the VersionId assigned by AWS.
// Pass version="" to let AWS generate the VersionId.
func (s *SecretsService[T]) SetVersion(ctx context.Context, payload *T, version string) (string, error) {
	b, err := marshal(payload)
	if err != nil {
		return "", err
	}
	input := &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(s.name),
		SecretString: aws.String(string(b)),
	}
	if version != "" {
		input.ClientRequestToken = aws.String(version)
	}
	out, err := s.client.PutSecretValue(ctx, input)
	if err != nil {
		return "", fmt.Errorf("put secret value: %w", err)
	}
	versionID := aws.ToString(out.VersionId)
	s.logger.Info("secret pending version written", "version", versionID, "secret", s.name)
	if err := s.movePendingStage(ctx, versionID); err != nil {
		return "", err
	}
	return versionID, nil
}

// PromoteVersion moves the secret identified by version from AWSPENDING to AWSCURRENT.
// Idempotent: if version is already AWSCURRENT, only the AWSPENDING stage is removed.
func (s *SecretsService[T]) PromoteVersion(ctx context.Context, version string) error {
	// Check if already AWSCURRENT to handle the idempotent case
	currentVersion, err := s.getVersionWithStage(ctx, "AWSCURRENT")
	if err != nil {
		return fmt.Errorf("find current version: %w", err)
	}
	if currentVersion == version {
		s.logger.Info("version already AWSCURRENT, removing AWSPENDING", "version", version, "secret", s.name)
		return s.DiscardVersion(ctx, version)
	}

	// Move AWSCURRENT to the new version (handles removing from old version)
	in := &secretsmanager.UpdateSecretVersionStageInput{
		SecretId:        aws.String(s.name),
		VersionStage:    aws.String("AWSCURRENT"),
		MoveToVersionId: aws.String(version),
	}
	if currentVersion != "" {
		in.RemoveFromVersionId = aws.String(currentVersion)
	}
	if _, err := s.client.UpdateSecretVersionStage(ctx, in); err != nil {
		return fmt.Errorf("promote version to AWSCURRENT: %w", err)
	}
	s.logger.Info("version promoted to AWSCURRENT", "version", version, "secret", s.name)
	return nil
}

// DiscardVersion removes the AWSPENDING stage from the given version.
func (s *SecretsService[T]) DiscardVersion(ctx context.Context, version string) error {
	in := &secretsmanager.UpdateSecretVersionStageInput{
		SecretId:            aws.String(s.name),
		VersionStage:        aws.String("AWSPENDING"),
		RemoveFromVersionId: aws.String(version),
	}
	if _, err := s.client.UpdateSecretVersionStage(ctx, in); err != nil {
		return fmt.Errorf("discard AWSPENDING from version %s: %w", version, err)
	}
	return nil
}

// HealthCheck confirms the secret resource exists and the AWS credentials are valid.
func (s *SecretsService[T]) HealthCheck(ctx context.Context) error {
	_, err := s.client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(s.name),
	})
	if err != nil {
		if isNotFound(err) {
			return fmt.Errorf("secret %q not found; create it via Terraform before invoking the service", s.name)
		}
		return fmt.Errorf("secrets manager connectivity check: %w", err)
	}
	s.logger.Info("Secrets Manager connectivity confirmed", "secret", s.name)
	return nil
}

// helpers are implemented in client_helper.go and client_local.go
