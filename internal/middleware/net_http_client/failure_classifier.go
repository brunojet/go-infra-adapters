package net_http_client

import (
	"errors"
	"net/http"
)

// FailureClassifier determines if an error should increment the circuit breaker.
// Returns nil if the response should NOT increment the breaker.
// Returns an error if it SHOULD increment the breaker.
type FailureClassifier interface {
	ClassifyError(resp *http.Response) error
}

type DefaultFailureClassifier struct{}

var errServerFailure = errors.New("server failure")

// ClassifyError returns an error only if this is a real server failure.
// Returns nil for 2xx-4xx responses.
func (c *DefaultFailureClassifier) ClassifyError(resp *http.Response) error {
	if resp == nil {
		return nil
	}

	switch {
	case resp.StatusCode >= http.StatusInternalServerError:
		return errServerFailure
	case resp.StatusCode == http.StatusTooManyRequests:
		return errServerFailure
	}

	return nil
}
