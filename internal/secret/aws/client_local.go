package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func (s *SecretsService[T]) getVersionWithStage(ctx context.Context, stage string) (string, error) {
	out, err := s.client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(s.name),
	})
	if err != nil {
		return "", fmt.Errorf("describe secret: %w", err)
	}
	for versionID, stages := range out.VersionIdsToStages {
		for _, stg := range stages {
			if stg == stage {
				return versionID, nil
			}
		}
	}
	return "", nil
}

// moveStage moves a versioning stage from one version to another.
// stage: "AWSPENDING" or "AWSCURRENT"
// Idempotent: returns nil if toVersionID already has the stage.
func (s *SecretsService[T]) moveStage(ctx context.Context, stage, toVersionID string) error {
	oldVersion, err := s.getVersionWithStage(ctx, stage)
	if err != nil {
		return fmt.Errorf("find current %s: %w", stage, err)
	}

	// Idempotent: if toVersionID already has stage, nothing to do
	if oldVersion == toVersionID {
		return nil
	}

	in := &secretsmanager.UpdateSecretVersionStageInput{
		SecretId:        aws.String(s.name),
		VersionStage:    aws.String(stage),
		MoveToVersionId: aws.String(toVersionID),
	}
	if oldVersion != "" {
		in.RemoveFromVersionId = aws.String(oldVersion)
	}
	if _, err := s.client.UpdateSecretVersionStage(ctx, in); err != nil {
		return fmt.Errorf("move %s stage: %w", stage, err)
	}
	return nil
}

// movePendingStage moves AWSPENDING stage to versionID.
// Convenience wrapper around moveStage.
func (s *SecretsService[T]) movePendingStage(ctx context.Context, versionID string) error {
	return s.moveStage(ctx, "AWSPENDING", versionID)
}
