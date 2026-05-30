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
	in := &secretsmanager.UpdateSecretVersionStageInput{
		SecretId:        aws.String(s.name),
		VersionStage:    aws.String("AWSPENDING"),
		MoveToVersionId: aws.String(versionID),
	}
	if _, err := s.client.UpdateSecretVersionStage(ctx, in); err != nil {
		return fmt.Errorf("move AWSPENDING stage: %w", err)
	}
	return nil
}
