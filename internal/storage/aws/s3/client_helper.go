package s3

import (
	"errors"
	"fmt"
	"time"

	"github.com/aws/smithy-go"
	"github.com/brunojet/go-infra-adapters/v4/pkg/storage/contracts"
)

func fillObjectInfoUnsafe(key string, contentLength *int64, contentType, etag *string, objInfo *contracts.ObjectInfo) {
	objInfo.Key = key
	if contentLength != nil {
		objInfo.Size = *contentLength
	}
	if contentType != nil {
		objInfo.ContentType = *contentType
	}
	if etag != nil {
		objInfo.Metadata = map[string]string{etagKey: *etag}
	}
}

type lockExistsError struct {
	key string
}

func (e *lockExistsError) Error() string {
	return fmt.Sprintf("lock already exists for key %s", e.key)
}

func lockIsExpired(expiredStr string) bool {
	if expiredStr == "" {
		return false
	}
	expiredTime, err := time.Parse(time.RFC3339, expiredStr)
	if err != nil {
		return false
	}
	return time.Now().After(expiredTime)
}

func isIfNoneMatchError(err error) bool {
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == preconditionFailedCode
}
