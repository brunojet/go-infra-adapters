package aws

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	smTypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// isNotFound reports whether err represents a "resource not found" response
// from Secrets Manager. Typed SDK errors are checked first; the string
// fallback covers test mocks and future SDK changes.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var smNF *smTypes.ResourceNotFoundException
	if errors.As(err, &smNF) {
		return true
	}
	// Fallback for non-typed or wrapped errors (e.g. test mocks, future SDK changes).
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "resourcenotfoundexception")
}

func marshal[T any](v *T) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return b, nil
}

func unmarshal[T any](s string) (*T, error) {
	var v T
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	return &v, nil
}
