package s3

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// deleteObjectSafe deletes an object, ignoring NoSuchKey errors (idempotent).
// If eTag is provided, uses conditional delete (IfMatch) for atomicity.
func (b *bucketAdapter) deleteObjectSafe(ctx context.Context, key, eTag string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	}

	if eTag != "" {
		input.IfMatch = aws.String(eTag)
	}

	_, err := b.client.DeleteObject(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey" {
			return nil // Object not found is OK (idempotent)
		}
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}
