// Package contracts defines lightweight interfaces used by higher-level
// packages to interact with HTTP clients without depending on concrete
// implementations.
package contracts

import (
	"context"
	"net/http"
)

// HttpClient is the minimal contract for performing HTTP requests in the
// repository. Implementations should return the underlying *http.Response
// which callers are responsible for closing.
type HttpClient interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}
