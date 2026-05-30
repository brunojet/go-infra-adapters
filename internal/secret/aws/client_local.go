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

func (s *SecretsService[T]) movePendingStage(ctx context.Context, versionID string) error {
	// Find previous version with AWSPENDING to remove it before moving
	oldPending, err := s.getVersionWithStage(ctx, "AWSPENDING")
	if err != nil {
		return fmt.Errorf("find current AWSPENDING: %w", err)
	}

	// Idempotent: if versionID already has AWSPENDING, nothing to do
	if oldPending == versionID {
		return nil
	}

	in := &secretsmanager.UpdateSecretVersionStageInput{
		SecretId:        aws.String(s.name),
		VersionStage:    aws.String("AWSPENDING"),
		MoveToVersionId: aws.String(versionID),
	}
	if oldPending != "" {
		in.RemoveFromVersionId = aws.String(oldPending)
	}
	if _, err := s.client.UpdateSecretVersionStage(ctx, in); err != nil {
		return fmt.Errorf("move AWSPENDING stage: %w", err)
	}
	return nil
}
